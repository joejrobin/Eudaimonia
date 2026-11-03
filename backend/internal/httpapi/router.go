// Package httpapi wires the node's HTTP and WebSocket endpoints.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/anthonychavezdev/eudaimonia/backend/internal/config"
)

// Version is overridden at build time with -ldflags "-X .../httpapi.Version=...".
var Version = "dev"

// NewRouter returns the node's root handler.
func NewRouter(cfg config.Config, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	// Liveness: the process is up.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Readiness: dependencies are reachable. Extend once the database is wired in.
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Public node descriptor, the first thing a client fetches when adding a node.
	mux.HandleFunc("GET /api/v1/node", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"name":        cfg.NodeName,
			"version":     Version,
			"livekit_url": cfg.LiveKitURL,
			"time":        time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Realtime channel between a client and this node. Placeholder until the
	// event protocol exists (see docs/RESEARCH.md section 4).
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "realtime endpoint not implemented yet", http.StatusNotImplemented)
	})

	return requestLogger(logger, mux)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// requestLogger logs one line per request at debug level.
func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Debug("request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
			"duration", time.Since(start),
		)
	})
}
