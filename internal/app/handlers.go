package app

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/russross/blackfriday/v2"
)

type handlers struct {
	app           *githubApp
	templateFiles embed.FS
}

func newHandlers(app *githubApp, templateFiles embed.FS) *handlers {
	return &handlers{
		app:           app,
		templateFiles: templateFiles,
	}
}

type browseData struct {
	Owner    string
	Repo     string
	Ref      string
	Path     string
	Contents []*contentItem
	Error    string
}

type contentItem struct {
	Name  string
	Path  string
	Type  string
	Size  int64
	IsDir bool
}

type refsData struct {
	Branches []string
	Tags     []string
	Current  string
}

// browseHandler handles /browse/{owner}/{repo}/{ref}/{path...}
func (h *handlers) browseHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Parse URL path: /browse/owner/repo/ref/path...
	path := strings.TrimPrefix(r.URL.Path, "/browse/")
	parts := strings.SplitN(path, "/", 4)

	if len(parts) < 2 {
		http.Error(w, "Invalid path: expected /browse/owner/repo[/ref[/path]]", http.StatusBadRequest)
		return
	}

	owner := parts[0]
	repo := parts[1]

	// Get default branch if ref not specified
	ref := ""
	filePath := ""
	if len(parts) >= 3 {
		ref = parts[2]
	}
	if len(parts) == 4 {
		filePath = parts[3]
	}

	if ref == "" {
		defaultBranch, err := h.app.getDefaultBranch(ctx, owner, repo)
		if err != nil {
			slog.Error("Failed to get default branch", "error", err, "owner", owner, "repo", repo)
			http.Error(w, fmt.Sprintf("Failed to get repository: %v", err), http.StatusInternalServerError)
			return
		}
		ref = defaultBranch
		// Redirect to include the ref in the URL
		http.Redirect(w, r, fmt.Sprintf("/browse/%s/%s/%s/%s", owner, repo, ref, filePath), http.StatusFound)
		return
	}

	// Check if this is a file or directory
	slog.Debug("Fetching contents", "owner", owner, "repo", repo, "ref", ref, "path", filePath)
	contents, err := h.app.getDirectoryContents(ctx, owner, repo, filePath, ref)
	if err != nil || len(contents) == 0 {
		// Try as file instead (API returns empty array for files sometimes)
		fileContent, fileErr := h.app.getFileContent(ctx, owner, repo, filePath, ref)
		if fileErr != nil {
			slog.Error("Failed to get contents", "dirError", err, "fileError", fileErr, "owner", owner, "repo", repo, "ref", ref, "path", filePath)
			http.Error(w, fmt.Sprintf("Failed to get contents: %v", err), http.StatusInternalServerError)
			return
		}
		h.renderFile(w, r, owner, repo, ref, filePath, fileContent)
		return
	}

	slog.Debug("Rendering directory", "path", filePath, "numItems", len(contents))
	h.renderDirectory(w, r, owner, repo, ref, filePath, contents)
}

func (h *handlers) renderDirectory(w http.ResponseWriter, r *http.Request, owner, repo, ref, path string, contents []*github.RepositoryContent) {
	data := browseData{
		Owner:    owner,
		Repo:     repo,
		Ref:      ref,
		Path:     path,
		Contents: make([]*contentItem, 0, len(contents)),
	}

	for _, content := range contents {
		item := &contentItem{
			Name: *content.Name,
			Path: *content.Path,
			Type: *content.Type,
		}
		if content.Size != nil {
			item.Size = int64(*content.Size)
		}
		item.IsDir = *content.Type == "dir"
		data.Contents = append(data.Contents, item)
	}

	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		h.renderTemplate(w, "templates/tree-list.tmpl", data)
	} else {
		h.renderTemplate(w, "templates/browse.tmpl", data)
	}
}

func (h *handlers) renderFile(w http.ResponseWriter, r *http.Request, owner, repo, ref, path string, content *github.RepositoryContent) {
	fileData := struct {
		Owner      string
		Repo       string
		Ref        string
		Path       string
		Content    string
		Size       int64
		IsMarkdown bool
		HTML       template.HTML
	}{
		Owner: owner,
		Repo:  repo,
		Ref:   ref,
		Path:  path,
	}

	decoded, err := content.GetContent()
	if err != nil {
		slog.Error("Failed to decode file content", "error", err, "path", path)
		http.Error(w, "Failed to decode file content", http.StatusInternalServerError)
		return
	}
	fileData.Content = decoded

	if content.Size != nil {
		fileData.Size = int64(*content.Size)
	}

	// Check if this is a markdown file
	if strings.HasSuffix(strings.ToLower(path), ".md") {
		fileData.IsMarkdown = true
		// Render markdown to HTML using blackfriday
		htmlBytes := blackfriday.Run([]byte(decoded))
		fileData.HTML = template.HTML(htmlBytes)
	}

	slog.Debug("Rendering file", "path", path, "size", fileData.Size, "contentLen", len(fileData.Content), "isMarkdown", fileData.IsMarkdown)

	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		h.renderTemplate(w, "templates/file-content.tmpl", fileData)
	} else {
		h.renderTemplate(w, "templates/browse-file.tmpl", fileData)
	}
}

// refsHandler handles /api/refs/{owner}/{repo}?current={ref}
func (h *handlers) refsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Parse URL path: /api/refs/owner/repo
	path := strings.TrimPrefix(r.URL.Path, "/api/refs/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) != 2 {
		http.Error(w, "Invalid path: expected /api/refs/owner/repo", http.StatusBadRequest)
		return
	}

	owner := parts[0]
	repo := parts[1]
	current := r.URL.Query().Get("current")

	branches, err := h.app.listBranches(ctx, owner, repo)
	if err != nil {
		slog.Error("Failed to list branches", "error", err, "owner", owner, "repo", repo)
		http.Error(w, "Failed to list branches", http.StatusInternalServerError)
		return
	}

	tags, err := h.app.listTags(ctx, owner, repo)
	if err != nil {
		slog.Error("Failed to list tags", "error", err, "owner", owner, "repo", repo)
		// Don't fail if tags fail
		tags = nil
	}

	data := refsData{
		Current:  current,
		Branches: make([]string, 0, len(branches)),
		Tags:     make([]string, 0, len(tags)),
	}

	for _, branch := range branches {
		if branch.Name != nil {
			data.Branches = append(data.Branches, *branch.Name)
		}
	}

	for _, tag := range tags {
		if tag.Name != nil {
			data.Tags = append(data.Tags, *tag.Name)
		}
	}

	h.renderTemplate(w, "templates/ref-selector.tmpl", data)
}

func (h *handlers) renderTemplate(w http.ResponseWriter, templatePath string, data interface{}) {
	w.Header().Set("Content-Type", "text/html")

	// Try to load the shared file-display partial
	sharedPartial, _ := h.templateFiles.ReadFile("templates/_file-display.tmpl")

	tmplData, err := h.templateFiles.ReadFile(templatePath)
	if err != nil {
		slog.Error("Template read error", "path", templatePath, "error", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	t := template.New("template").Funcs(template.FuncMap{
		"urlEncode": url.PathEscape,
	})

	// Parse shared partial first if it exists
	if len(sharedPartial) > 0 {
		t, err = t.Parse(string(sharedPartial))
		if err != nil {
			slog.Error("Template parse error for shared partial", "error", err)
			http.Error(w, "Template parse error", http.StatusInternalServerError)
			return
		}
	}

	// Then parse the main template
	t, err = t.Parse(string(tmplData))
	if err != nil {
		slog.Error("Template parse error", "path", templatePath, "error", err)
		http.Error(w, "Template parse error", http.StatusInternalServerError)
		return
	}

	if err := t.Execute(w, data); err != nil {
		slog.Error("Template execution error", "path", templatePath, "error", err)
		// Can't call http.Error here since we've already started writing the response
		return
	}
}
