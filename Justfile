# textserve Justfile

build:
    go build -ldflags "-X main.version=$(cat VERSION)" -o bin/textserve ./cmd/textserve
    go build -ldflags "-X main.version=$(cat VERSION)" -o bin/textserve-mcp ./cmd/textserve-mcp

test:
    go test ./...

lint:
    go vet ./...

install: build
    ln -sf $(pwd)/bin/textserve ~/.local/bin/textserve
    ln -sf $(pwd)/bin/textserve-mcp ~/.local/bin/textserve-mcp
    mkdir -p ~/.config/fish/completions
    ln -sf $(pwd)/completions/textserve.fish ~/.config/fish/completions/textserve.fish
