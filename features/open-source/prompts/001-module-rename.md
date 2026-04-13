---
id: "001"
title: "Rename Go module and create example configs"
phase: "oss-prep"
repo: "mcp-fleet"
model: "sonnet"
depends_on: []
budget_usd: 2.00
---

# Module Rename + Example Configs

## Tasks

### 1. Change Go module path

`go.mod`: change `github.com/pdonorio/mcp-fleet` → `github.com/paperworlds/mcp-fleet`

Find-replace `github.com/pdonorio/mcp-fleet` in every `.go` file under `cmd/` and `internal/`.
Use `go mod edit -module github.com/paperworlds/mcp-fleet` then fix imports.

### 2. Create registry.yaml.example

Three fictional servers showing each transport type (http, native, stdio).
Use placeholder values:
- `op://YourVault/YourItem/field` for 1Password
- `https://your-instance.example.com` for URLs
- `your-aws-profile` for AWS profiles

### 3. Create servers/example-http/server.yaml

Full schema example with comments explaining every field. Include:
- image, transport, port, container_port, endpoint_path
- tags, deps, env (with op:// placeholder), volumes, extra_args
- health config
- A `hook.sh` pre-start template
- A `README.md` documenting the schema

### 4. Create servers/example-native/server.yaml

Native transport example showing command, args, working_dir, pid file, PID probe.

### 5. Verify

```bash
go vet ./...
go test ./...
just build
./bin/mcpf --version
```

## Constraints

- Every `.go` file import must be updated — grep for old module path after, must return 0 results
- Example configs must be valid YAML that `mcpf` can parse
