// Package grpcserver implements the internal gRPC contract defined in
// proto/auth.proto: ValidateToken and GetUser, consumed by other CodeShop
// services (API Gateway, Order Service, Download Service) per
// docs/architecture/SERVICE_CATALOG.md §2.
package grpcserver

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/codeshop-platform/auth-service/internal/domain"
	"github.com/codeshop-platform/auth-service/internal/service"
	"github.com/codeshop-platform/auth-service/pkg/pb"
)

// Server implements pb.AuthServiceServer.
type Server struct {
	pb.UnimplementedAuthServiceServer
	auth service.AuthService
}

// New returns a gRPC AuthService server backed by the given AuthService.
func New(auth service.AuthService) *Server {
	return &Server{auth: auth}
}

// ValidateToken verifies a JWT access token statelessly (signature +
// expiry) so callers do not need Auth Service to be highly available for
// every single request, per docs/architecture/ARCHITECTURE.md §12.
func (s *Server) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if req.GetAccessToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "access_token is required")
	}

	claims, err := s.auth.ValidateAccessToken(ctx, req.GetAccessToken())
	if err != nil {
		reason := "invalid"
		if errors.Is(err, domain.ErrTokenExpired) {
			reason = "expired"
		}
		return &pb.ValidateTokenResponse{
			Valid:  false,
			Reason: reason,
		}, nil
	}

	return &pb.ValidateTokenResponse{
		Valid:  true,
		UserId: claims.UserID.String(),
		Email:  claims.Email,
		Role:   string(claims.Role),
	}, nil
}

// GetUser returns identity information for a user by ID. Used by services
// that only hold a user_id cross-service reference
// (docs/architecture/DATABASE_DESIGN.md §8).
func (s *Server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	userID, err := parseUUID(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user_id must be a valid UUID")
	}

	user, err := s.auth.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to load user")
	}

	return &pb.GetUserResponse{
		UserId:   user.ID.String(),
		Email:    user.Email,
		FullName: user.FullName,
		Role:     string(user.Role),
		Status:   string(user.Status),
	}, nil
}
