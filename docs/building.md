# Building & Running

```sh
just build          # templ generate + build + install to GOPATH/bin
just install        # quick install from local source (no templ, no local binary)
just test           # go test -v ./...
just test-short     # go test ./... (non-verbose)
just test-cover     # tests with HTML coverage report
just lint           # golangci-lint
just fmt            # gofumpt (formatter)
just vuln           # govulncheck
just templ          # generate templ templates only
just web            # run web UI dev server with air (live reload)
just vendor         # vendor DaisyUI/Tailwind/htmx from npm
just release        # goreleaser snapshot build
just clean          # remove build artifacts
```
