---
id: "008"
title: "Phase 9 — pp tool selection via graph-mcp"
phase: "phase-9"
repo: "mcp-fleet"
model: "sonnet"
depends_on: ["007"]
budget_usd: 3.00
---

# Phase 9 — pp Tool Selection via graph-mcp

The payoff phase. Before pp spawns a worker agent, it queries graph-mcp to determine
which MCP tools the worker actually needs, then starts only those servers.

Workers get a focused tool surface instead of the full fleet.

## Context

- `pp` is the phase-prompt runner that spawns worker agents
- `mcpf` manages server lifecycle
- `graph-mcp` exposes `search(intent)` which returns toolmap + system nodes
- toolmap nodes have `connections` linking them to system nodes (e.g. snowflake)
- system nodes have `labels` that match mcpf tags (e.g. `data`, `docker`, `monitoring`)

## Prerequisite check

```bash
./bin/mcpf health graph-mcp         # must be healthy
graphk query type toolmap           # must return results (populated in Phase 6)
```

## How pp currently works

Read the pp source to understand its current flow before modifying it. Find pp with:
```bash
which pp
cat $(which pp)
```

pp likely: reads a prompt file, spawns a `claude` subprocess with the prompt,
passes env vars or flags. Understand exactly what happens before adding the
tool-selection step.

## Tasks

### 1. Map toolmap labels to mcpf tags

The connection between graph search results and mcpf is the tag system.

Create `~/.local/mcpf/toolmap-tags.yaml` that maps graph node labels/ids to
mcpf server tags:

```yaml
# Maps graph-mcp node labels → mcpf server tags
# Used by pp to derive which servers to start for a given task
mappings:
  snowflake: [data]
  datadog: [monitoring]
  docker: [docker]
  airflow: [data, docker]
  airbyte: [data, docker]
  slack: [comms]
  github: [ci]
  filesystem: [native]
  memory: [knowledge]
  graph: [knowledge, native]
  fetch: [native]
```

This file is the bridge between the graph world (node labels) and the mcpf world
(server tags). It should be maintained alongside registry.yaml.

### 2. Add tool-selection logic to pp

Modify pp to add a pre-spawn step:

**Before spawning the worker agent:**

1. Read the phase prompt file
2. Extract the first `## Tasks` section or the first 500 chars as the "task intent"
3. Call `graph-mcp search(intent)` via the MCP protocol
   - If graph-mcp is not running: skip tool selection, use default tag set, warn
4. From search results, collect all `system` node labels and `toolmap` node labels
5. Map those labels to mcpf tags using `toolmap-tags.yaml`
6. Call `mcpf start --tag <derived-tags>` — starts only matched servers
7. Spawn the worker agent

**After the worker completes:**
- Optionally call `mcpf stop --tag <derived-tags>` (only servers started by this run)
- Print summary: "Started N servers for this task: [list]"

Implementation detail — calling graph-mcp from pp:
- pp can shell out to a small Python helper, or call the MCP server directly via stdio
- Simplest: write a helper script `~/.local/bin/graph-search`:
  ```bash
  #!/usr/bin/env python3
  # Usage: graph-search "intent string"
  # Prints JSON array of {id, type, name, labels}
  ```
  This helper calls graphk directly (bypasses MCP protocol overhead for pp's use).

### 3. Handle the no-match case

If `search(intent)` returns no results or graph-mcp is unavailable:
- Fall back to `--tag docker` (most common default)
- Print warning: "graph-mcp unavailable — using default tag set"
- Do not block the pp run

### 4. Add --no-tool-select flag

```
pp run <phase> --no-tool-select
```

Skips the graph query and starts nothing. Used when:
- Worker needs no external tools (docs, code editing only)
- Debugging pp itself
- graph-mcp is down

### 5. Measure per-agent tool-description tokens

In ai-proxy logs, tool descriptions appear as part of the system prompt sent to
the model. Measure:

**Before (all servers running):** spawn a worker with full fleet active — record
tool-description token count.

**After (subset):** spawn same worker with only derived tags — record
tool-description token count.

Save to `reports/phase9-tool-selection.md`:
```markdown
# Phase 9 Tool Selection Measurement
Phase prompt: 003-phase4-status-health.md
Task intent: "implement health checks for docker-based MCP servers"
Derived tags: [docker, monitoring]
Servers started: datadog, docker (2 of 11)

Tool description tokens:
- Before (all 11 servers): <N>
- After (2 servers): <N>
- Reduction: <N> tokens (<X>%)
```

### 6. Tests

Write integration tests for the tool-selection logic:

`tests/test_tool_selection.py` (or shell tests):
- `test_search_returns_tags`: given intent "query snowflake data", assert derived tags include `data`
- `test_no_match_falls_back`: given intent with no graph matches, assert falls back to `docker`
- `test_flag_skips_selection`: `pp run <phase> --no-tool-select` starts no servers

## Constraints

- pp must still work if graph-mcp is down (graceful fallback)
- `--no-tool-select` must always be available as escape hatch
- Do not hardcode server names in pp — always go through tags
- toolmap-tags.yaml is the only place the mapping lives

## Completion gate

```bash
# Run a phase with tool selection active:
pp run 003   # Phase 4 — status/health
# Observe: pp prints "Starting servers for task: [docker, monitoring]"
# Worker completes successfully

# Measurement report exists:
cat reports/phase9-tool-selection.md  # shows token reduction

# Fallback works:
./bin/mcpf stop graph-mcp
pp run 003   # should warn but not fail
./bin/mcpf start graph-mcp
```
