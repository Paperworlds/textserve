---
id: "002"
title: "Phase 3 — status dashboard + health checks"
phase: "phase-3"
repo: "mcp-fleet"
model: "sonnet"
depends_on: ["001"]
budget_usd: 2.50
---

# Phase 3 — Status & Health

Implement `mcpf status` table, health probes, and the `mcpf doctor` diagnostic.

## Tasks

1. Write `lib/health.sh`:
   - `health_http <name>` — `curl -sf http://localhost:<port>/health`
   - `health_toollist <name>` — probe stdio servers by listing tools
   - `health_pid <name>` — check pid file exists and process is alive
   - Dispatcher: picks probe type from server.yaml `health.probe` field
   - Configurable timeout per server

2. Implement `mcpf status`:
   - Table: name | transport | port | status (running/stopped/unhealthy) | uptime
   - `--json` flag: machine-readable output

3. Implement `mcpf health [name]`:
   - Run probes for one or all servers
   - Exit code 0 = all healthy, 1 = any unhealthy
   - Print per-server result

4. Implement `mcpf doctor`:
   - Registry parse check (yq can read it)
   - Image availability (docker pull --dry-run or manifest inspect)
   - Port conflict detection
   - Dep precondition checks (run each server's deps[].cmd)
   - Summary: pass/warn/fail per check

5. Write `tests/test_health.sh` — mock curl responses, verify probe logic and exit codes.

## Integration

- `mcpf status` should write to `~/.files/states/mcp-fleet.json` for statusline integration
  (create parent dirs with `mkdir -p` if needed).
- Format: `{"updated_at": "...", "healthy": N, "total": N, "unhealthy": ["name1"]}`

## Completion gate

Run `just test` before finishing. All bats tests (including test_health.sh) must pass.
