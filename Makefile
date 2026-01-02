.PHONY: build run test clean docker docker-run lint

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Build the binary
build:
	go build -ldflags="$(LDFLAGS)" -o updati ./cmd/updati

# Run the binary
run: build
	./updati

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f updati
	go clean

# Build Docker image
docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(DATE) \
		-t updati:$(VERSION) \
		-t updati:latest \
		.

# Run in Docker
docker-run:
	docker run --rm \
		-e GITHUB_TOKEN \
		-e UPDATI_OWNER \
		-e UPDATI_REPO_PATTERNS \
		updati:latest

# Run linter
lint:
	golangci-lint run ./...

# Install development dependencies
dev-deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Format code
fmt:
	go fmt ./...

# Check if code is formatted
check-fmt:
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)

