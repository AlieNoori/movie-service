package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v3"
	"movieexample.com/gen"
	"movieexample.com/movie/internal/controller/movie"
	"movieexample.com/pkg/discovery"
	"movieexample.com/pkg/discovery/consul"
	"movieexample.com/pkg/tracing"

	"github.com/grpc-ecosystem/go-grpc-middleware/ratelimit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	metadatagateway "movieexample.com/movie/internal/gateway/metadata/grpc"
	ratinggateway "movieexample.com/movie/internal/gateway/rating/grpc"
	grpchandler "movieexample.com/movie/internal/handler/grpc"
)

const serviceName = "movie"

type limiter struct {
	l *rate.Limiter
}

func newLimiter(limit int, burst int) *limiter {
	return &limiter{rate.NewLimiter(rate.Limit(limit), burst)}
}

func (l *limiter) Limit() bool {
	return !l.l.Allow()
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	f, err := os.Open("configs/default.yaml")
	if err != nil {
		logger.Fatal("Failed to open configuration", zap.Error(err))
	}
	defer f.Close()

	var cfg config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		logger.Fatal("Failed to parse configuration", zap.Error(err))
	}

	port := cfg.API.Port

	logger.Info("Starting the movie service", zap.Int("port", port))

	ctx, cancel := context.WithCancel(context.Background())

	tp, err := tracing.NewJaegerProvider(cfg.Jaeger.URL, serviceName)
	if err != nil {
		logger.Fatal("Failed to initialize Jaeger provider", zap.Error(err))
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	registry, err := consul.NewRegistry(cfg.ServiceDiscovery.Consul.Address)
	if err != nil {
		panic(err)
	}

	instanceID := discovery.GenerateInstanceID(serviceName)
	if err := registry.Register(ctx, instanceID, serviceName, fmt.Sprintf("localhost:%d", port)); err != nil {
		panic(err)
	}

	go func() {
		for {
			if err := registry.ReportHealthyState(instanceID, serviceName); err != nil {
				logger.Error("Failed to report healthy state", zap.Error(err))
			}
			time.Sleep(1 * time.Second)
		}
	}()

	serverCert, err := tls.LoadX509KeyPair("configs/server.crt", "configs/server.key")
	if err != nil {
		logger.Fatal("Failed to load server key pair", zap.Error(err))
	}

	trustedCert, err := os.ReadFile("configs/server.crt")
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(trustedCert)

	serverCreds := credentials.NewServerTLSFromCert(&serverCert)

	clientCreds := credentials.NewTLS(&tls.Config{
		RootCAs: certPool,
	})

	metadataGateway := metadatagateway.New(registry, clientCreds)
	ratingGateway := ratinggateway.New(registry, clientCreds)

	ctrl := movie.New(ratingGateway, metadataGateway)
	h := grpchandler.New(ctrl)

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	const limit = 100
	const burst = 100
	l := newLimiter(limit, burst)
	srv := grpc.NewServer(
		grpc.Creds(serverCreds),
		grpc.UnaryInterceptor(ratelimit.UnaryServerInterceptor(l)),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	reflection.Register(srv)
	gen.RegisterMovieServiceServer(srv, h)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)

	wg.Go(func() {
		s := <-sigChan
		cancel()
		logger.Info("Received signal, attempting graceful shutdown", zap.Stringer("signal", s))
		srv.GracefulStop()
		logger.Info("Gracefully stopped the gRPC server")
		registry.Deregister(ctx, instanceID, serviceName)
		logger.Info("Deregister service from discovery")
		if err := tp.Shutdown(ctx); err != nil {
			logger.Fatal("Failed to shut down Jaeger prodiver", zap.Error(err))
		}
		wg.Done()
	})

	if err := srv.Serve(lis); err != nil {
		logger.Fatal("Failed to start the gRPC server", zap.Error(err))
	}
	wg.Wait()
}
