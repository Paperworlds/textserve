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
   - `docker_run <name>` — start container from server.yaml config (image, port, env, volumes)
   - `docker_stop <name>` — stop and remove container
   - `docker_status <name>` — running/stopped/unknown
   - `docker_logs <name> [-f]` — tail logs

2. Write `lib/claude.sh`:
   - `claude_register <name>` — `claude mcp add --transport http <name> http://localhost:<port>`
   - `claude_deregister <name>` — `claude mcp remove <name>`
   - Handle stdio servers (sentry): `managed_by: claude` flag skips docker ops

3. Write `bin/mcpf` dispatcher:
   - Subcommands: `start`, `stop`, `restart`, `logs`, `list`
   - `start <name|--tag t>` — runs docker_run + claude_register
   - `stop <name|--tag t>` — runs docker_stop + claude_deregister
   - `restart` — stop + start
   - `logs <name> [-f]` — streams container logs
   - `list [--tag t]` — lists servers from registry, optionally filtered

4. Replace `skills/locals/bin/mcp-<name>` wrappers with one-liner shims:
   `#!/usr/bin/env bash\nexec mcpf start <name> "$@"`
   Create shims for all 11 docker-managed servers.

5. Write `tests/test_cli.sh` — mock docker commands, verify start/stop/list output.

## Constraints

- `mcpf start --tag docker` must start all docker-transport servers in dependency order
- stdio servers (sentry): `start` is a no-op with an informational message
- Native servers (airflow): use pid file, not docker
