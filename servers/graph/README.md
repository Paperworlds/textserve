# graph-mcp

MCP server exposing the graphk Kuzu knowledge graph as tools for Claude.

## Tools

| Tool | Description |
|------|-------------|
| `query_node` | Return the neighborhood around a node up to depth N |
| `query_type` | Return all nodes of a given type, optionally filtered by status |
| `query_relation` | Return all nodes connected via a given relation |
| `query_why` | Follow the motivated_by chain from a node upward |
| `search` | Keyword search across node names and labels |
| `list_labels` | Return all distinct labels in the graph |

## Transport

Native Python — managed by mcpf via PID file at `/tmp/.mcp-graph-mcp.pid`.

## Source

`graph_mcp/` package lives at `/Users/projects/personal/graph-roadmap/src/graph_mcp/`.
