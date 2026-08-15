package handler

import (
	"context"
	"errors"

	"github.com/asepnrdn/codeshop/services/auth-service/internal/domain"
	"github.com/asepnrdn/codeshop/services/auth-service/internal/service"
	pb "github.com/asepnrdn/codeshop/services/auth-service/proto/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCHandler struct {
	pb.UnimplementedAuthServiceServer
	authService service.AuthService
}

func NewGRPCHandler(authService service.AuthService) *GRPCHandler {
	return &GRPCHandler{
		authService: authService,
	}
}

func (h *GRPCHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	user, err := h.authService.Register(ctx, req.GetEmail(), req.GetPassword(), req.GetFullName())
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.RegisterResponse{
		UserId:    user.ID.String(),
		Email:     user.Email,
		FullName:  user.FullName,
		Role:      string(user.Role),
		CreatedAt: timestamppb.New(user.CreatedAt),
	}, nil
}

func (h *GRPCHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	tokens, err := h.authService.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    tokens.TokenType,
		UserId:       tokens.UserID.String(),
	}, nil
}

func (h *GRPCHandler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	claims, err := h.authService.ValidateToken(ctx, req.GetAccessToken())
	if err != nil {
		return &pb.ValidateTokenResponse{Valid: false}, nil
	}

	expTime := claims.ExpiresAt.Time
	return &pb.ValidateTokenResponse{
		Valid:     true,
		UserId:    claims.UserID,
		Email:     claims.Email,
		Role:      claims.Role,
		ExpiresAt: timestamppb.New(expTime),
	}, nil
}

func (h *GRPCHandler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	tokens, err := h.authService.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.RefreshTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
		TokenType:    tokens.TokenType,
	}, nil
}

func (h *GRPCHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	var uid uuid.UUID
	if req.GetUserId() != "" {
		parsedUID, err := uuid.Parse(req.GetUserId())
		if err == nil {
			uid = parsedUID
		}
	}

	err := h.authService.Logout(ctx, req.GetRefreshToken(), uid)
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.LogoutResponse{
		Success: true,
		Message: "Logged out successfully",
	}, nil
}

func (h *GRPCHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	uid, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id format")
	}

	user, err := h.authService.GetUser(ctx, uid)
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.GetUserResponse{
		UserId:    user.ID.String(),
		Email:     user.Email,
		FullName:  user.FullName,
		Role:      string(user.Role),
		Status:    string(user.Status),
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}, nil
}

func mapDomainErrorToGRPC(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrAccountSuspended):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidToken), errors.Is(err, domain.ErrExpiredToken), errors.Is(err, domain.ErrTokenRevoked):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, "an internal server error occurred")
	}
}
