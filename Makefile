.PHONY: dev/api dev/pwa install/pwa build build/api build/pwa test lint typecheck docker/build docker/run clean

# Run the Go API server locally
dev/api:
	cd api && go run .

# Run the Vite dev server
dev/pwa:
	cd pwa && npm run dev

# Install PWA dependencies (CI-safe: npm ci)
install/pwa:
	cd pwa && npm ci

# Build both PWA and API
build: build/pwa build/api

# Compile the Go binary
build/api:
	cd api && go build -v ./...

# Build the PWA into api/static
build/pwa:
	cd pwa && npm run build

# Run Go tests with race detector
test:
	cd api && go test -v -race ./...

# Lint both Go and TypeScript
lint:
	cd api && golangci-lint run
	cd pwa && npm run lint

# Type-check the PWA without emitting files
typecheck:
	cd pwa && npx tsc -b --noEmit

TAG ?= latest

# Build the Docker image
docker/build:
	docker build -t brewmaster:$(TAG) .

# Run the Docker image locally (requires ANTHROPIC_API_KEY in env)
docker/run:
	docker run --rm -p 8080:8080 \
		-e ANTHROPIC_API_KEY="$$ANTHROPIC_API_KEY" \
brewmaster:$(TAG)

# Remove build artifacts
clean:
	rm -f brewmaster
	rm -rf api/static
