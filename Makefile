.PHONY: build run docker-build docker-run clean

# Build the Go application locally
build:
	go build -o local-container-registry .

# Build and run Docker container (override default run behavior)
run:
	@echo "🐳 Building Docker image..."
	@docker build -t local-container-registry .
	@echo "✅ Docker image built successfully!"
	@echo "🚀 You can now run: docker run --rm -it local-container-registry"

# Run the Go application locally (if you need local execution)
run-local:
	go run .

# Build Docker image
docker-build:
	docker build -t local-container-registry .

# Build and run Docker container
docker-run:
	docker build -t local-container-registry .
	docker run --rm -it local-container-registry

# Clean up Docker images
clean:
	docker rmi local-container-registry || true

# Help command
help:
	@echo "Available commands:"
	@echo "  build        - Build the Go application locally"
	@echo "  run          - Build Docker image (main command)"
	@echo "  run-local    - Run the Go application locally"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Build and run Docker container"
	@echo "  clean        - Remove Docker image"
