package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/uber-go/tally/v4"
	"github.com/uber-go/tally/v4/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v3"
	"movieexample.com/gen"
	"movieexample.com/metadata/internal/controller/metadata"
	grpchandler "movieexample.com/metadata/internal/handler/grpc"
	"movieexample.com/metadata/internal/repository/mysql"
	"movieexample.com/metadata/internal/repository/redis"
	"movieexample.com/metadata/migrations"
	"movieexample.com/pkg/discovery"
	"movieexample.com/pkg/discovery/consul"
	"movieexample.com/pkg/tracing"
)

const serviceName = "metadata"

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
	logger.Info("Starting the metadata service", zap.Int("port", port))

	ctx, cancel := context.WithCancel(context.Background())

	tp, err := tracing.NewJaegerProvider(cfg.Jaeger.URL, serviceName)
	if err != nil {
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	reporter := prometheus.NewReporter(prometheus.Options{Registerer: nil})
	scope, closer := tally.NewRootScope(tally.ScopeOptions{
		Prefix:         serviceName,
		Tags:           map[string]string{"env": "prod"},
		CachedReporter: reporter,
		Separator:      prometheus.DefaultSeparator,
		SanitizeOptions: &tally.SanitizeOptions{
			NameCharacters: tally.ValidCharacters{
				Ranges:     tally.AlphanumericRange,
				Characters: []rune{'_'},
			},
			KeyCharacters: tally.ValidCharacters{
				Ranges:     tally.AlphanumericRange,
				Characters: []rune{'_'},
			},
			ValueCharacters: tally.ValidCharacters{
				Ranges:     tally.AlphanumericRange,
				Characters: []rune{'_'},
			},
			ReplacementCharacter: '_',
		},
	}, 10*time.Second)
	defer closer.Close()

	startCounter := scope.Counter("service_starts")
	startCounter.Inc(1)

	http.Handle("/metrics", reporter.HTTPHandler())
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Prometheus.MetricsPort), nil); err != nil {
			logger.Fatal("Failed to start the metrics handler", zap.Error(err))
		}
	}()

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
			time.Sleep(2 * time.Second)
		}
	}()

	repo, err := mysql.NewWithMigration("root:password@/movieexample", migrations.FS, ".")
	if err != nil {
		panic(err)
	}
	defer repo.Close()

	cache := redis.New(repo)
	svc := metadata.New(cache)

	h := grpchandler.New(svc)

	creds, err := credentials.NewServerTLSFromFile("configs/server.crt", "configs/server.key")
	if err != nil {
		logger.Fatal("Failed to load key pair", zap.Error(err))
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("faild to listen: %v", err)
		logger.Fatal("Failed to listen", zap.Error(err))
	}
	srv := grpc.NewServer(grpc.Creds(creds), grpc.StatsHandler(otelgrpc.NewServerHandler()))
	reflection.Register(srv)
	gen.RegisterMetadataServiceServer(srv, h)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)

	wg.Go(func() {
		s := <-sigChan
		cancel()
		logger.Info("Received signal %v, attempting graceful shutdown", zap.Stringer("signal", s))
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
