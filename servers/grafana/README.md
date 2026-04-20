# grafana

Grafana MCP server for Paradigmconnect — query dashboards, panels, and metrics.

## Tools

- `list_dashboards` — list available Grafana dashboards
- `get_dashboard` — fetch a dashboard definition
- `query_datasource` — run a metrics query against a configured datasource
- `list_datasources` — list configured data sources
- `get_panel_data` — get rendered panel data

## Transport

- **Transport:** http
- **Port:** 9891
- **Endpoint:** http://localhost:9891/mcp

## Auth

1Password item: `Grafana Paradigmconnect` (Private vault)

| Field | Env var |
|-------|---------|
| `GRAFANA_API_KEY` | `GRAFANA_SERVICE_ACCOUNT_TOKEN` |

Credential is cached at `~/.cache/mcp-grafana/token`.

Grafana URL: `https://paradigmconnect.grafana.net`

## Prerequisites

Docker must be running.

## Usage

```bash
mcpf start grafana
claude mcp add --transport http grafana http://localhost:9891/mcp
```
