package middleware

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/codeshop-platform/auth-service/internal/webutil"
)

// HeaderRequestID is the header CodeShop uses end-to-end for single-request
// tracing, per docs/architecture/API_DESIGN.md §10.
const HeaderRequestID = "X-Request-ID"

// RequestID assigns an X-Request-ID to every request (reusing one supplied
// by an upstream caller/CDN/API Gateway if present, per API_DESIGN.md §10),
// stores it in the request context for handlers/logging, and echoes it back
// on the response.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderRequestID)
		if id == "" {
			id = uuid.NewString()
		}

		ctx := webutil.WithRequestID(r.Context(), id)
		w.Header().Set(HeaderRequestID, id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
