// Command server is the Auth Service entry point. It wires together the
// REST API (public, docs/architecture/API_DESIGN.md), the internal gRPC
// API (proto/auth.proto), PostgreSQL, Redis, and the RabbitMQ event
// publisher, and runs the HTTP and gRPC servers concurrently until an
// interrupt signal triggers a graceful shutdown.
package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/codeshop-platform/auth-service/internal/config"
	"github.com/codeshop-platform/auth-service/internal/event"
	"github.com/codeshop-platform/auth-service/internal/grpcserver"
	"github.com/codeshop-platform/auth-service/internal/handler"
	appmiddleware "github.com/codeshop-platform/auth-service/internal/middleware"
	"github.com/codeshop-platform/auth-service/internal/repository"
	"github.com/codeshop-platform/auth-service/internal/service"
	"github.com/codeshop-platform/auth-service/pkg/pb"
)

func main() {
	// .env is optional and only used for local development convenience;
	// in Docker/Kubernetes, real environment variables are used instead.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- PostgreSQL (Auth DB - docs/decisions/ADR-006-postgresql-database.md) ---
	dbPool, err := connectPostgres(ctx, cfg)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer dbPool.Close()

	// --- Redis (sessions/rate limiting - docs/decisions/ADR-007-redis.md) ---
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("connect redis: %v", err)
	}

	// --- RabbitMQ (publishes UserRegistered - docs/architecture/EVENT_DESIGN.md) ---
	// Non-fatal if unavailable: Auth Service's primary responsibilities
	// (register/login/validate) do not depend on the message broker, per
	// the failure-mode philosophy in docs/architecture/ARCHITECTURE.md §12.
	publisher, err := event.NewRabbitMQPublisher(cfg.RabbitMQURL, cfg.RabbitMQExchange)
	if err != nil {
		log.Printf("warn: rabbitmq unavailable, UserRegistered events will not be published: %v", err)
		publisher = nil
	} else {
		defer publisher.Close()
	}

	// --- wiring ---
	userRepo := repository.NewUserRepository(dbPool)
	tokenRepo := repository.NewTokenRepository(redisClient, dbPool)
	tokenSvc := service.NewTokenService(cfg)
	authSvc := service.NewAuthService(cfg, userRepo, tokenRepo, tokenSvc, publisher, redisClient)

	httpServer := buildHTTPServer(cfg, authSvc, tokenSvc, dbPool, redisClient)
	grpcServer := buildGRPCServer(authSvc)

	// --- run servers concurrently ---
	errCh := make(chan error, 2)

	go func() {
		log.Printf("auth-service HTTP server listening on :%s", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	grpcLis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("listen grpc: %v", err)
	}
	go func() {
		log.Printf("auth-service gRPC server listening on :%s", cfg.GRPCPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("shutdown signal received")
	case err := <-errCh:
		log.Printf("server error: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http graceful shutdown error: %v", err)
	}
	grpcServer.GracefulStop()

	log.Println("auth-service stopped")
}

func connectPostgres(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = cfg.DatabaseMaxConns
	poolCfg.ConnConfig.ConnectTimeout = cfg.DatabaseConnTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.DatabaseConnTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func buildHTTPServer(
	cfg *config.Config,
	authSvc service.AuthService,
	tokenSvc service.TokenService,
	dbPool *pgxpool.Pool,
	redisClient *redis.Client,
) *http.Server {
	r := chi.NewRouter()

	r.Use(appmiddleware.RequestID)
	r.Use(appmiddleware.Recovery)
	r.Use(cors.Handler(cors.Options{
		// Public REST is meant to sit behind the API Gateway / CDN
		// (docs/architecture/ARCHITECTURE.md §4-6); permissive CORS here
		// is acceptable for local/dev use and is expected to be tightened
		// at the Gateway/CDN layer in production.
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	healthHandler := handler.NewHealthHandler(dbPool, redisClient)
	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(authSvc)

	r.Get("/healthz", healthHandler.Live)
	r.Get("/readyz", healthHandler.Ready)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})

		r.Route("/users", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(appmiddleware.RequireAuth(tokenSvc))
				r.Get("/me", userHandler.Me)
			})
		})
	})

	return &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func buildGRPCServer(authSvc service.AuthService) *grpc.Server {
	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, grpcserver.New(authSvc))

	// Standard gRPC health-checking protocol, so Kubernetes/other services
	// can probe readiness the same way for every internal gRPC service.
	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(s, healthSrv)

	// Reflection makes local debugging with grpcurl straightforward,
	// consistent with the tooling note in
	// docs/decisions/ADR-003-grpc-internal-communication.md ("memerlukan
	// tooling tambahan (mis. grpcurl)").
	reflection.Register(s)

	return s
}
