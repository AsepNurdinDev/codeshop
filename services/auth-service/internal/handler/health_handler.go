package handler

import (
	"context"
	"github.com/codeshop-platform/auth-service/internal/webutil"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// HealthHandler exposes liveness/readiness endpoints for container
// orchestration (Docker healthcheck today, Kubernetes probes later —
// docs/architecture/ARCHITECTURE.md §18).
type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redisClient}
}

// Live is a liveness probe: the process is up and can serve HTTP. It never
// checks dependencies, so a slow database never causes Kubernetes to kill
// a healthy pod.
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	webutil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready is a readiness probe: the service can actually serve traffic,
// meaning its critical dependencies (Postgres, Redis) are reachable.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]string{}
	ready := true

	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			checks["database"] = "unreachable"
			ready = false
		} else {
			checks["database"] = "ok"
		}
	}

	if h.redis != nil {
		if err := h.redis.Ping(ctx).Err(); err != nil {
			checks["redis"] = "unreachable"
			ready = false
		} else {
			checks["redis"] = "ok"
		}
	}

	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	webutil.WriteJSON(w, status, map[string]any{"status": readyLabel(ready), "checks": checks})
}

func readyLabel(ready bool) string {
	if ready {
		return "ready"
	}
	return "not_ready"
}
