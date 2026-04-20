# datadog

Datadog MCP server — query metrics, monitors, logs, and events.

## Tools

- `query_metrics` — execute a metrics query
- `list_monitors` — list Datadog monitors
- `get_monitor` — fetch monitor details and status
- `list_dashboards` — list dashboards
- `search_logs` — search log events
- `list_hosts` — list infrastructure hosts
- `get_events` — fetch the event stream

## Transport

- **Transport:** http
- **Port:** 9897
- **Endpoint:** http://localhost:9897/mcp

## Auth

1Password item: `Datadog API Keys` (Private vault)

| Field | Env var |
|-------|---------|
| `api-key` | `DD_API_KEY` |
| `app-key` | `DD_APP_KEY` |

Credentials are cached at `~/.cache/mcp-datadog/`.

Datadog site: `datadoghq.com`

## Prerequisites

Docker must be running.

## Usage

```bash
mcpf start datadog
claude mcp add --transport http datadog http://localhost:9897/mcp
```
