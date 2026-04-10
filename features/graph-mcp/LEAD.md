# graph-mcp — Feature Lead

Feature of: mcp-fleet
Plan: `/Users/paulie/.local/projects/graph-mcp.md`

## What this feature builds

An MCP server wrapping a Kuzu knowledge graph. Everything Claude currently loads
upfront into context — memory files, toolmap, skills, feedback rules, references,
bookmarks — becomes connected nodes in the graph, queried on demand instead of
pre-loaded.

End state:
- `graphk ingest` pulls `~/.claude-work/memory/` into graph.yaml → Kuzu DB
- `graph-mcp` MCP server exposes `search`, `query_node`, `query_type`, etc.
- `MEMORY.md` is stubbed to 3 lines — no eager loading
- `pp` queries graph-mcp before spawning workers to derive minimal tool set

## What already exists

Both dependencies are done and unblocked:

**graphk** — Python CLI, uv-installed at `~/.local/bin/graphk`
- Has `sync`, `validate`, `query node/type/relation/why` subcommands
- Kuzu DB at `~/.local/graphk/data/` (check exact path with `graphk --help`)
- Source: find with `uv tool dir graph-roadmap`

**mcpf** — Go CLI at `~/projects/personal/mcp-fleet/bin/mcpf`
- Manages 11 MCP servers via `registry.yaml`
- Supports `native` transport (Python process, PID probe) — same as airflow server
- `mcpf start/stop/health/status` all work

**Memory archive** — `~/.claude-work/memory/`
- Flat .md files with partial/no frontmatter
- These are the input to Phase 6 ingestion

## Phase order and dependency

```
005 → 006 → 007 → 008
```

Each phase depends on the previous completing successfully. Do not run a phase if
its prerequisite check fails — fix the blocker first.

## Constraints

- graphk source must not be broken — add `graph_mcp/` alongside existing `graphk/` package
- Do not modify memory file bodies — frontmatter only
- `mcpf health graph-mcp` is the health gate before proceeding to Phase 7+
- `graphk ingest` must be idempotent — safe to re-run
- All new Python code uses the same Python version as graphk (check `python --version` in graphk venv)
- graph-mcp MCP server uses the `mcp` Python SDK (stdio transport)
- Port 9900 reserved for graph-mcp (confirm no conflict in registry.yaml before assigning)
- Do NOT delete memory files — stub or archive only

## Key file paths

| Resource | Path |
|----------|------|
| Memory archive | `~/.claude-work/memory/` |
| graphk binary | `~/.local/bin/graphk` |
| graphk source | `uv tool dir graph-roadmap` (run to find) |
| Kuzu DB | `~/.local/graphk/data/` |
| mcpf binary | `~/projects/personal/mcp-fleet/bin/mcpf` |
| registry.yaml | `~/projects/personal/mcp-fleet/registry.yaml` |
| graph-mcp server dir | `~/projects/personal/mcp-fleet/servers/graph-mcp/` |
| token reports | `~/projects/personal/mcp-fleet/features/graph-mcp/reports/` |

## How to run phases

From `~/projects/personal/mcp-fleet/features/graph-mcp/`:

```bash
pp run 005   # Phase 6 — archive refactor + graphk ingest
pp run 006   # Phase 7 — graph-mcp server + fleet registration
pp run 007   # Phase 8 — memory stub + token measurement
pp run 008   # Phase 9 — pp tool selection integration
```

Each prompt has a **Prerequisite check** and a **Completion gate** — verify both
before marking a phase done.

## Running pp in the background

Always use the Bash tool's `run_in_background: true` parameter when launching
`pp run <id>` — never use shell `&`. This lets the lead session stay responsive
and get notified when the task completes.

After launching, poll with `pp status` or `pp log <id>` to check progress.
