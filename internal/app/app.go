package app

import (
	"embed"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

type App struct {
	githubApp *githubApp
	handlers  *handlers
}

func New() *App {
	githubApp := newGithubApp()
	handlers := newHandlers(githubApp, templateFiles)

	if os.Getenv("GITHUB_TOKEN") == "" {
		slog.Info("GitHub API authentication not configured (using unauthenticated requests)")
	}

	return &App{
		githubApp: githubApp,
		handlers:  handlers,
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
			http.FileServer(http.FS(staticFiles)).ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	})

	// GitHub browser routes
	mux.HandleFunc("/browse/", app.handlers.browseHandler)
	mux.HandleFunc("/api/refs/", app.handlers.refsHandler)
}
