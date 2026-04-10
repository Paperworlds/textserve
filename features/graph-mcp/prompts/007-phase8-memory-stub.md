---
id: "007"
title: "Phase 8 — replace flat memory loading with graph-mcp stub"
phase: "phase-8"
repo: "mcp-fleet"
model: "sonnet"
depends_on: ["006"]
budget_usd: 2.00
---

# Phase 8 — Replace Flat Memory Loading

graph-mcp is running. Now replace the eagerly-loaded memory files with a minimal
stub that directs Claude to query the graph on demand.

## Context

Currently Claude's system prompt loads at session start:
- `~/.claude-work/memory/MEMORY.md` — full index + linked memory files
- `toolmap.yaml` — 11K YAML of intent-to-tool mappings
- Skills files, bookmarks, pointers

All loaded regardless of relevance. This phase stubs them out and measures the
token saving via ai-proxy.

## Prerequisite check

Before starting:
```bash
./bin/mcpf health graph   # must be healthy
graphk query type feedback     # must return results
```

If either fails, stop — Phase 7 is not complete.

## Tasks

### 1. Measure baseline token cost

Before making any changes, record the current token cost of a fresh session:

```bash
# Open a new Claude session and immediately run /status or ask a trivial question.
# Check ai-proxy logs for system prompt token count.
# Record: baseline_system_prompt_tokens = <N>
```

Save the measurement to `reports/phase8-token-baseline.md`:
```markdown
# Phase 8 Token Baseline
Date: <today>
System prompt tokens (before stub): <N>
Files contributing:
- MEMORY.md index: ~<N> tokens
- Linked memory files: ~<N> tokens
- toolmap.yaml: ~<N> tokens
- Other: ~<N> tokens
```

### 2. Stub MEMORY.md

Replace `~/.claude-work/memory/MEMORY.md` with:

```markdown
# Memory

All knowledge is in the graph. Use graph-mcp tools to query on demand.

- `search(intent)` — find relevant nodes by keyword
- `query_node(id)` — get a node and its neighbors
- `query_type(type)` — list all nodes of a type (feedback, reference, project, etc.)

Do not load memory files directly. Query the graph instead.
```

**Before replacing**: copy the original to `~/.claude-work/memory/archive/MEMORY.md.bak`
so it can be restored if needed.

### 3. Remove eager toolmap loading

Locate where `toolmap.yaml` is loaded into Claude's context (check
`~/.claude-work/settings.json` or equivalent config). Comment it out or remove
the include.

If toolmap.yaml is referenced in MEMORY.md (now stubbed), that's already handled.
If it's loaded via a separate mechanism, disable that mechanism.

### 4. Remove eager skills/bookmarks loading

Same as above — locate and disable any config that pre-loads:
- Skills files (`~/.claude/skills/*.md`)
- Bookmark files
- Other flat knowledge files

Keep the files themselves — only disable the eager loading.

### 5. Measure post-stub token cost

Open a new Claude session and record system prompt tokens again:
```bash
# Same method as step 1
```

Append to `reports/phase8-token-baseline.md`:
```markdown
# Phase 8 Token After Stub
System prompt tokens (after stub): <N>
Reduction: <N> tokens (<X>%)
```

### 6. Smoke test graph-mcp is usable

In the new session (with stubbed memory), verify Claude can retrieve knowledge:
- Ask: "What's the feedback rule about database tests?" — Claude should call `search("database tests")` and return the relevant feedback node
- Ask: "What MCP tool should I use to query Snowflake?" — Claude should call `search("query snowflake")` and return the toolmap node

If Claude can't find information that used to be in memory, check if that file
had proper frontmatter and was ingested in Phase 6.

### 7. Rollback procedure

If something breaks badly, restore:
```bash
cp ~/.claude-work/memory/archive/MEMORY.md.bak ~/.claude-work/memory/MEMORY.md
# re-enable any disabled config
```

Document the rollback steps in `reports/phase8-token-baseline.md`.

## Constraints

- Do NOT delete any memory files — only stub the index and disable eager loading
- Original MEMORY.md must be backed up before replacement
- graph-mcp must be running and healthy before stubbing
- Rollback must be possible with a single command

## Completion gate

```bash
# New Claude session opens with stubbed memory (MEMORY.md = 3 lines)
# graph-mcp is healthy
./bin/mcpf health graph

# Claude can retrieve knowledge on demand:
# "What feedback do we have about Go testing?" → search returns results

# Token report exists:
cat reports/phase8-token-baseline.md  # shows before + after + reduction %
```
