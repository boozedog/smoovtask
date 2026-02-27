# Building & Running

```sh
just install        # quick install from local source (use during dev)
just build          # templ generate + build + install
just test           # go test -v ./...
just test-short     # go test ./... (non-verbose)
just test-cover     # tests with HTML coverage report
just lint           # golangci-lint
just fmt            # gofumpt (formatter)
just vuln           # govulncheck
just release        # goreleaser snapshot build
just clean          # remove build artifacts
```
