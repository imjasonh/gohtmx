package app

import (
	"embed"
	"net/http"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

type App struct {
	todoApp  *todoApp
	handlers *handlers
}

func New(filename string) *App {
	todoApp := newTodoApp(filename)
	handlers := newHandlers(todoApp, templateFiles)

	return &App{
		todoApp:  todoApp,
		handlers: handlers,
	}
}

func (app *App) SetupRoutes(mux *http.ServeMux) {
	// Static files and root handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			data, err := staticFiles.ReadFile("static/index.html")
			if err != nil {
				http.Error(w, "Could not read index.html", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write(data)
			return
		}

		// Serve static files for any path that starts with /static/
		if strings.HasPrefix(r.URL.Path, "/static/") {
			// Strip the /static prefix and serve from the embedded filesystem
			path := strings.TrimPrefix(r.URL.Path, "/static/")
			data, err := staticFiles.ReadFile("static/" + path)
			if err != nil {
				http.NotFound(w, r)
				return
			}

			// Set appropriate content type
			if strings.HasSuffix(path, ".css") {
				w.Header().Set("Content-Type", "text/css")
			} else if strings.HasSuffix(path, ".js") {
				w.Header().Set("Content-Type", "application/javascript")
			}

			w.Write(data)
			return
		}

		http.NotFound(w, r)
	})

	// API routes
	mux.HandleFunc("/todos", app.handlers.todosHandler)
	mux.HandleFunc("/todos/", app.handlers.todosPathHandler)
}
