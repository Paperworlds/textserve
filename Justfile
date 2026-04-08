# mcp-fleet Justfile

build:
    go build -ldflags "-X main.version=$(cat VERSION)" -o bin/mcpf ./cmd/mcpf

test:
    go test ./...

lint:
    go vet ./...

install: build
    ln -sf $(pwd)/bin/mcpf ~/.local/bin/mcpf
    mkdir -p ~/.config/fish/completions
    ln -sf $(pwd)/completions/mcpf.fish ~/.config/fish/completions/mcpf.fish
