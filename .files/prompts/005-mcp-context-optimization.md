# 005 — MCP Context Optimization

## Goal

Audit all MCP servers in this profile and reduce their total token footprint in the
Claude Code context window. The `/context` command shows "MCP tools: Xk tokens" —
target is to get this as low as possible without losing functionality.

## Background reading

Read this first: https://scottspence.com/posts/optimising-mcp-server-context-usage-in-claude-code

Key techniques:
- **Tool consolidation** — merge similar tools into one with a `provider` or `type`
  parameter. Example: 20 tools → 8 tools saved 60% (14k → 5.6k tokens).
- **Description trimming** — one sentence max per tool. No prose, no examples.
- **Selective activation** — disable rarely-used tools by default.

## Steps

1. Find all MCP server configs for this profile:
   - `~/.claude/settings.json` (global)
   - `.claude/settings.json` (project-level, if any)
   - Note which servers are active and how many tools each exposes

2. Run `/context` and record the "MCP tools" baseline token count.

3. For each active MCP server, list its tools with estimated token cost. Focus on:
   - Tools with verbose descriptions (>20 words)
   - Groups of tools doing similar things (candidates for consolidation)
   - Tools that are rarely or never used

4. Propose and implement optimizations:
   - Trim descriptions to one tight sentence
   - Consolidate where the server supports it (check server docs/source)
   - Disable or remove tools not used in this profile

5. Run `/context` again and report before/after MCP token counts.

## Integration with mcpf

If `mcpf` can control which tools a server exposes (via `registry.yaml` or
`servers/<name>/server.yaml`), use that instead of editing server source directly.
Note any `mcpf` feature gaps that would make this easier (e.g. per-profile tool
allow/deny lists) — those are candidates for a future phase.

## Output

Write a report to `.files/reports/005-mcp-context-optimization.md`:
- Before/after token counts per server
- Changes made (description text, tools disabled)
- Recommended `mcpf` features to make this sustainable
