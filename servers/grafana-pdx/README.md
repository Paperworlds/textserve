# grafana-pdx

Grafana MCP server for Paradex — query dashboards, panels, and metrics.

## Tools

- `list_dashboards` — list available Grafana dashboards
- `get_dashboard` — fetch a dashboard definition
- `query_datasource` — run a metrics query against a configured datasource
- `list_datasources` — list configured data sources
- `get_panel_data` — get rendered panel data

## Transport

- **Transport:** http
- **Port:** 9892
- **Endpoint:** http://localhost:9892/mcp

## Auth

1Password item: `op://Private/Grafana Paradex`

| Field | Env var |
|-------|---------|
| `GRAFANA_API_KEY` | `GRAFANA_SERVICE_ACCOUNT_TOKEN` |

Credential is cached at `~/.cache/mcp-grafana-pdx/token`.

Grafana URL: `https://paradex.grafana.net`

## Prerequisites

Docker must be running.

## Usage

```bash
mcpf start grafana-pdx
claude mcp add --transport http grafana-pdx http://localhost:9892/mcp
```
