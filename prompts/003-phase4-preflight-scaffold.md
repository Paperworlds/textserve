---
id: "003"
title: "Phase 4 — preflight API + mcpf add scaffolding"
phase: "phase-4"
repo: "mcp-fleet"
model: "sonnet"
depends_on: ["002"]
budget_usd: 2.50
---

# Phase 4 — Preflight & Scaffolding

Implement the `preflight` command (used by knowledge-harvester) and
`mcpf add` for scaffolding new servers.

## Tasks

1. Write `lib/preflight.sh`:
   - `mcpf preflight --tags t1,t2 [--json]`
   - Runs dep checks + health probes for all servers matching tags
   - JSON output schema (see idea file `~/.local/projects/mcp-fleet.md`)
   - Exit code 0 = all ready, 1 = any blocking

2. Implement `mcpf add <name>`:
   - Prompts: transport (docker/stdio/native), port, image (if docker), tags
   - Creates `servers/<name>/` from templates
   - Appends entry to `registry.yaml`
   - Auto-assigns next available port in 9880–9899 range if not specified
   - Prints next steps (write hook.sh if needed, run `mcpf start <name>`)

3. Write `templates/server.yaml.tmpl`, `templates/hook.sh.tmpl`, `templates/README.md.tmpl`.

4. Write `completions/mcpf.fish`:
   - Complete subcommands
   - Complete server names from registry.yaml
   - Complete `--tag` values from known tags

5. Write `tests/test_preflight.sh` — verify JSON schema and exit codes.
   Write `tests/test_scaffold.sh` — verify `mcpf add` creates expected files.

## Constraints

- Preflight JSON must match the schema in the idea file exactly (knowledge-harvester depends on it).
- `mcpf add` must not overwrite existing server dirs — abort with error if `servers/<name>/` exists.
