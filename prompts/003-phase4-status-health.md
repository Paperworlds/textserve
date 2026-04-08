---
id: "003"
title: "Phase 4 — status dashboard + health checks + doctor"
phase: "phase-4"
repo: "mcp-fleet"
model: "sonnet"
depends_on: ["002"]
budget_usd: 2.50
---

# Phase 4 — Status & Health

Implement `mcpf status`, `mcpf health`, and `mcpf doctor`. These are the
observability commands — no credential resolution or container lifecycle needed,
just querying state and probing endpoints.

## Tasks

### 1. internal/health package

`internal/health/health.go`:

**`ProbeHTTP(name string, cfg *registry.ServerConfig) error`**
- `curl -sf --max-time <timeout> http://localhost:<port><health.endpoint>`
- Returns nil if exit 0, error with status code otherwise

**`ProbePID(cfg *registry.ServerConfig) error`**
- Read PID from `pid_file`, check process alive with `kill -0 <pid>`
- Returns nil if alive

**`ProbeToolList(name string) error`**
- Send MCP `initialize` + `tools/list` JSON-RPC to `claude mcp get-tools <name>`
  (or equivalent stdio probe) — returns nil if response has tool count > 0
- If no viable probe, return a "probe not implemented" error (not a failure)

**`Probe(name string, cfg *registry.ServerConfig) (string, error)`**
- Dispatcher: picks probe type based on transport + health.probe field
  - `native` → ProbePID
  - `stdio` + `probe: tool-list` → ProbeToolList
  - `http` → ProbeHTTP
- Returns ("healthy"/"unhealthy"/"unknown", error)

### 2. mcpf status

Output table columns: `NAME | TRANSPORT | PORT | STATUS | UPTIME`

- STATUS: healthy / unhealthy / stopped / unknown
- UPTIME: from `docker inspect --format {{.State.StartedAt}}` for docker servers;
  from PID start time for native; "-" for stdio
- Use `github.com/olekukonko/tablewriter`

`--json` flag outputs:
```json
{
  "updated_at": "2026-04-08T14:00:00Z",
  "healthy": 8,
  "total": 12,
  "unhealthy": ["airbyte", "airflow"]
}
```

Also write that JSON to `~/.files/states/mcp-fleet.json` on every `status` run
(create parent dirs with `os.MkdirAll`).

### 3. mcpf health [name]

- No args: probe all servers, print per-server result, exit 1 if any unhealthy
- With name: probe single server, exit 1 if unhealthy
- Output: `✓ jenkins  healthy` / `✗ airbyte  unhealthy: connection refused`

### 4. mcpf doctor

Full diagnostic — does not require servers to be running:

1. **Registry parse** — Load registry.yaml successfully
2. **Server configs** — Load all 12 server.yaml files without error
3. **Port conflicts** — No two servers share the same port
4. **Image availability** — For docker servers: `docker manifest inspect <image>`
   (warn if unavailable, don't fail)
5. **Dep checks** — Run each server's `deps[].cmd`, report pass/fail + hint
6. **Cache status** — For each op-cached credential, report whether cache file
   exists (doesn't validate, just presence check)

Output format:
```
[PASS] registry.yaml parses cleanly (12 servers)
[PASS] all server.yaml files load
[PASS] no port conflicts
[WARN] image mcp-airbyte not found in local registry (pull needed)
[FAIL] airflow dep: curl warp check failed — Enable Cloudflare WARP
[PASS] jenkins credentials cached
[MISS] snowflake credentials not cached (run: mcpf start snowflake)
```

Exit code: 0 if no FAILs (WARNs are OK), 1 if any FAIL.

### 5. Tests

`internal/health/health_test.go`:
- Test ProbeHTTP against a local httptest.Server that returns 200 → healthy
- Test ProbeHTTP against a server that returns 500 → unhealthy
- Test ProbeHTTP with unreachable port → unhealthy
- Test ProbePID with current process PID → healthy
- Test ProbePID with PID 99999 → unhealthy

`cmd/mcpf/status_test.go`:
- Test JSON output schema matches expected structure (mock health probes)
- Test status file is written to correct path

## Constraints

- Health probes must respect per-server `health.timeout`
- `mcpf status` must complete within 30s total (probe servers concurrently
  using goroutines with individual timeouts)
- `go vet ./...` must pass

## Completion gate

Run `go test ./...` — all tests must pass.
Run `just build` — binary compiles.
Smoke test: `./bin/mcpf list` and `./bin/mcpf doctor` run without panicking.
