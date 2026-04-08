# jenkins

CI/CD MCP server — exposes Jenkins build management tools via the MCP protocol.

## Tools

- `list_jobs` — list all Jenkins jobs
- `get_job` — get details for a specific job
- `build_job` — trigger a build
- `get_build` — get build details and status
- `get_build_log` — fetch console log for a build
- `list_builds` — list recent builds for a job
- `abort_build` — abort a running build

## Transport

- **Transport:** http
- **Port:** 9887
- **Endpoint:** http://localhost:9887/mcp

## Auth

1Password item: `op://Private/CI Dev Secrets`

| Field | Env var |
|-------|---------|
| `JENKINS_USER` | `JENKINS_USERNAME` |
| `JENKINS_API_TOKEN` | `JENKINS_TOKEN` |

Credentials are cached at `~/.cache/mcp-jenkins/`.

## Prerequisites

None. Docker must be running.

## Usage

```bash
mcpf start jenkins
claude mcp add --transport http jenkins http://localhost:9887/mcp
```
