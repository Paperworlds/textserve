# graph-mcp Testing Guide

Manual testing steps to verify the graph-mcp integration end to end.

## Prerequisites

```bash
# 1. Server must be installed and healthy
mcpf health graph         # must show: ✓ graph healthy
claude mcp list           # graph must show ✓ Connected

# 2. Memory stub must be active
cat ~/.claude-work/memory/MEMORY.md   # must be the 3-line stub (see step 0)
```

## 0. Ensure MEMORY.md is the 3-line stub

The stub replaces the old eager-loading index. It should contain only:

```markdown
# Memory

Context lives in the knowledge graph. Use the `graph` MCP tools to query it:
- `search(intent)` — find nodes by keyword/topic
- `query_type(type)` — list all nodes of a type (project, feedback, system, goal…)
- `query_node(id)` — explore a node and its neighbors
```

If the file is longer than this, replace it with the stub above and verify
that none of the old memory files are auto-loaded into context on the next
session start.

**Rollback** if anything breaks:
```bash
cp ~/.claude-work/memory/archive/MEMORY.md.bak ~/.claude-work/memory/MEMORY.md
```

## 1. Start and verify the server

```bash
mcpf start graph          # registers + starts the stdio server
mcpf health graph         # ✓ graph healthy
```

If already registered but unhealthy:
```bash
mcpf restart graph
mcpf health graph
```

Open a **new** Claude session after starting — the MCP connection is established
at session init, not mid-conversation.

## 2. Verify stub is active in a fresh session

Open a new Claude session in any repo. Confirm:
- MEMORY.md loads as 3 lines — not the full index
- No memory files are auto-loaded into context

## 3. Test `search()` — nodes and edges

Ask Claude directly and observe which tools it calls:

| Prompt | Expected tool call | Expected result |
|--------|--------------------|-----------------|
| "What feedback do we have about Go testing?" | `search("go testing")` | `feedback-golang`, `feedback-go-test-flakiness` |
| "What MCP tool should I use to query Snowflake?" | `search("query snowflake")` | toolmap node → snowflake MCP + `feedback-snowflake-accountadmin` |
| "How should I handle 1Password secrets?" | `search("1password secrets")` | `feedback-1password-secrets` |

**Verify edges are returned**: the result should contain both `nodes` and `edges`
keys, not just a flat node list. If edges are missing, re-run `graphk sync` to
refresh the DB, then restart the server.

## 4. Test `query_type()` — breadth with edges

Ask: *"What active projects do we have?"*

Claude should call `query_type("project")` and return nodes plus edges between
them. Verify:
- At least 5 node results
- `edges` key is present in the response (may be empty if nodes are unconnected)

## 5. Test `query_node()` depth

Take a node ID from a search result (e.g. `feedback-golang`) and ask:
*"Tell me more about feedback-golang and what it connects to."*

Claude should call `query_node("feedback-golang", depth=1)` and return
the node plus its neighbors and edges.

## 6. Test proactive graph usage

Ask something that requires project context **without** mentioning the graph:

> "What are we currently working on?"
> "What system handles alert triage?"

Claude should reach for `search()` or `query_type()` on its own — not wait to
be asked. If it doesn't, the MEMORY.md stub wording may need a stronger nudge.

## 7. Test the no-match fallback

Ask something not in the graph — a very recent or obscure topic.
Claude should say it didn't find it rather than hallucinating an answer.

## 8. Token measurement

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

**`mcpf start/restart` modifies `.claude.json`** — these commands write to
`~/.claude-work/.claude.json`. The change only adds/updates the `graph` entry
and does not affect other registered MCPs. Verify with:
```bash
cat ~/.claude-work/.claude.json | python3 -c "
import json,sys; d=json.load(sys.stdin)
for k,v in d.get('mcpServers',{}).items(): print(k)
"
```

**DB out of sync** — if `graph.yaml` was edited but `graphk sync` wasn't re-run,
the graph tools will return stale data. Always sync after editing the YAML:
```bash
graphk sync
mcpf restart graph
```
