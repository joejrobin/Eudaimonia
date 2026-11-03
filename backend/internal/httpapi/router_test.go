package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthonychavezdev/eudaimonia/backend/internal/config"
)

func TestHealthz(t *testing.T) {
	h := NewRouter(config.FromEnv(), slog.Default())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("GET /healthz = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestNodeDescriptor(t *testing.T) {
	cfg := config.FromEnv()
	cfg.NodeName = "test-node"
	h := NewRouter(cfg, slog.Default())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/node", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/node = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["name"] != "test-node" {
		t.Fatalf("name = %v, want test-node", body["name"])
	}
}
