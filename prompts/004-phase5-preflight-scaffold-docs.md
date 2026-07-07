---
id: "004"
title: "Phase 5 — preflight API + mcpf add + completions + docs"
phase: "phase-5"
repo: "mcp-fleet"
model: haiku
model: "sonnet"
depends_on: ["003"]
budget_usd: 2.50
---

# Phase 5 — Preflight, Scaffolding, Completions & Docs

Final phase. Implement the preflight command (used by knowledge-harvest), the
`mcpf add` scaffolding command, fish completions, and all documentation.

## Tasks

### 1. internal/preflight package

`internal/preflight/preflight.go`:

**`Run(tags []string, repoRoot string) (*Report, error)`**
- Filter servers by tags (union — any matching tag included)
- For each matched server:
  1. Run dep checks (`internal/deps`)
  2. Run health probe (`internal/health`)
  3. Collect result
- Return `Report` struct

**Report JSON schema** (must match exactly — knowledge-harvest depends on it):
```json
{
  "timestamp": "2026-04-08T14:30:00Z",
  "requested_tags": ["data", "monitoring"],
  "servers": {
    "datadog":      {"status": "healthy",   "port": 9897, "tools": 20},
    "airbyte":      {"status": "unhealthy", "error": "dep_failed: kubectl port-forward not active"},
    "airflow":      {"status": "stopped"}
  },
  "ready": false,
  "blocking": ["airbyte"]
}
```

- `ready: true` only if all matched servers are healthy
- `blocking`: servers that are unhealthy or dep-failed
- `tools` field: omit if unknown (probe didn't return tool count)

**`mcpf preflight --tags t1,t2 [--json]`**
- Default output: human-readable summary
- `--json`: emit Report JSON to stdout
- Exit code: 0 if `ready: true`, 1 otherwise

### 2. mcpf add

Interactive scaffolding for new servers. No interactive prompts — all inputs via flags:

```
mcpf add <name> --transport http --port 9899 --image my-image [--tags ci,docker]
```

Steps:
1. Abort if `servers/<name>/` already exists
2. Auto-assign next available port in 9880–9899 if `--port` not given
3. Create `servers/<name>/` directory
4. Write `servers/<name>/server.yaml` from template
5. Write `servers/<name>/hook.sh` stub from template (chmod +x)
6. Write `servers/<name>/README.md` stub from template
7. Append entry to `registry.yaml`
8. Print next steps

### 3. Update templates/

Rewrite template files for Go-style server.yaml schema:

`templates/server.yaml.tmpl`:
```yaml
image: "{{.Image}}"
transport: {{.Transport}}
port: {{.Port}}
container_port: {{.Port}}
endpoint_path: /mcp
tags: [{{.TagsCSV}}]
env: []
volumes: []
extra_args: []
deps: []
health:
  endpoint: /health
  timeout: 5
```

`templates/hook.sh.tmpl` — now just a pre-start side-effect script stub:
```bash
#!/usr/bin/env bash
# pre-start hook for {{.Name}}
# Use for side effects only (port-forwards, precondition checks).
# Credential injection belongs in server.yaml env[] entries.
```

`templates/README.md.tmpl` — standard per-server README skeleton.

### 4. completions/mcpf.fish

Fish shell completions:
- Complete subcommands: start stop restart logs list status health doctor preflight add
- Complete server names (from `mcpf list`) for: start stop restart logs health
- Complete `--tag` values: ci docker data monitoring comms native stdio
- Complete `--tags` for preflight (comma-separated, same values)

### 5. Per-server README.md

Write `servers/<name>/README.md` for all 12 servers. Each must include:
- What the server does / what tools it exposes (list them)
- Transport + port
- Auth setup (which 1Password item, which fields)
- Any deps or preconditions
- Example `mcpf start <name>` and registration URL

Source: `~/projects/personal/skills/claude-code/memory/mcp-setup.md` has
detailed notes for each server.

### 6. Top-level README.md

Write `README.md`:
- What mcp-fleet is (one paragraph)
- Quick start: `just install`, `mcpf status`, `mcpf start <name>`
- Full CLI reference table
- Registry schema reference
- How to add a new server (`mcpf add`)
- How to rotate credentials (clear `~/.cache/mcp-<name>/`)

### 7. Update skills repo memory

Update `~/projects/personal/skills/claude-code/memory/mcp-setup.md`:
- Add note at top: "mcp-fleet is now the source of truth. See ~/projects/personal/mcp-fleet/"
- Update "File Locations" table to point at mcp-fleet paths
- Keep all existing content (don't delete)

### 8. Deprecation checklist

Do NOT delete any skills files. Instead append a checklist to the top-level
`README.md` under a `## Migration` section:

Files safe to remove after 2-week shim period:
- `~/projects/personal/skills/locals/mcp-hooks/*.sh` (all 12)
- `~/projects/personal/skills/locals/bin/mcp-<name>` wrappers (all 12)
- `~/.config/mcp-servers.conf` (replaced by registry.yaml)

### 9. Tests

`internal/preflight/preflight_test.go`:
- Test Report JSON serializes to expected schema
- Test ready=false when any server is unhealthy
- Test blocking list contains correct server names

`cmd/mcpf/add_test.go`:
- Test mcpf add creates expected files
- Test mcpf add aborts if servers/<name>/ exists

## Constraints

- Preflight JSON schema must be exact — no extra or missing top-level keys
- `mcpf add` must not overwrite existing server dirs
- `go vet ./...` must pass

## Completion gate

Run `go test ./...` — all tests must pass.
Run `just build` — binary compiles.
Run `./bin/mcpf preflight --tags docker --json` — valid JSON output (servers may
be stopped, that's fine).
Run `./bin/mcpf doctor` — completes without panic.
