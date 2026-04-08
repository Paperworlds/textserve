# airflow

Airflow MCP server — native Python process that interfaces with AWS MWAA (Managed Apache Airflow).

## Tools

- `list_dags` — list all DAGs in the Airflow environment
- `get_dag` — get DAG details and status
- `trigger_dag` — trigger a DAG run
- `get_dag_run` — get details of a specific DAG run
- `list_dag_runs` — list recent DAG runs
- `get_task_log` — fetch task instance logs

## Transport

- **Transport:** native (Python process)
- **Port:** 9894
- **Endpoint:** http://localhost:9894/mcp

The server runs as a Python process managed by a PID file at `/tmp/.mcp-airflow.pid`.

## Auth

No credential injection — uses AWS SSO via the `paradigm-data.admin` profile, which must be authenticated before starting.

Environment:
- `AWS_PROFILE=paradigm-data.admin`
- `AWS_REGION=us-east-1`
- `MWAA_MCP_READONLY=true`

## Prerequisites

1. **Cloudflare WARP** must be active:
   ```bash
   curl -sf https://api.cloudflare.com/cdn-cgi/trace | grep 'warp=on'
   ```
2. **AWS SSO** must be authenticated:
   ```bash
   aws sso login --profile paradigm-data.admin
   ```
3. The Python virtualenv must exist at:
   `~/projects/personal/skills/locals/docker/mcp-airflow/.venv`

## Usage

```bash
mcpf start airflow
claude mcp add --transport http airflow http://localhost:9894/mcp
```
