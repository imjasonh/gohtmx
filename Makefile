.PHONY: build run dev clean install-air

KO_DOCKER_REPO?=ttl.sh/jason

release:
	go build -o bin/gohtmx .

# Run the server
run:
	go run . -port=8080

# Development mode - watch for changes and auto-reload
dev:
	go tool air

image:
	KO_DOCKER_REPO=$(KO_DOCKER_REPO) ko build -P --sbom=none
