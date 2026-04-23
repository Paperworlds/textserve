# airbyte

Airbyte MCP server — manage data pipeline connections, sources, and destinations.

## Tools

- `list_connections` — list all Airbyte connections
- `get_connection` — get details for a connection
- `trigger_sync` — trigger a manual sync for a connection
- `get_job_status` — check the status of a sync job
- `list_sources` — list configured sources
- `list_destinations` — list configured destinations

## Transport

- **Transport:** http
- **Port:** 9893
- **Endpoint:** http://localhost:9893/mcp

## Auth

No credentials required (internal API).

The server connects to Airbyte at `http://host.docker.internal:18001`.

## Prerequisites

1. Cloudflare WARP or cluster access must be active.
2. kubectl port-forward must be running:
   ```bash
   kubectl port-forward svc/airbyte-helmv2-airbyte-server-svc 18001:8001 -n airbyte-helmv2
   ```

The `pre-start.sh` hook handles this automatically when `mcpf start airbyte` is run.

## Usage

```bash
mcpf start airbyte
claude mcp add --transport http airbyte http://localhost:9893/mcp
```
