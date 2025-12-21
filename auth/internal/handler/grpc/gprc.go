package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"movieexample.com/auth/internal/controller/auth"
	"movieexample.com/gen"
)

type Handler struct {
	gen.UnimplementedAuthServiceServer
	svc *auth.Controller
}

// New creates a new auth gRPC handler.
func New(svc *auth.Controller) *Handler {
	return &Handler{svc: svc}
}

// Register handles the user registration request.
func (h *Handler) Register(ctx context.Context, req *gen.RegisterRequest) (*gen.RegisterResponse, error) {
	err := h.svc.Register(ctx,
		req.Email,
		req.Username,
		req.Password,
		req.FirstName,
		req.LastName,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register user: %v", err)
	}

	return &gen.RegisterResponse{}, nil
}

// GetToken performs verification of user credentials and returns a JWT token in case of success.
func (h *Handler) GetToken(ctx context.Context, req *gen.GetTokenRequest) (*gen.GetTokenResponse, error) {
	token, err := h.svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Invalid email or password")
	}

	tokenString, err := h.svc.GetTokenString(token)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &gen.GetTokenResponse{Token: tokenString}, nil
}

// ValidateToken performs JWT token validation.
func (h *Handler) ValidateToken(ctx context.Context, req *gen.ValidateTokenRequest) (*gen.ValidateTokenResponse, error) {
	username, err := h.svc.IsValid(req.Token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token")
	}

	return &gen.ValidateTokenResponse{Username: username}, nil
}
