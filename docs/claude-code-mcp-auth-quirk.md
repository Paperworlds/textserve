# Claude Code HTTP MCP auth quirk

When an HTTP MCP server is registered in `claude.json` but was **not running** when
the Claude Code session opened, Claude Code enters a broken "needs authentication"
state and shows an OAuth flow. The server doesn't actually use OAuth — this is a
false trigger caused by the 404 it received at session startup.

## Symptom

```
Status: △ needs authentication
Error: SDK auth failed: HTTP 404: Invalid OAuth error response: ...
Raw body: 404 page not found
```

## Workaround (already in a session)

1. Start the server: `mcpf start <name>`
2. Open `/mcp` in Claude Code
3. **Disable** the server, then **re-enable** it — no OAuth needed, it connects fine

## Prevention

Start your servers **before** opening the Claude Code session. The server must be
healthy at session startup or Claude Code will latch into the auth error state.

This is the main reason `tw start` (textworkspace) enforces the order:
servers up → session open.
