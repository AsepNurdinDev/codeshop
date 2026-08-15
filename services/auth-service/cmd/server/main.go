package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asepnrdn/codeshop/services/auth-service/internal/config"
	"github.com/asepnrdn/codeshop/services/auth-service/internal/handler"
	"github.com/asepnrdn/codeshop/services/auth-service/internal/middleware"
	"github.com/asepnrdn/codeshop/services/auth-service/internal/repository"
	"github.com/asepnrdn/codeshop/services/auth-service/internal/security"
	"github.com/asepnrdn/codeshop/services/auth-service/internal/service"
	pb "github.com/asepnrdn/codeshop/services/auth-service/proto/v1"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()
	log.Printf("Starting CodeShop Auth Service [%s]...", cfg.Environment)

	// 1. Initialize Database
	db, err := initDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// 2. Initialize Redis (optional connection, logs warning if fails)
	rdb := initRedis(cfg.RedisURL)
	if rdb != nil {
		defer rdb.Close()
	}

	// 3. Initialize Security & Managers
	jwtMgr := security.NewJWTManager(
		cfg.JWTSecret,
		cfg.JWTIssuer,
		cfg.AccessTokenExpiration,
		cfg.RefreshTokenExpiration,
	)

	// 4. Initialize Repositories & Services
	userRepo := repository.NewPostgresUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db, rdb)
	authSvc := service.NewAuthService(userRepo, tokenRepo, jwtMgr)

	// 5. Initialize gRPC Server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.UnaryRecoveryInterceptor(),
			middleware.UnaryLoggingInterceptor(),
		),
	)

	// Register Auth Handler
	authHandler := handler.NewGRPCHandler(authSvc)
	pb.RegisterAuthServiceServer(grpcServer, authHandler)

	// Register Health Handler
	healthHandler := handler.NewHealthHandler(db, rdb)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthHandler)

	// Register Reflection for dev tools (grpcurl)
	reflection.Register(grpcServer)

	// 6. Listen & Serve
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	// Channel for graceful shutdown signals
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Auth Service gRPC server listening on port %s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-stopChan
	log.Println("Shutting down Auth Service gracefully...")

	// Graceful shutdown with 10s timeout
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("Auth Service stopped cleanly.")
	case <-time.After(10 * time.Second):
		log.Println("Force stopping Auth Service due to timeout.")
		grpcServer.Stop()
	}
}

func initDatabase(dbURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("error opening DB connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("error pinging DB: %w", err)
	}

	log.Println("Connected to PostgreSQL successfully.")
	return db, nil
}

func initRedis(redisURL string) *redis.Client {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("Warning: Redis URL invalid (%v). Proceeding without Redis cache.", err)
		return nil
	}

	rdb := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis (%v). Token revocation will rely solely on PostgreSQL.", err)
		return nil
	}

	log.Println("Connected to Redis successfully.")
	return rdb
}
