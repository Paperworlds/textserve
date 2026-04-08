# snowflake

Snowflake MCP server — provides read-only SQL query access to Snowflake.

## Tools

- `execute_query` — run a SELECT or DESCRIBE query
- `list_tables` — list tables in a database/schema
- `describe_table` — get column definitions for a table
- `list_databases` — list accessible databases
- `list_schemas` — list schemas within a database

## Transport

- **Transport:** http
- **Port:** 9890
- **Endpoint:** http://localhost:9890/snowflake-mcp

## Auth

1Password item: `op://Private/REDACTED` (uses item ID to avoid duplicate name ambiguity)

| Field | Env var |
|-------|---------|
| `SNOWFLAKE_ACCOUNT` | `SNOWFLAKE_ACCOUNT` |
| `SNOWFLAKE_USER` | `SNOWFLAKE_USER` |
| `SNOWFLAKE_PASSWORD` | `SNOWFLAKE_PASSWORD` |

Credentials are cached at `~/.cache/mcp-snowflake/`.

## Prerequisites

- Docker must be running.
- A network policy `MCP_LOCAL_POLICY` with `ALLOWED_IP_LIST = ('0.0.0.0/0')` must be applied to your Snowflake user. Docker connects from the actual network IP, not localhost.
- Query permissions are configured via `~/.mcp/snowflake-tools.yaml` (mounted read-only into the container).

## Usage

```bash
mcpf start snowflake
claude mcp add --transport http snowflake http://localhost:9890/snowflake-mcp
```
