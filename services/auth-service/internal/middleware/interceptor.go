package middleware

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

// UnaryLoggingInterceptor logs incoming RPC requests safely without leaking sensitive data like passwords or tokens.
func UnaryLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		requestID := getOrGenerateRequestID(ctx)
		ctx = context.WithValue(ctx, RequestIDKey, requestID)

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		st, _ := status.FromError(err)

		log.Printf(
			"[gRPC] req_id=%s method=%s status=%s duration=%v err=%v",
			requestID,
			info.FullMethod,
			st.Code(),
			duration,
			err,
		)

		return resp, err
	}
}

// UnaryRecoveryInterceptor catches panics in RPC handlers and returns an Internal status code.
func UnaryRecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC RECOVERY] method=%s panic=%v", info.FullMethod, r)
				err = status.Errorf(codes.Internal, "internal server panic occurred")
			}
		}()
		return handler(ctx, req)
	}
}

func getOrGenerateRequestID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if reqIDs := md.Get("x-request-id"); len(reqIDs) > 0 && reqIDs[0] != "" {
			return reqIDs[0]
		}
	}
	return fmt.Sprintf("req-%s", uuid.New().String()[:8])
}
