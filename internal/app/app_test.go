package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests use the same embedded files as the main app

func TestAppIntegration(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_todos.jsonl")

	// Create test app
	application := New(testFile)

	// Create a test server
	mux := http.NewServeMux()
	application.SetupRoutes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("EmptyTodos", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/todos")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		expected := `<div class="empty-state">No todos yet. Add one above!</div>`
		if strings.TrimSpace(string(body)) != expected {
			t.Errorf("Expected empty state, got: %s", string(body))
		}
	})

	t.Run("AddTodo", func(t *testing.T) {
		// Add a todo
		resp, err := http.PostForm(server.URL+"/todos", url.Values{
			"text": {"Test todo item"},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Check that the todo was saved to file
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Fatal("Todo file was not created")
		}

		// Read and verify the JSONL content
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}

		var todo todo
		if err := json.Unmarshal(content, &todo); err != nil {
			t.Fatal("Failed to parse JSONL:", err)
		}

		if todo.Text != "Test todo item" {
			t.Errorf("Expected 'Test todo item', got '%s'", todo.Text)
		}

		if todo.ID != 1 {
			t.Errorf("Expected ID 1, got %d", todo.ID)
		}

		if todo.Completed {
			t.Error("Expected todo to be incomplete")
		}
	})

	t.Run("GetTodos", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/todos")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		html := string(body)
		if !strings.Contains(html, "Test todo item") {
			t.Error("Todo item not found in HTML response")
		}

		if !strings.Contains(html, `hx-put="/todos/1/toggle"`) {
			t.Error("Toggle button not found")
		}

		if !strings.Contains(html, `hx-delete="/todos/1"`) {
			t.Error("Delete button not found")
		}
	})

	t.Run("ToggleTodo", func(t *testing.T) {
		// Toggle todo completion
		req, err := http.NewRequest("PUT", server.URL+"/todos/1/toggle", nil)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		html := string(body)
		if !strings.Contains(html, "completed") {
			t.Error("Todo should be marked as completed")
		}

		if !strings.Contains(html, "Undo") {
			t.Error("Should show Undo button for completed todo")
		}

		// Verify the file was updated
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}

		var todo todo
		if err := json.Unmarshal(content, &todo); err != nil {
			t.Fatal("Failed to parse JSONL:", err)
		}

		if !todo.Completed {
			t.Error("Todo should be completed in file")
		}
	})

	t.Run("AddMultipleTodos", func(t *testing.T) {
		// Add second todo
		resp, err := http.PostForm(server.URL+"/todos", url.Values{
			"text": {"Second todo"},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		// Add third todo
		resp, err = http.PostForm(server.URL+"/todos", url.Values{
			"text": {"Third todo"},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		// Verify file contains multiple JSON lines
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) != 3 {
			t.Errorf("Expected 3 lines in JSONL file, got %d", len(lines))
		}

		// Verify each line is valid JSON
		for i, line := range lines {
			var todo todo
			if err := json.Unmarshal([]byte(line), &todo); err != nil {
				t.Errorf("Line %d is not valid JSON: %s", i+1, err)
			}
		}
	})

	t.Run("DeleteTodo", func(t *testing.T) {
		// Delete the second todo (ID 2)
		req, err := http.NewRequest("DELETE", server.URL+"/todos/2", nil)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify file now contains 2 lines (deleted one)
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatal(err)
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) != 2 {
			t.Errorf("Expected 2 lines after deletion, got %d", len(lines))
		}

		// Verify the right todo was deleted (should not contain "Second todo")
		fileContent := string(content)
		if strings.Contains(fileContent, "Second todo") {
			t.Error("Deleted todo still exists in file")
		}
	})

	t.Run("StaticFiles", func(t *testing.T) {
		// Test index.html
		resp, err := http.Get(server.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.Header.Get("Content-Type") != "text/html" {
			t.Error("Expected text/html content type for index")
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		html := string(body)
		if !strings.Contains(html, "<title>TODO List</title>") {
			t.Error("Index page should contain title")
		}

		// Test CSS
		resp, err = http.Get(server.URL + "/static/style.css")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.Header.Get("Content-Type") != "text/css" {
			t.Error("Expected text/css content type for CSS")
		}
	})

	t.Run("PersistenceAcrossRestart", func(t *testing.T) {
		// Create a new app instance with the same file (simulating server restart)
		newApp := newTodoApp(testFile)

		// It should load existing todos
		if len(newApp.todos) != 2 {
			t.Errorf("Expected 2 todos after restart, got %d", len(newApp.todos))
		}

		// Check that nextID is set correctly
		if newApp.nextID != 4 { // Should be max ID + 1
			t.Errorf("Expected nextID to be 4, got %d", newApp.nextID)
		}
	})

	t.Run("LoadManualJSONL", func(t *testing.T) {
		// Create a separate test file with manually created JSONL content
		manualFile := filepath.Join(tempDir, "manual_todos.jsonl")

		// Write manual JSONL content
		manualContent := `{"id":1,"text":"Manual test todo","completed":false,"created_at":"2025-09-01T12:00:00Z"}
{"id":2,"text":"Second manual item","completed":true,"created_at":"2025-09-01T12:01:00Z"}
{"id":5,"text":"High ID todo","completed":false,"created_at":"2025-09-01T12:02:00Z"}
`

		if err := os.WriteFile(manualFile, []byte(manualContent), 0644); err != nil {
			t.Fatal("Failed to write manual JSONL file:", err)
		}

		// Create app with manual file
		manualApp := newTodoApp(manualFile)

		// Verify todos were loaded correctly
		if len(manualApp.todos) != 3 {
			t.Errorf("Expected 3 todos from manual file, got %d", len(manualApp.todos))
		}

		// Verify specific todo content
		foundManual := false
		foundCompleted := false
		for _, todo := range manualApp.todos {
			if todo.Text == "Manual test todo" && todo.ID == 1 && !todo.Completed {
				foundManual = true
			}
			if todo.Text == "Second manual item" && todo.ID == 2 && todo.Completed {
				foundCompleted = true
			}
		}

		if !foundManual {
			t.Error("Manual test todo not found or incorrect")
		}

		if !foundCompleted {
			t.Error("Completed manual todo not found or incorrect")
		}

		// Verify nextID is set correctly (should be max ID + 1 = 6)
		if manualApp.nextID != 6 {
			t.Errorf("Expected nextID to be 6, got %d", manualApp.nextID)
		}

		// Test that we can add a new todo with correct ID
		if err := manualApp.addTodo("New todo after manual load"); err != nil {
			t.Fatal("Failed to add todo after manual load:", err)
		}

		// Verify the new todo has ID 6
		if len(manualApp.todos) != 4 {
			t.Error("Expected 4 todos after adding one")
		}

		// The new todo should be at the beginning (newest first)
		newTodo := manualApp.todos[0]
		if newTodo.ID != 6 {
			t.Errorf("Expected new todo to have ID 6, got %d", newTodo.ID)
		}

		if newTodo.Text != "New todo after manual load" {
			t.Errorf("Expected 'New todo after manual load', got '%s'", newTodo.Text)
		}

		// Verify file contains valid JSONL with 4 lines
		content, err := os.ReadFile(manualFile)
		if err != nil {
			t.Fatal("Failed to read updated manual file:", err)
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) != 4 {
			t.Errorf("Expected 4 lines in updated JSONL file, got %d", len(lines))
		}

		// Verify each line is valid JSON
		for i, line := range lines {
			var todo todo
			if err := json.Unmarshal([]byte(line), &todo); err != nil {
				t.Errorf("Line %d is not valid JSON: %s", i+1, err)
			}
		}
	})
}

func TestTodoAppUnit(t *testing.T) {
	t.Run("NewTodoApp", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "new_test_todos.jsonl")
		app := newTodoApp(testFile)

		if len(app.todos) != 0 {
			t.Error("Expected empty todos list for new app")
		}

		if app.nextID != 1 {
			t.Errorf("Expected nextID to be 1, got %d", app.nextID)
		}

		if app.filename != testFile {
			t.Errorf("Expected filename to be %s, got %s", testFile, app.filename)
		}
	})

	t.Run("AddTodo", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "add_test_todos.jsonl")
		app := newTodoApp(testFile)

		err := app.addTodo("First todo")
		if err != nil {
			t.Fatal("Failed to add todo:", err)
		}

		if len(app.todos) != 1 {
			t.Errorf("Expected 1 todo, got %d", len(app.todos))
		}

		todo := app.todos[0]
		if todo.Text != "First todo" {
			t.Errorf("Expected 'First todo', got '%s'", todo.Text)
		}

		if todo.ID != 1 {
			t.Errorf("Expected ID 1, got %d", todo.ID)
		}

		if todo.Completed {
			t.Error("Expected todo to be incomplete")
		}
	})

	t.Run("ToggleTodo", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "toggle_test_todos.jsonl")
		app := newTodoApp(testFile)
		app.addTodo("Toggle test")

		// Toggle to completed
		err := app.toggleTodo(1)
		if err != nil {
			t.Fatal("Failed to toggle todo:", err)
		}

		if !app.todos[0].Completed {
			t.Error("Expected todo to be completed after toggle")
		}

		// Toggle back to incomplete
		err = app.toggleTodo(1)
		if err != nil {
			t.Fatal("Failed to toggle todo back:", err)
		}

		if app.todos[0].Completed {
			t.Error("Expected todo to be incomplete after second toggle")
		}

		// Try to toggle non-existent todo
		err = app.toggleTodo(999)
		if err == nil {
			t.Error("Expected error when toggling non-existent todo")
		}
	})

	t.Run("DeleteTodo", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "delete_test_todos.jsonl")
		app := newTodoApp(testFile)
		app.addTodo("To be deleted")
		app.addTodo("To remain")

		if len(app.todos) != 2 {
			t.Fatal("Setup failed: expected 2 todos")
		}

		// Delete first todo
		err := app.deleteTodo(1)
		if err != nil {
			t.Fatal("Failed to delete todo:", err)
		}

		if len(app.todos) != 1 {
			t.Errorf("Expected 1 todo after deletion, got %d", len(app.todos))
		}

		if app.todos[0].Text != "To remain" {
			t.Error("Wrong todo was deleted")
		}

		// Try to delete non-existent todo
		err = app.deleteTodo(999)
		if err == nil {
			t.Error("Expected error when deleting non-existent todo")
		}
	})

	t.Run("GetTodos", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "get_test_todos.jsonl")
		app := newTodoApp(testFile)
		app.addTodo("First")
		app.addTodo("Second")

		todos := app.getTodos()
		if len(todos) != 2 {
			t.Errorf("Expected 2 todos, got %d", len(todos))
		}

		// Should be in reverse order (newest first)
		if todos[0].Text != "Second" {
			t.Error("Expected newest todo first")
		}

		if todos[1].Text != "First" {
			t.Error("Expected oldest todo last")
		}
	})
}
