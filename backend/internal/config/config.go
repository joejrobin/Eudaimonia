// Package config reads node configuration from the environment.
//
// Every setting has a development default so `go run ./cmd/server` works with no
// setup; production deployments override them (see compose.yaml and .env.example).
package config

import (
	"log/slog"
	"os"
	"strings"
)

// Config is the fully resolved configuration for one node process.
type Config struct {
	// Addr is the host:port the HTTP/WebSocket listener binds to.
	Addr string
	// NodeName is a human-readable label for this node, shown to clients.
	NodeName string
	// DatabaseURL is a PostgreSQL connection string.
	DatabaseURL string
	// LiveKitURL is the WebSocket URL clients use to reach this node's SFU.
	LiveKitURL string
	// LiveKitAPIKey and LiveKitAPISecret sign the room tokens the node mints.
	LiveKitAPIKey    string
	LiveKitAPISecret string
	// LogLevel controls slog verbosity.
	LogLevel slog.Level
}

// FromEnv builds a Config from environment variables.
func FromEnv() Config {
	return Config{
		Addr:             getenv("BACKEND_ADDR", ":8080"),
		NodeName:         getenv("NODE_NAME", "dev-node"),
		DatabaseURL:      getenv("DATABASE_URL", "postgres://eudaimonia:eudaimonia@127.0.0.1:5432/eudaimonia?sslmode=disable"),
		LiveKitURL:       getenv("LIVEKIT_URL", "ws://127.0.0.1:7880"),
		LiveKitAPIKey:    getenv("LIVEKIT_API_KEY", "devkey"),
		LiveKitAPISecret: getenv("LIVEKIT_API_SECRET", "secret"),
		LogLevel:         parseLevel(getenv("LOG_LEVEL", "info")),
	}
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
