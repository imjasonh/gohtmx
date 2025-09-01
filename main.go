package main

import (
	"gohtmx/internal/app"
	"log/slog"
	"net/http"
)

func main() {
	// Create the application
	application := app.New("todos.jsonl")

	// Set up routes
	mux := http.NewServeMux()
	application.SetupRoutes(mux)

	// Start server
	port := ":8080"
	slog.Info("Starting server", "port", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		slog.Error("Server failed to start", "error", err)
	}
}
