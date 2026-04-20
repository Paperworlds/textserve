# sentry

Sentry MCP server — stdio transport managed directly by Claude Code. Provides access to Sentry issues and events.

## Tools

- `get_sentry_issue` — fetch details for a Sentry issue by ID or URL
- `get_sentry_issue_event` — get a specific event (latest/oldest/recommended) for an issue
- `get_sentry_issue_events` — list multiple events for an issue
- `search_issues` — search for Sentry issues with filters

## Transport

- **Transport:** stdio
- **Managed by:** Claude Code (not started by mcpf)

This server is registered and managed directly by Claude Code, not through Docker. `mcpf start sentry` is a no-op.

## Auth

1Password item: `Sentry MCP` (Private vault)

The SENTRY_TOKEN is injected as an environment variable when Claude Code starts the MCP process.

## Prerequisites

- The Sentry MCP server must be registered in `~/.config/claude/claude_desktop_config.json` or equivalent Claude Code config.

## Usage

This server is always-on (user-scope registration). No manual start needed.

To re-register:
```bash
claude mcp add sentry -- npx @sentry/mcp-server@latest --transport stdio
```
