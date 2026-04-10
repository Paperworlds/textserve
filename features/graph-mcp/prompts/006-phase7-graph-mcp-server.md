---
id: "006"
title: "Phase 7 — graph-mcp server + fleet registration"
phase: "phase-7"
repo: "mcp-fleet"
model: "sonnet"
depends_on: ["005"]
budget_usd: 3.50
---

# Phase 7 — graph-mcp Server + Fleet Registration

Build the graph-mcp MCP server and register it in the fleet. This server exposes
graphk's Kuzu backend as MCP tools — Claude queries it on demand instead of loading
everything into context upfront.

## Context

- graphk CLI: `~/.local/bin/graphk` — has `query node/type/relation/why` subcommands
- Kuzu DB: `~/.local/graphk/data/company.db`
- graphk source: `/Users/projects/personal/graph-roadmap/src/graph_roadmap/` (editable install via `.pth`)
- Transport: native Python (not Docker), managed by mcpf via PID file
- mcp-fleet repo: `~/projects/personal/mcp-fleet`

## Tasks

### 1. Create servers/graph-mcp/ directory

```
servers/graph-mcp/
  server.yaml
  README.md
```

`servers/graph-mcp/server.yaml`:
```yaml
transport: native
command: python
args: ["-m", "graph_mcp"]
working_dir: "~/.local/share/uv/tools/graph-roadmap"  # adjust to actual graphk source path
port: 9900
tags: [native, knowledge, memory]
health:
  probe: pid
deps: []
env: []
```

Adjust `working_dir` to the actual graphk source location (`/Users/projects/personal/graph-roadmap/src/graph_roadmap/`).
Port 9900 is reserved for graph-mcp — check `registry.yaml` for conflicts and use next available if needed.

### 2. Add graph-mcp to registry.yaml

Append to `registry.yaml`:
```yaml
- name: graph-mcp
  path: servers/graph-mcp/server.yaml
  tags: [native, knowledge, memory]
```

### 3. Implement graph_mcp Python module

Create `graph_mcp/` package inside the graphk source tree (alongside existing `graphk/` package):

```
graph_mcp/
  __init__.py
  server.py      # MCP server entry point
  tools.py       # Tool implementations
```

Use the `mcp` Python SDK (`pip install mcp` or add to graphk's dependencies).

**`server.py`** — MCP server entry point:
```python
from mcp.server.stdio import stdio_server
from mcp import Server
from .tools import register_tools

app = Server("graph-mcp")
register_tools(app)

if __name__ == "__main__":
    import asyncio
    asyncio.run(stdio_server(app))
```

**`tools.py`** — implement these MCP tools:

| Tool | Signature | Maps to |
|------|-----------|---------|
| `query_node` | `(id: str, depth: int = 1)` | `graphk query node <id>` |
| `query_type` | `(type: str, status: str = "active")` | `graphk query type <type>` |
| `query_relation` | `(id: str, relation: str, direction: str = "out")` | `graphk query relation <id> <rel>` |
| `query_why` | `(id: str)` | `graphk query why <id>` |
| `search` | `(intent: str)` | keyword search across node names + labels |
| `list_labels` | `()` | return all distinct labels in the graph |

Implementation strategy:
- Shell out to `graphk query ...` subcommands and parse stdout (simplest, most robust)
- Parse JSON output if `graphk` supports `--json` flag; otherwise parse text
- Each tool returns a dict (MCP will serialize to JSON)
- On error: return `{"error": "<message>"}` rather than raising

`search(intent)` implementation:
- Tokenize `intent` into words
- Query Kuzu directly (or via `graphk`) for nodes where `name` or `labels` contain any token
- Return list of `{id, type, name, labels}` — top 10 by relevance (label match > name match)
- This is the key tool pp uses for tool selection

### 4. Add `__main__.py`

```python
# graph_mcp/__main__.py
from .server import main
main()
```

So `python -m graph_mcp` works.

### 5. Add mcp dependency to graphk

In the graphk project's `pyproject.toml`, add `mcp>=1.0.0` to dependencies.
Run `uv sync` inside the graphk source dir to install it.

### 6. Test mcpf integration

```bash
cd ~/projects/personal/mcp-fleet
./bin/mcpf start graph-mcp
./bin/mcpf health graph-mcp     # must be healthy
./bin/mcpf status               # graph-mcp appears in list
```

### 7. Tests

`graph_mcp/tests/test_tools.py`:
- `test_query_node_returns_dict`: call `query_node("some-existing-id")` — returns dict with `id` key
- `test_query_type_returns_list`: call `query_type("feedback")` — returns list
- `test_search_returns_results`: call `search("snowflake")` — returns non-empty list
- `test_search_empty_intent`: call `search("")` — returns empty list, no error
- `test_query_node_unknown_id`: call `query_node("nonexistent-xyz")` — returns `{"error": ...}` not exception

## Constraints

- Server must start via `python -m graph_mcp` from the graphk source dir
- All tools must return JSON-serializable dicts or lists
- `mcpf health graph-mcp` must pass (PID file exists and process is alive)
- Do not modify existing graphk query commands — only add the new MCP layer on top

## Completion gate

```bash
./bin/mcpf start graph-mcp
./bin/mcpf health graph-mcp     # exit 0
python -m pytest graph_mcp/tests/  # all pass
# In a separate terminal, test via MCP protocol:
echo '{"method":"tools/list"}' | python -m graph_mcp  # returns tool list
```
