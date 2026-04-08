# slack-search

Slack Search MCP server — full-text search across Slack messages using a user token.

## Tools

- `search_messages` — search messages across all Slack channels
- `search_files` — search files shared in Slack

## Transport

- **Transport:** http
- **Port:** 9896
- **Endpoint:** http://localhost:9896/mcp

## Auth

1Password item: `op://Private/Slack Bot Token`

| Field | Env var |
|-------|---------|
| `user-token` | `SLACK_USER_TOKEN` |

Credential is cached at `~/.cache/mcp-slack-search/user-token`.

Uses a **user token** (not bot token) for broader search access.

## Prerequisites

Docker must be running.

## Usage

```bash
mcpf start slack-search
claude mcp add --transport http slack-search http://localhost:9896/mcp
```
