# Build the application
build:
    go build -o bin/moshr cmd/moshr/main.go

# Run in web mode (default port 8080)
run: build
    ./bin/moshr -web
alias r := run

# Run with custom port
run-port PORT: build
    ./bin/moshr -web -port={{PORT}}

# Install dependencies
deps:
    go mod tidy

# Clean build artifacts
clean:
    rm -rf bin/
    rm -f *.avi *.mp4 *.webm *.jpg *.jpeg *.png

# Development server with live reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
    air -c .air.toml

# Format code
fmt:
    go fmt ./...

# Run tests
test:
    go test ./...

# Lint code (requires golangci-lint)
lint:
    golangci-lint run

# Build for different platforms
build-all:
    GOOS=linux GOARCH=amd64 go build -o bin/moshr-linux-amd64 cmd/moshr/main.go
    GOOS=darwin GOARCH=amd64 go build -o bin/moshr-darwin-amd64 cmd/moshr/main.go
    GOOS=darwin GOARCH=arm64 go build -o bin/moshr-darwin-arm64 cmd/moshr/main.go
    GOOS=windows GOARCH=amd64 go build -o bin/moshr-windows-amd64.exe cmd/moshr/main.go

# Default recipe
default: build