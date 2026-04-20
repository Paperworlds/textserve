# paradex-db

Paradex DB MCP server — read-only access to the Paradex production PostgreSQL database via SSE transport.

## Tools

- `execute_query` — run a read-only SQL query
- `list_tables` — list tables in the paradex database
- `describe_table` — describe a table's schema

## Transport

- **Transport:** http (SSE mode)
- **Port:** 9898
- **Endpoint:** http://localhost:9898/sse

Note: This server uses SSE endpoint path (`/sse`), not `/mcp`.

## Auth

1Password item: `Paradex DB` (Private vault)

Database credentials are cached via `cache_file` entries:
- `~/.cache/paradex-db-mcp/host` — DB host
- `~/.cache/paradex-db-mcp/user` — DB user

AWS credentials are mounted from `~/.aws/config` and `~/.aws/sso` (read-only, symlinks resolved).

AWS profile: `paradex-prod.basic`, region: `ap-northeast-1`

## Prerequisites

1. **Cloudflare WARP** must be active — the database is only accessible from a trusted network.
   ```bash
   curl -sf https://api.cloudflare.com/cdn-cgi/trace | grep 'warp=on'
   ```
2. AWS SSO must be authenticated for the `paradex-prod.basic` profile.

## Usage

```bash
mcpf start paradex-db
claude mcp add --transport http paradex-db http://localhost:9898/sse
```
