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

func TestApp(t *testing.T) {
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
		// Add a todo via HTTP
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
		// Toggle todo completion via HTTP
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
	})

	t.Run("AddMultipleTodos", func(t *testing.T) {
		// Add second todo via HTTP
		resp, err := http.PostForm(server.URL+"/todos", url.Values{
			"text": {"Second todo"},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		// Add third todo via HTTP
		resp, err = http.PostForm(server.URL+"/todos", url.Values{
			"text": {"Third todo"},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		// Test todo ordering (newest first)
		resp, err = http.Get(server.URL + "/todos")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		html := string(body)
		thirdTodoPos := strings.Index(html, "Third todo")
		firstTodoPos := strings.Index(html, "Test todo item")
		if thirdTodoPos == -1 || firstTodoPos == -1 {
			t.Error("Expected todos not found in HTML")
		} else if thirdTodoPos > firstTodoPos {
			t.Error("Expected newest todo (Third todo) to appear before older todo (Test todo item)")
		}
	})

	t.Run("DeleteTodo", func(t *testing.T) {
		// Delete the second todo (ID 2) via HTTP
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
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Try to toggle non-existent todo
		req, err := http.NewRequest("PUT", server.URL+"/todos/999/toggle", nil)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Error("Expected error status when toggling non-existent todo")
		}

		// Try to delete non-existent todo
		req, err = http.NewRequest("DELETE", server.URL+"/todos/999", nil)
		if err != nil {
			t.Fatal(err)
		}

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Error("Expected error status when deleting non-existent todo")
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
