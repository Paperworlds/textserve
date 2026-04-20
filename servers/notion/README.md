# notion

Notion MCP server — read and write Notion pages, databases, and blocks.

## Tools

- `search` — search across Notion workspace
- `get_page` — fetch a Notion page
- `create_page` — create a new page in a database or as a child
- `update_page` — update page properties
- `get_database` — fetch a database schema
- `query_database` — query rows in a database with filters
- `append_block` — append content blocks to a page

## Transport

- **Transport:** http
- **Port:** 9888
- **Endpoint:** http://localhost:9888/mcp

## Auth

1Password item: `Notion MCP` (Private vault)

| Field | Env var |
|-------|---------|
| `INTERNAL_INTEGRATION_TOKEN` | `NOTION_TOKEN` |

Credential is cached at `~/.cache/mcp-notion/token`.

The token is also injected into `OPENAPI_MCP_HEADERS` as a Bearer auth header with Notion-Version `2022-06-28`.

## Prerequisites

Docker must be running.

## Usage

```bash
mcpf start notion
claude mcp add --transport http notion http://localhost:9888/mcp
```
