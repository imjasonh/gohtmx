# Go + HTMX Example

A sample Go binary with embedded static assets, providing an interactive web app using [htmx](https://htmx.org)

## Features

- Go server with embedded frontend assets
- Hot reload development with [air](https://github.com/air-verse/air)
- Single binary or Docker image deployment

## Development

```bash
make dev    # Start development server with hot reload
```

Changes to any file triggers a rebuild and reload, which takes ~1 second.

## Build & Run

```bash
make release  # Build standalone release binary (~11 MB)
./bin/gohtmx  # Run binary

# Or run directly:
make run    # Generate assets and run

# Or build and push a multi-arch image
make image
```

The resulting image is ~7.3 MB

Server runs on http://localhost:8080
