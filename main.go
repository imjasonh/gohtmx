package main

import (
	"bufio"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

type Todo struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

type TodoApp struct {
	todos    []Todo
	nextID   int
	filename string
}

func NewTodoApp(filename string) *TodoApp {
	app := &TodoApp{
		todos:    []Todo{},
		nextID:   1,
		filename: filename,
	}
	app.loadTodos()
	return app
}

func (app *TodoApp) loadTodos() error {
	file, err := os.Open(app.filename)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 4)
		if len(parts) != 4 {
			continue
		}

		id, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		completed := parts[1] == "true"
		text := parts[2]
		createdAt, err := time.Parse(time.RFC3339, parts[3])
		if err != nil {
			createdAt = time.Now()
		}

		todo := Todo{
			ID:        id,
			Text:      text,
			Completed: completed,
			CreatedAt: createdAt,
		}

		app.todos = append(app.todos, todo)
		if id >= app.nextID {
			app.nextID = id + 1
		}
	}

	return scanner.Err()
}

func (app *TodoApp) saveTodos() error {
	file, err := os.Create(app.filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, todo := range app.todos {
		line := fmt.Sprintf("%d|%t|%s|%s\n",
			todo.ID,
			todo.Completed,
			todo.Text,
			todo.CreatedAt.Format(time.RFC3339))
		if _, err := file.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

func (app *TodoApp) addTodo(text string) error {
	todo := Todo{
		ID:        app.nextID,
		Text:      text,
		Completed: false,
		CreatedAt: time.Now(),
	}

	app.todos = append([]Todo{todo}, app.todos...)
	app.nextID++

	return app.saveTodos()
}

func (app *TodoApp) toggleTodo(id int) error {
	for i := range app.todos {
		if app.todos[i].ID == id {
			app.todos[i].Completed = !app.todos[i].Completed
			return app.saveTodos()
		}
	}
	return fmt.Errorf("todo with id %d not found", id)
}

func (app *TodoApp) deleteTodo(id int) error {
	for i, todo := range app.todos {
		if todo.ID == id {
			app.todos = append(app.todos[:i], app.todos[i+1:]...)
			return app.saveTodos()
		}
	}
	return fmt.Errorf("todo with id %d not found", id)
}

func main() {
	app := NewTodoApp("todos.txt")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

	http.HandleFunc("/todos", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetTodos(w, r, app)
		case http.MethodPost:
			handleAddTodo(w, r, app)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/todos/", func(w http.ResponseWriter, r *http.Request) {
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
			handleToggleTodo(w, r, app, id)
		} else if len(parts) == 1 && r.Method == http.MethodDelete {
			handleDeleteTodo(w, r, app, id)
		} else {
			http.Error(w, "Invalid request", http.StatusBadRequest)
		}
	})

	port := ":8080"
	slog.Info("Starting server", "port", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		slog.Error("Server failed to start", "error", err)
	}
}

func handleGetTodos(w http.ResponseWriter, r *http.Request, app *TodoApp) {
	w.Header().Set("Content-Type", "text/html")

	if len(app.todos) == 0 {
		data, err := templateFiles.ReadFile("templates/empty-state.tmpl")
		if err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, string(data))
		return
	}

	tmplData, err := templateFiles.ReadFile("templates/todos-list.tmpl")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	t, err := template.New("todos").Parse(string(tmplData))
	if err != nil {
		http.Error(w, "Template parse error", http.StatusInternalServerError)
		return
	}

	if err := t.Execute(w, app.todos); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}

func handleAddTodo(w http.ResponseWriter, r *http.Request, app *TodoApp) {
	text := strings.TrimSpace(r.FormValue("text"))
	if text == "" {
		http.Error(w, "Todo text cannot be empty", http.StatusBadRequest)
		return
	}

	if err := app.addTodo(text); err != nil {
		slog.Error("Failed to add todo", "error", err)
		http.Error(w, "Failed to add todo", http.StatusInternalServerError)
		return
	}

	handleGetTodos(w, r, app)
}

func handleToggleTodo(w http.ResponseWriter, r *http.Request, app *TodoApp, id int) {
	if err := app.toggleTodo(id); err != nil {
		slog.Error("Failed to toggle todo", "error", err, "id", id)
		http.Error(w, "Failed to toggle todo", http.StatusInternalServerError)
		return
	}

	handleGetTodos(w, r, app)
}

func handleDeleteTodo(w http.ResponseWriter, r *http.Request, app *TodoApp, id int) {
	if err := app.deleteTodo(id); err != nil {
		slog.Error("Failed to delete todo", "error", err, "id", id)
		http.Error(w, "Failed to delete todo", http.StatusInternalServerError)
		return
	}

	handleGetTodos(w, r, app)
}
