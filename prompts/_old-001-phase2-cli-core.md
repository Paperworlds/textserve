---
id: "001"
title: "Phase 2 — CLI core: start/stop/restart/logs/list"
phase: "phase-2"
repo: "mcp-fleet"
model: "sonnet"
depends_on: ["000"]
budget_usd: 3.00
---

# Phase 2 — CLI Core

Implement the `bin/mcpf` entrypoint and the core docker + Claude lifecycle commands.
Migrate logic from `~/projects/personal/skills/locals/bin/mcp-manage` and
the per-server `mcp-<name>` wrappers.

## Tasks

1. Write `lib/docker.sh`:
   - `docker_run <name>` — start container using registry fields (image, port, container_port,
     env, volumes). Before calling `docker run`, source `servers/<name>/hook.sh` if it exists —
     this sets EXTRA_DOCKER_FLAGS, EXTRA_CMD_ARGS, MCP_ENDPOINT_PATH (same pattern as
     `skills/locals/bin/mcp-manage`). Container name is always `mcp-<name>`.
   - `docker_stop <name>` — `docker rm -f mcp-<name>`
   - `docker_status <name>` — running/stopped/unknown via `docker inspect`
   - `docker_logs <name> [-f]` — tail logs

2. Write `lib/claude.sh`:
   - `claude_register <name>` — `claude mcp add --transport http <name> http://localhost:<port><endpoint_path>`
     where `endpoint_path` comes from `get_server_field <name> endpoint_path` in registry.
     Endpoint paths vary per server: snowflake uses `/snowflake-mcp`, paradex-db uses `/sse`,
     most others use `/mcp`. Never hardcode — always read from registry.
   - `claude_deregister <name>` — `claude mcp remove <name>`
   - stdio servers (`managed_by: claude`): `start` prints an info message and exits 0 (no-op)

3. Write `bin/mcpf` dispatcher:
   - Subcommands: `start`, `stop`, `restart`, `logs`, `list`
   - `start <name|--tag t>` — runs docker_run + claude_register
   - `stop <name|--tag t>` — runs docker_stop + claude_deregister
   - `restart` — stop + start
   - `logs <name> [-f]` — streams container logs
   - `list [--tag t]` — lists servers from registry, optionally filtered

4. Native server support (airflow):
   - `transport: native` servers skip docker entirely
   - `start`: source `servers/airflow/hook.sh`, exec the native process in background,
     write PID to `pid_file` from registry
   - `stop`: kill PID from `pid_file`, remove pid file
   - `status`: check pid file exists and process is alive

5. Replace `skills/locals/bin/mcp-<name>` wrappers with one-liner shims:
   ```bash
   #!/usr/bin/env bash
   exec mcpf start <name> "$@"
   ```
   Create shims for all 12 servers (including sentry — even though it's a no-op, the shim
   provides a consistent interface).

6. Write `tests/test_cli.sh` — mock docker commands (override with bash functions in test
   scope), verify start/stop/list output and exit codes.

## Constraints

- `mcpf start --tag docker` must start all docker-transport servers in dependency order
- stdio servers (sentry): `start` is a no-op with an informational message, exit 0
- Native servers (airflow): use pid file, not docker
- All scripts must pass `shellcheck`

## Completion gate

Run `just test` before finishing. All bats tests (including test_cli.sh) must pass.
