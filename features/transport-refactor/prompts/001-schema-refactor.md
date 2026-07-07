---
id: "001"
title: "Refactor registry schema — transport → protocol + runtime"
phase: "transport-refactor"
repo: "mcp-fleet"
model: "sonnet"
depends_on: []
budget_usd: 2.50
---

# Schema Refactor: transport → protocol + runtime

Split the overloaded `transport` field into two explicit fields.

## Background

Current `transport` conflates protocol and lifecycle:
- `transport: http` + `image:` = Docker container, HTTP
- `transport: native` = local process, stdio
- `transport: stdio` + `managed_by: claude` = Claude-managed, stdio

Target: `protocol` (http | stdio) + `runtime` (docker | process | claude).
Remove `managed_by` — `runtime: claude` replaces it.

## Tasks

### 1. Update registry.go struct

In `internal/registry/registry.go`, change the `ServerConfig` struct:

```go
// Before
Transport     string   `yaml:"transport"`
ManagedBy     string   `yaml:"managed_by,omitempty"`

// After
Protocol      string   `yaml:"protocol"`
Runtime       string   `yaml:"runtime"`
```

Do the same for `ServerEntry` if it mirrors `ServerConfig`.

### 2. Update all Go code that reads Transport/ManagedBy

Find-replace across `internal/` and `cmd/`:

| File | Old | New |
|------|-----|-----|
| `claude.go` | `cfg.Transport == "stdio" && cfg.ManagedBy == "claude"` | `cfg.Runtime == "claude"` |
| `health.go` | `cfg.Transport == "native"` | `cfg.Runtime == "process"` |
| `health.go` | `cfg.Transport == "stdio" && cfg.Health.Probe == "tool-list"` | `cfg.Runtime == "claude" && cfg.Health.Probe == "tool-list"` |
| `preflight.go` | `sc.Transport == "stdio" && sc.ManagedBy == "claude"` | `sc.Runtime == "claude"` |
| `claude.go` Register | `"--transport", "http"` | `"--transport", cfg.Protocol` |

Note: `claude mcp add --transport` still needs the protocol value passed through.

### 3. Update all tests

Fix every test fixture that sets `Transport:` or `ManagedBy:`:
- `claude_test.go` — multiple fixtures
- `native_test.go` — one fixture

### 4. Update registry.yaml

All 13 server entries. Mapping:

| Server | Old | protocol | runtime |
|--------|-----|----------|---------|
| jenkins, snowflake, grafana, grafana-pdx, airbyte, slack, slack-search, datadog, paradex-db | `transport: http` | `http` | `docker` |
| airflow | `transport: native` | `stdio` | `process` |
| graph | `transport: native` | `stdio` | `process` |
| sentry | `transport: stdio`, `managed_by: claude` | `stdio` | `claude` |
| datadog-security | `transport: stdio`, `managed_by: claude` | `stdio` | `claude` |

Remove `managed_by:` from sentry and datadog-security.

### 5. Update servers/*/server.yaml

Every `server.yaml` that has `transport:` needs the same rename.

### 6. Verify

```bash
go vet ./...
go test ./...
just build
./bin/mcpf status          # all servers listed correctly
./bin/mcpf health graph    # still healthy
```

## Constraints

- Every occurrence of `Transport` and `ManagedBy` in Go code must be gone — grep must return 0
- Tests must pass
- `mcpf status` must list all 13 servers correctly
