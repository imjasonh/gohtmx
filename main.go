package main

import (
	"flag"
	"gohtmx/internal/app"
	"log/slog"
	"net/http"
)

var (
	file = flag.String("file", "todos.jsonl", "Path to the JSONL file to store todos")
	port = flag.String("port", "8080", "Port to run the server on")
)

func main() {
	flag.Parse()

	// Create the application
	application := app.New(*file)

	// Set up routes
	mux := http.NewServeMux()
	application.SetupRoutes(mux)

	// Start server

	slog.Info("Starting server", "port", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		slog.Error("Server failed to start", "error", err)
	}
}
