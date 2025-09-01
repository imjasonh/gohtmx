package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type todo struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

type todoApp struct {
	todos    []todo
	nextID   int
	filename string
}

func newTodoApp(filename string) *todoApp {
	app := &todoApp{
		todos:    []todo{},
		nextID:   1,
		filename: filename,
	}
	app.loadTodos()
	return app
}

func (app *todoApp) loadTodos() error {
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

		var todo todo
		if err := json.Unmarshal([]byte(line), &todo); err != nil {
			slog.Warn("Failed to parse todo line", "line", line, "error", err)
			continue
		}

		app.todos = append(app.todos, todo)
		if todo.ID >= app.nextID {
			app.nextID = todo.ID + 1
		}
	}

	return scanner.Err()
}

func (app *todoApp) saveTodos() error {
	file, err := os.Create(app.filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, todo := range app.todos {
		if err := encoder.Encode(todo); err != nil {
			return err
		}
	}

	return nil
}

func (app *todoApp) addTodo(text string) error {
	newTodo := todo{
		ID:        app.nextID,
		Text:      text,
		Completed: false,
		CreatedAt: time.Now(),
	}

	app.todos = append([]todo{newTodo}, app.todos...)
	app.nextID++

	return app.saveTodos()
}

func (app *todoApp) toggleTodo(id int) error {
	for i := range app.todos {
		if app.todos[i].ID == id {
			app.todos[i].Completed = !app.todos[i].Completed
			return app.saveTodos()
		}
	}
	return fmt.Errorf("todo with id %d not found", id)
}

func (app *todoApp) deleteTodo(id int) error {
	for i, todo := range app.todos {
		if todo.ID == id {
			app.todos = append(app.todos[:i], app.todos[i+1:]...)
			return app.saveTodos()
		}
	}
	return fmt.Errorf("todo with id %d not found", id)
}

func (app *todoApp) getTodos() []todo {
	return app.todos
}
