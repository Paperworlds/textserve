# mcp-fleet — Project Report

## What was built

A standalone Go CLI (`mcpf`) that manages a fleet of 12 local MCP servers from a single entrypoint. Replaces 12 per-server bash scripts and a flat config file with a declarative registry and typed Go packages.

**Binary:** `~/.local/bin/mcpf`
**Registry:** `registry.yaml` (12 servers) + `servers/<name>/server.yaml` (per-server config)
**Module:** `github.com/pdonorio/mcp-fleet` (Go 1.22)

## Use case

**Before:** Starting a data session meant finding the right bash script for each server, manually exporting credentials from 1Password, and hoping the docker flags were still correct. Stopping was manual. No status overview existed.

**After:**

```bash
# Check what's ready before opening Claude
mcpf preflight --tags data --json
# → JSON report: which servers healthy, which blocking (and why)

# Bring up the data stack
mcpf start --tag data
# → fetches creds from 1Password (cached), starts containers, registers with Claude

# Work in Claude — snowflake/airbyte/airflow tools all live

# Tear down
mcpf stop --tag data
# → deregisters from Claude, stops containers
```

`mcpf status` writes `~/.files/states/mcp-fleet.json` on every run — the statusline shows live server count passively.

The preflight JSON output is a hard contract consumed by `knowledge-harvest` to gate sessions on fleet readiness.

## Fleet summary (as of project completion)

| Transport | Count | Servers |
|-----------|-------|---------|
| http (docker) | 10 | jenkins, snowflake, grafana, grafana-pdx, notion, airbyte, slack, slack-search, datadog, paradex-db |
| native (venv) | 1 | airflow |
| stdio (claude) | 1 | sentry |

## What's not done yet

- **Notion**: image build issue unrelated to migration — tracked as follow-up
- **Full cutover**: local fleet servers still need `mcpf start --tag docker` to register in Claude (intentionally deferred — run when ready)
- **Cloud MCPs** (opensea, tldraw, paradex public endpoints): out of scope for mcpf — managed directly by Claude

## Key decisions

| Decision | Choice | Why |
|----------|--------|-----|
| Language | Go | Testable, real YAML parsing, goroutines for concurrent health checks |
| Credentials | Declarative in server.yaml (`op://` URIs) | No bash coupling; Go fetches + caches via `op read` |
| Hook files | Side-effects only (kubectl port-forward) | Not credential injection — clean separation |
| Tests | `go test ./...` | 32 tests across 9 packages; replaces bats |

## Phases completed

| Phase | Description | Tests |
|-------|-------------|-------|
| 1 | Registry scaffold, folder structure | bats (bash baseline) |
| 2 | Go module, server.yaml migration, internal/registry + internal/op | go test |
| 3 | internal/docker, internal/native, internal/claude, cobra CLI | go test |
| 4 | internal/health, mcpf status/health/doctor | go test |
| 5 | internal/preflight, mcpf add, fish completions, docs | go test |

**Total pipeline cost:** ~$13.00 across 5 sonnet runs (Phase 5 ran haiku).

## Metrics at completion

```
go test ./...   → 32 passed, 9 packages
go vet ./...    → clean
just build      → ok
mcpf doctor     → 0 FAIL (WARP-off FAILs expected, 2 servers need WARP)
mcpf preflight  → valid JSON, correct exit codes
```
