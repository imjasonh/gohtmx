# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A GitHub repository browser built with Go and HTMX. The application allows browsing public GitHub repositories with server-side rendering and HTMX-powered dynamic updates.

## Architecture

### Application Structure

**Three-layer architecture:**

1. **`main.go`**: Entry point, sets up HTTP server and routes
2. **`internal/app/app.go`**: Application initialization, embeds static assets and templates using `//go:embed`
3. **`internal/app/handlers.go`**: HTTP handlers for routing and request processing
4. **`internal/app/github.go`**: GitHub API client wrapper using `google/go-github/v66`

### Key Design Patterns

**Embedded Assets**: All templates (`templates/*.tmpl`) and static files (`static/*`) are embedded at compile time using `go:embed`. The single binary contains everything needed to run.

**HTMX Philosophy**: Server returns HTML fragments. HTMX in the browser handles:
- `hx-get`: Fetching content
- `hx-target`: Where to inject responses
- `hx-swap`: How to replace content
- `hx-push-url`: Updating browser URL without page refresh

Avoid JavaScript fetch/XMLHttpRequest - let HTMX handle it declaratively in HTML attributes.

**Template Rendering**: Two template types per feature:
- Full page templates (`browse.tmpl`, `browse-file.tmpl`): For direct URL access
- Partial templates (`tree-list.tmpl`, `file-content.tmpl`): For HTMX requests

Handlers check `HX-Request` header to decide which template to render.

### URL Structure

All repository content follows RESTful pattern: `/browse/{owner}/{repo}/{ref}/{path}`

- `owner/repo`: GitHub repository
- `ref`: Branch or tag name (e.g., `main`, `v1.0.0`)
- `path`: File or directory path within repo (empty for root)

Examples:
- `/browse/golang/go/master/` - Root directory
- `/browse/golang/go/master/src/runtime/proc.go` - Specific file

### GitHub API Integration

**Authentication**: Optional via `GITHUB_TOKEN` environment variable. The Makefile automatically exports token from `gh auth token`.

**Rate Limiting**: Unauthenticated: 60 req/hour. Authenticated: 5000 req/hour.

**File vs Directory Detection**: GitHub API quirk - files may return empty directory array. Handler logic: if `getDirectoryContents()` returns empty array OR errors, try `getFileContent()`.

## Development Commands

### Hot Reload Development
```bash
make dev
```
Uses `go tool air` for automatic rebuild on file changes (~1 second). Air is a go tool dependency.

### Build and Run
```bash
make release    # Build binary to bin/gohtmx
make run        # go run with port 8080
```

### Testing
```bash
go test ./...                    # All tests
go test ./internal/app           # Package-specific
go test -run TestApp/EmptyTodos  # Single subtest
```

The test file `app_test.go` is currently outdated (references old todo app code). Tests need updating for GitHub browser functionality.

### Container Images
```bash
make image  # Uses ko to build multi-arch image
```
Uses `ko` (not Docker) to build ~7.3 MB images. Configure with `KO_DOCKER_REPO` environment variable.

## Template System

Templates use Go's `html/template` with custom functions:
- `urlEncode`: For URL path encoding

Template data structures defined in `handlers.go`:
- `browseData`: Directory listings (owner, repo, ref, path, contents)
- `contentItem`: Individual file/directory entries
- `refsData`: Branches and tags for ref selector

## Making Changes

When adding features that involve both directory and file views, remember to:
1. Update both full-page and partial templates
2. Handle both HTMX and non-HTMX request paths
3. Update URL patterns in handlers and template links
4. Test direct URL access and HTMX navigation

When modifying styles, the UI prioritizes information density:
- Base font: 13px
- Compact spacing (0.25-0.5rem margins/padding)
- Viewport-based heights for maximum content visibility
