# slack

Slack MCP server — read channels, send messages, and interact with Slack.

## Tools

- `list_channels` — list public channels
- `get_channel_history` — fetch recent messages from a channel
- `post_message` — post a message to a channel
- `reply_to_thread` — reply to an existing thread
- `get_user` — look up a Slack user
- `list_users` — list workspace members

## Transport

- **Transport:** http
- **Port:** 9895
- **Endpoint:** http://localhost:9895/mcp

## Auth

1Password item: `Slack Bot Token` (Private vault)

| Field | Env var |
|-------|---------|
| `token` | `SLACK_BOT_TOKEN` |

Credential is cached at `~/.cache/mcp-slack/bot-token`.

Team ID: `T072ZU3U7`

## Prerequisites

Docker must be running. Slack bot token must have appropriate scopes (channels:history, chat:write, users:read, etc.).

## Usage

```bash
mcpf start slack
claude mcp add --transport http slack http://localhost:9895/mcp
```
