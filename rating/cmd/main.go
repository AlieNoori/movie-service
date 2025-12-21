package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v3"
	"movieexample.com/gen"
	"movieexample.com/pkg/discovery"
	"movieexample.com/pkg/discovery/consul"
	"movieexample.com/rating/internal/controller/rating"
	grpchandler "movieexample.com/rating/internal/handler/grpc"
	"movieexample.com/rating/internal/ingester/kafka"
	"movieexample.com/rating/internal/repository/mysql"
	"movieexample.com/rating/migrations"
)

const serviceName = "rating"

func main() {
	f, err := os.Open("configs/default.yaml")
	if err != nil {
		panic(err)
	}

	var cfg config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		panic(err)
	}

	port := cfg.API.Port
	log.Printf("Starting the rating service on port %d", port)

	registry, err := consul.NewRegistry(cfg.ServiceDiscovery.Consul.Address)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	instanceID := discovery.GenerateInstanceID(serviceName)
	if err := registry.Register(ctx, instanceID, serviceName, fmt.Sprintf("rating:%d", port)); err != nil {
		panic(err)
	}

	go func() {
		for {
			if err := registry.ReportHealthyState(instanceID, serviceName); err != nil {
				log.Println("Failed to report healthy state: " + err.Error())
			}
			time.Sleep(1 * time.Second)
		}
	}()

	defer registry.Deregister(ctx, instanceID, serviceName)

	// repo, err := mysql.New("root:password@/movieexample")
	repo, err := mysql.NewWithMigration("root:password@/movieexample", migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	ingester, err := kafka.NewIngester(cfg.Kafka.Address, cfg.Kafka.GroupID, cfg.Kafka.Topic)
	if err != nil {
		log.Fatalf("failed to initialize ingester: %v", err)
	}

	ctrl := rating.New(repo, ingester)

	go func() {
		if err := ctrl.StartIngestion(ctx); err != nil {
			log.Printf("failed to start ingestion: %v", err)
		}
	}()

	h := grpchandler.New(ctrl)

	creds, err := credentials.NewServerTLSFromFile("configs/server.crt", "configs/server.key")
	if err != nil {
		log.Fatalf("Failed to load key pair: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer(grpc.Creds(creds))
	reflection.Register(srv)
	gen.RegisterRatingServiceServer(srv, h)

	if err := srv.Serve(lis); err != nil {
		panic(err)
	}
}
