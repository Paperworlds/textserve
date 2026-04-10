# graph-mcp Testing Guide

Manual testing steps to verify the graph-mcp integration end to end.

## Prerequisites

```bash
mcpf health graph-mcp     # must be healthy
claude mcp list           # graph-mcp must show ✓ Connected
cat ~/.claude-work/memory/MEMORY.md   # must be the 3-line stub
```

## 1. Verify stub is active in a fresh session

Open a new Claude session in any repo. Confirm:
- MEMORY.md loads as 3 lines — not the full index
- No memory files are auto-loaded into context

## 2. Test `search()` for known topics

Ask Claude directly and observe which tools it calls:

| Prompt | Expected tool call | Expected result |
|--------|--------------------|-----------------|
| "What feedback do we have about Go testing?" | `search("go testing")` | `feedback-golang`, `feedback-go-test-flakiness` |
| "What MCP tool should I use to query Snowflake?" | `search("query snowflake")` | toolmap node → snowflake MCP + `feedback-snowflake-accountadmin` |
| "How should I handle 1Password secrets?" | `search("1password secrets")` | `feedback-1password-secrets` |

## 3. Test `query_type()` breadth

Ask: *"What active projects do we have?"*

Claude should call `query_type("project")` and return the full project list.
Verify at least 5 results are returned.

## 4. Test `query_node()` depth

Take a node ID from a search result (e.g. `feedback-golang`) and ask:
*"Tell me more about feedback-golang and what it connects to."*

Claude should call `query_node("feedback-golang", depth=1)` and return
the node plus its neighbors.

## 5. Test the no-match fallback

Ask something not in the graph — a very recent or obscure topic.
Claude should say it didn't find it rather than hallucinating an answer.

## 6. Token measurement

After a few queries, check ai-proxy stats:

```bash
# Compare against Phase 8 baseline: ~89,005 tokens (full context spike)
# Expected: system prompt significantly lower with stub active
```

Baseline from Phase 8 report:
- MEMORY.md index alone: ~456 tokens
- All memory files (worst case): ~77,089 tokens
- After stub: ~84 tokens (−82%)

## Known risks

**Claude reads files directly** — Claude may try to `Read` memory files instead
of using graph-mcp tools. The stub instructs it not to, but if this happens
the MEMORY.md wording may need strengthening.

**Rollback** if anything breaks:
```bash
cp ~/.claude-work/memory/archive/MEMORY.md.bak ~/.claude-work/memory/MEMORY.md
```
