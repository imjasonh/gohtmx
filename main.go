package main

import (
	"flag"
	"gohtmx/internal/app"
	"log/slog"
	"net/http"
	"os"

	"cloud.google.com/go/compute/metadata"
	"github.com/chainguard-dev/clog/gcp"
)

var port = flag.String("port", "8080", "Port to run the server on")

func init() {
	level := slog.LevelInfo
	if l := os.Getenv("LOG_LEVEL"); l != "" {
		if err := level.UnmarshalText([]byte(l)); err != nil {
			slog.Error("Failed to parse LOG_LEVEL, defaulting to INFO", "got", l, "error", err)
		}
	}
	if metadata.OnGCE() {
		slog.SetDefault(slog.New(gcp.NewHandler(level)))
	} else {
		slog.SetDefault(slog.New(app.NewColorHandler(os.Stderr, level)))
	}
}

func main() {
	flag.Parse()

	// Create the application
	application := app.New()

	// Set up routes
	mux := http.NewServeMux()
	application.SetupRoutes(mux)

	// Start server
	slog.Info("Starting server", "port", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		slog.Error("Server failed to start", "error", err)
	}
}
