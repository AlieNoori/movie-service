package main

import (
	"context"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v3"
	"movieexample.com/auth/internal/controller/auth"
	grpchandler "movieexample.com/auth/internal/handler/grpc"
	"movieexample.com/auth/internal/repository/mysql"
	"movieexample.com/auth/migrations"
	"movieexample.com/gen"
	"movieexample.com/pkg/discovery"
	"movieexample.com/pkg/discovery/consul"
	"movieexample.com/pkg/tracing"
)

const (
	SECRET      = "QRLoXR6XCc6lc2kunrO2QEJMx1v3RjR7aesHCZGSyPs="
	serviceName = "auth"
)

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

	logger.Info("Starting the rating service", zap.Int("port", port))

	ctx, cancel := context.WithCancel(context.Background())

	tp, err := tracing.NewJaegerProvider(cfg.Jaeger.URL, serviceName)
	if err != nil {
		logger.Fatal("Failed to initialize Jaeger provider", zap.Error(err))
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Fatal("Failed to shut down Jaeger prodiver", zap.Error(err))
		}
	}()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	registry, err := consul.NewRegistry(cfg.ServiceDiscovery.Consul.Address)
	if err != nil {
		panic(err)
	}

	instanceID := discovery.GenerateInstanceID(serviceName)
	if err := registry.Register(ctx, instanceID, serviceName, fmt.Sprintf("auth:%d", port)); err != nil {
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

	defer registry.Deregister(ctx, instanceID, serviceName)

	repo, err := mysql.NewWithMigration("root:password@/movieexample", migrations.FS, ".")
	if err != nil {
		panic(err)
	}
	defer repo.Close()

	controller := auth.New(repo, func() []byte { return []byte(SECRET) })
	h := grpchandler.New(controller)

	creds, err := credentials.NewServerTLSFromFile("configs/server.crt", "configs/server.key")
	if err != nil {
		logger.Error("Failed to load key pair", zap.Error(err))
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		logger.Error("Failed to listen key pair", zap.Error(err))
	}

	srv := grpc.NewServer(grpc.Creds(creds), grpc.StatsHandler(otelgrpc.NewServerHandler()))
	reflection.Register(srv)
	gen.RegisterAuthServiceServer(srv, h)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)

	wg.Go(func() {
		defer wg.Done()
		s := <-sigChan
		cancel()
		logger.Info("Attempting graceful shutdown", zap.Stringer("signal", s))
		srv.GracefulStop()
		logger.Info("Gracefully stopped the gRPC server")
	})

	if err := srv.Serve(lis); err != nil {
		logger.Fatal("Failed to start the gRPC server", zap.Error(err))
	}

	wg.Done()
}
