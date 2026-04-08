---
id: "000"
title: "Phase 1 — registry + folder scaffold"
phase: "phase-1"
repo: "mcp-fleet"
model: "sonnet"
depends_on: []
budget_usd: 2.00
---

# Phase 1 — Registry & Scaffold

Set up the foundation: `registry.yaml` with all 11 MCP servers migrated from
`~/.config/mcp-servers.conf`, per-server `servers/<name>/server.yaml` files,
and `lib/registry.sh` for parsing.

## Source of truth

Current fleet config lives in `~/.config/mcp-servers.conf` and
`~/projects/personal/skills/claude-code/memory/mcp-setup.md`.

Read both files and extract: server names, ports, transport (docker/stdio/native),
image names, tags, any known deps or auth requirements.

## Tasks

1. Write `registry.yaml` with full schema (see idea file at
   `~/.local/projects/mcp-fleet.md` for schema). Include all 11 servers:
   jenkins, snowflake, grafana, grafana-pdx, slack, slack-search, datadog,
   notion, paradex-db, airbyte, airflow, sentry.

2. Write `servers/<name>/server.yaml` for each server — port, transport,
   image (if docker), tags, deps (empty list if none known yet).

3. Write `lib/registry.sh`:
   - `list_servers` — print all server names from registry.yaml
   - `get_server_field <name> <field>` — extract a single field via yq
   - `filter_by_tag <tag>` — list servers matching a tag
   - Uses `yq` (already installed via brew)

4. Write `tests/test_registry.sh` using bats:
   - Test `list_servers` returns all 11 servers
   - Test `get_server_field jenkins port` returns 9887
   - Test `filter_by_tag docker` returns only docker-transport servers

5. Write `Justfile` with: `test` (runs bats tests), `install` (runs install.sh),
   `lint` (shellcheck on all .sh files).

6. Write `install.sh` — symlinks `bin/mcpf` → `~/.local/bin/mcpf`.

## Constraints

- No Python. Pure bash + yq + jq.
- registry.yaml must be valid YAML parseable by `yq`.
- All shell scripts must pass `shellcheck`.
