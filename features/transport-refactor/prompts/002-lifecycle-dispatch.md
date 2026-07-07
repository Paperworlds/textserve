---
id: "002"
title: "Update start/stop/register dispatch to use runtime field"
phase: "transport-refactor"
repo: "mcp-fleet"
model: "sonnet"
depends_on: ["001"]
budget_usd: 2.00
---

# Update Lifecycle Dispatch

After the schema refactor, the start/stop/register logic in `cmd/mcpf/main.go`
needs to dispatch based on `runtime` instead of `transport`.

## Tasks

### 1. Audit the start path

Read `cmd/mcpf/main.go` and trace the start logic for each server type. Currently
it likely switches on `transport`:

- `http` + `image` → docker.Start
- `native` → native.Start
- `stdio` + `managed_by: claude` → skip (Claude manages)

Update to switch on `runtime`:
- `docker` → docker.Start
- `process` → native.Start
- `claude` → skip with message

### 2. Audit the stop path

Same pattern — update stop dispatch from transport to runtime.

### 3. Audit the register path

`claude.Register` uses `cfg.Transport` to decide the `--transport` flag value
passed to `claude mcp add`. This should now use `cfg.Protocol`:

```go
args := []string{"mcp", "add", "--transport", cfg.Protocol, ...}
```

Also: for `runtime: claude`, registration is a no-op (Claude already has it).
For `runtime: docker`, registration passes the HTTP URL.
For `runtime: process`, registration passes the stdio command.

### 4. Audit health dispatch

`health.Probe()` dispatches based on transport. Update:
- `runtime == "docker"` → HTTP health endpoint
- `runtime == "process"` → PID probe or TCP probe
- `runtime == "claude"` → tool-list probe

### 5. Update status/list output

If `mcpf status` or `mcpf list` displays the transport column, update to show
`protocol/runtime` or just `runtime`.

### 6. Verify end to end

```bash
go vet ./...
go test ./...
just build

# Docker server:
./bin/mcpf start datadog
./bin/mcpf health datadog
./bin/mcpf stop datadog

# Process server:
./bin/mcpf start graph
./bin/mcpf health graph

# Claude-managed:
./bin/mcpf start sentry     # should print "managed by Claude"

./bin/mcpf status            # all 13 servers listed with correct runtime
```

## Constraints

- `go vet ./...` and `go test ./...` must pass
- Every runtime type must start/stop/health correctly
- `mcpf status` output must be readable and correct
