# textserve Justfile

build:
    go build -ldflags "-X main.version=$(cat VERSION)" -o bin/textserve ./cmd/textserve

test:
    go test ./...

lint:
    go vet ./...

install: build
    ln -sf $(pwd)/bin/textserve ~/.local/bin/textserve
    mkdir -p ~/.config/fish/completions
    ln -sf $(pwd)/completions/textserve.fish ~/.config/fish/completions/textserve.fish
