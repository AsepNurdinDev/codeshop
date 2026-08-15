package handler

import (
	"context"
	"database/sql"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type HealthHandler struct {
	grpc_health_v1.UnimplementedHealthServer
	db  *sql.DB
	rdb *redis.Client
}

func NewHealthHandler(db *sql.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:  db,
		rdb: rdb,
	}
}

func (h *HealthHandler) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	if h.db != nil {
		if err := h.db.PingContext(ctx); err != nil {
			return &grpc_health_v1.HealthCheckResponse{
				Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
			}, nil
		}
	}

	if h.rdb != nil {
		if err := h.rdb.Ping(ctx).Err(); err != nil {
			return &grpc_health_v1.HealthCheckResponse{
				Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
			}, nil
		}
	}

	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

func (h *HealthHandler) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "health watch is not implemented")
}
