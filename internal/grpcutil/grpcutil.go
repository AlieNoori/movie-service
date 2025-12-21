package grpcutil

import (
	"context"
	"math/rand/v2"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"movieexample.com/pkg/discovery"
)

func ServiceConnection(ctx context.Context, serviceName string, registry discovery.Registry, creds credentials.TransportCredentials) (*grpc.ClientConn, error) {
	addrs, err := registry.ServiceAddresses(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	return grpc.NewClient(addrs[rand.IntN(len(addrs))], grpc.WithTransportCredentials(creds))
}
