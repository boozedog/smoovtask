bin := "st"

# Generate templ templates
templ:
    templ generate ./internal/web/templates/

# Build the binary and install to PATH
build: templ
    go build -o {{bin}} ./cmd/st
    go install ./cmd/st

# Quick install from local source (no templ, no local binary)
install:
    go install ./cmd/st

# Run all tests (verbose)
test:
    go test -v ./...

# Run tests (short)
test-short:
    go test ./...

# Run tests with coverage
test-cover:
    go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Format code
fmt:
    gofumpt -w -extra .

# Run linter
lint:
    golangci-lint run ./...

# Run vulnerability check
vuln:
    govulncheck ./...

# Build release binaries
release:
    goreleaser build --snapshot --clean

# Run web UI dev server with live reload
web:
    air

# Remove build artifacts
clean:
    rm -f {{bin}}
