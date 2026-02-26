bin := "sb"

# Build the binary and install to PATH
build:
    go build -o {{bin}} ./cmd/sb
    go install ./cmd/sb

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

# Remove build artifacts
clean:
    rm -f {{bin}}
