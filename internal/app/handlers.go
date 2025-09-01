package app

import (
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

type handlers struct {
	app           *todoApp
	templateFiles embed.FS
}

func newHandlers(app *todoApp, templateFiles embed.FS) *handlers {
	return &handlers{
		app:           app,
		templateFiles: templateFiles,
	}
}

func (h *handlers) getTodos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	if len(h.app.todos) == 0 {
		data, err := h.templateFiles.ReadFile("templates/empty-state.tmpl")
		if err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, string(data))
		return
	}

	tmplData, err := h.templateFiles.ReadFile("templates/todos-list.tmpl")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	t, err := template.New("todos").Parse(string(tmplData))
	if err != nil {
		http.Error(w, "Template parse error", http.StatusInternalServerError)
		return
	}

	if err := t.Execute(w, h.app.todos); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}

func (h *handlers) addTodo(w http.ResponseWriter, r *http.Request) {
	text := strings.TrimSpace(r.FormValue("text"))
	if text == "" {
		http.Error(w, "Todo text cannot be empty", http.StatusBadRequest)
		return
	}

	if err := h.app.addTodo(text); err != nil {
		slog.Error("Failed to add todo", "error", err)
		http.Error(w, "Failed to add todo", http.StatusInternalServerError)
		return
	}

	h.getTodos(w, r)
}

func (h *handlers) toggleTodo(w http.ResponseWriter, r *http.Request, id int) {
	if err := h.app.toggleTodo(id); err != nil {
		slog.Error("Failed to toggle todo", "error", err, "id", id)
		http.Error(w, "Failed to toggle todo", http.StatusInternalServerError)
		return
	}

	h.getTodos(w, r)
}

func (h *handlers) deleteTodo(w http.ResponseWriter, r *http.Request, id int) {
	if err := h.app.deleteTodo(id); err != nil {
		slog.Error("Failed to delete todo", "error", err, "id", id)
		http.Error(w, "Failed to delete todo", http.StatusInternalServerError)
		return
	}

	h.getTodos(w, r)
}

func (h *handlers) todosHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getTodos(w, r)
	case http.MethodPost:
		h.addTodo(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *handlers) todosPathHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/todos/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 {
		http.Error(w, "Invalid todo ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "Invalid todo ID", http.StatusBadRequest)
		return
	}

	if len(parts) == 2 && parts[1] == "toggle" && r.Method == http.MethodPut {
		h.toggleTodo(w, r, id)
	} else if len(parts) == 1 && r.Method == http.MethodDelete {
		h.deleteTodo(w, r, id)
	} else {
		http.Error(w, "Invalid request", http.StatusBadRequest)
	}
}
