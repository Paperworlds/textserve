# mcp-fleet — Lead Prompt

Use this to start a new implementation chat. Paste the contents into a fresh session
pointed at `~/projects/personal/mcp-fleet/`.

---

We are building `mcp-fleet` — a standalone bash CLI (`mcpf`) for managing a fleet
of ~11 MCP servers. The full spec is in `~/.local/projects/mcp-fleet.md`.

## What exists

- Repo scaffolded at `~/projects/personal/mcp-fleet/` (private GitHub: pdonorio/mcp-fleet)
- Phase prompts in `prompts/` define the 5 implementation phases
- Folder structure: `bin/`, `lib/`, `servers/` (one dir per server), `templates/`,
  `completions/`, `tests/`
- All files are currently empty stubs — nothing is implemented yet
- Pipeline registered with `pp` — run `pp status` to see task states

## Source material to read before starting

1. `~/.local/projects/mcp-fleet.md` — full spec: architecture, CLI commands, registry
   schema, preflight JSON contract, implementation plan
2. `~/.config/mcp-servers.conf` — current flat registry to migrate
3. `~/projects/personal/skills/claude-code/memory/mcp-setup.md` — current MCP docs
4. `~/projects/personal/skills/locals/bin/mcp-manage` — docker lifecycle logic to migrate
5. One of `~/projects/personal/skills/locals/bin/mcp-snowflake` — example server wrapper

## Constraints

- No Python. Pure bash + yq + jq.
- All shell scripts must pass shellcheck.
- bats tests for registry parsing and health checks.
- `mcpf` must work as a single entrypoint — no per-server scripts on PATH after migration.

## Start with Phase 1

Run `pp run` to execute the first pending prompt, or read `prompts/000-phase1-scaffold.md`
and implement manually. Phase 1 goal: `registry.yaml` + `lib/registry.sh` + bats tests passing.
