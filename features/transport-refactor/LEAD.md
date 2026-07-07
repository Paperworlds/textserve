# transport-refactor — Feature Lead

Feature of: mcp-fleet

## What this feature does

Split the overloaded `transport` field into two explicit fields: `protocol` and
`runtime`. Makes the registry schema clearer and extensible for the open-source
release as `textserve`.

## Current schema (overloaded)

```yaml
transport: http    # + image: → means Docker container, HTTP protocol
transport: native  # → means local process, stdio protocol
transport: stdio   # + managed_by: claude → means Claude-managed, stdio protocol
```

`transport` conflates "how it talks" with "who runs it."

## Target schema

```yaml
protocol: http | stdio          # how the MCP server communicates
runtime: docker | process | claude   # who manages the lifecycle
```

Mapping:
| Old | New protocol | New runtime |
|-----|-------------|-------------|
| `transport: http` + `image:` | `protocol: http` | `runtime: docker` |
| `transport: native` | `protocol: stdio` | `runtime: process` |
| `transport: stdio` + `managed_by: claude` | `protocol: stdio` | `runtime: claude` |

## Files that need changes

Go code (~6 switch/comparison sites):
- `internal/registry/registry.go` — struct fields: `Transport` → `Protocol` + `Runtime`
- `internal/claude/claude.go` — 2 checks on Transport
- `internal/health/health.go` — 2 checks on Transport
- `internal/preflight/preflight.go` — 1 check on Transport
- `internal/native/native_test.go` — 1 test fixture
- `internal/claude/claude_test.go` — multiple test fixtures

Data files:
- `registry.yaml` — all 13 server entries
- `servers/*/server.yaml` — all server configs

## Constraints

- `go vet ./...` and `go test ./...` must pass after
- `mcpf status` must work with the new schema
- Backward compat: if a registry.yaml still has `transport:`, map it to the new fields with a deprecation warning (optional, can skip for clean break if doing this on the OSS branch)
- `managed_by` field is removed — `runtime: claude` replaces it

## Running pp in the background

Always use the Bash tool's `run_in_background: true` parameter when launching
`pp run <id>` — never use shell `&`.
