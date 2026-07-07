---
id: "005"
title: "Phase 6 — archive refactor + graphk ingest pipeline"
phase: "phase-6"
repo: "mcp-fleet"
model: "sonnet"
depends_on: []
budget_usd: 3.00
---

# Phase 6 — Archive Refactor + Ingestion Pipeline

First phase of graph-mcp. Prepare the memory archive so it can be ingested into
the Kuzu graph, then add a `graphk ingest` command that parses frontmatter and
writes nodes to graph.yaml.

The archive lives at `~/.claude-work/memory/`. It is the authoritative source —
human-editable, git-tracked. The graph is a derived index built from it.

## Context

- `graphk` is a uv-installed Python CLI at `~/.local/bin/graphk`
- Its source is at `/Users/projects/personal/graph-roadmap/src/graph_roadmap/` (editable install)
- graph.yaml lives at `~/.local/graphk/data/graph.yaml` (or wherever `graphk` stores it)
- Kuzu DB is at `~/.local/graphk/data/company.db`
- `graphk sync` rebuilds the Kuzu DB from graph.yaml

## Tasks

### 1. Audit memory files

Inspect all files in `~/.claude-work/memory/`:
- List every file and its current frontmatter (if any)
- Note which files already have `type:`, `labels:`, `description:`, `connections:` fields
- Note which are missing frontmatter entirely

### 2. Add frontmatter to all memory files

Each file needs consistent frontmatter so the ingestion pipeline can parse it:

```yaml
---
type: feedback          # feedback | project | reference | skill | user | toolmap | bookmark
labels: [go, testing]
description: "one-line hook used in graph search results"
connections:
  - to: <node-id>
    relation: applies_to | motivated_by | uses | informs
---
```

Rules:
- `type` must be one of the enum values above
- `description` is a single sentence, max 120 chars — this is what appears in `search()` results
- `labels` drive label-based search; reuse existing labels from graph.yaml where possible
- `connections` is optional but add where obvious (e.g. a snowflake feedback file connects to the snowflake system node)
- Do NOT alter the body content of any file — only add/update frontmatter
- Files in `~/.claude-work/memory/archive/` should be skipped (excluded from ingestion)

### 3. Extend graph.yaml schema

Add new node types to support the memory archive. The existing schema has `system`,
`problem`, `goal`, `agent`, `decision`. Add:

| type     | description                                             |
|----------|---------------------------------------------------------|
| feedback | A learned rule about how to work — source of truth for behavior |
| reference | Pointer to an external resource (URL, path, system)  |
| project  | An ongoing initiative with context, deadline, motivation |
| user     | Facts about a user's role, expertise, preferences      |
| skill    | A reusable task template (slash command equivalent)    |
| toolmap  | An intent-to-tool mapping ("query snowflake" → snowflake MCP) |
| bookmark | A URL or path worth remembering with context           |

Add these as valid `type` values in graph.yaml's schema section. Do not remove
existing types.

### 4. Add `graphk ingest <dir>` command

Add a subcommand to the graphk CLI:

```
graphk ingest <dir>
```

Behavior:
1. Walk `<dir>` recursively, skip `archive/` subdirectory and dotfiles
2. For each `.md` file, parse YAML frontmatter (between `---` delimiters)
3. If frontmatter has `type:` and `description:`, create or update a node in graph.yaml:
   - `id`: derived from filename (strip extension, replace `_` with `-`)
   - `type`: from frontmatter
   - `name`: from frontmatter `description` (used as display name)
   - `labels`: from frontmatter `labels` array
   - `status`: `active` (default for all ingested nodes)
   - `connections`: from frontmatter `connections` array
4. Files without valid frontmatter: print warning, skip
5. Print summary: "Ingested N nodes, skipped M files"

Implementation notes:
- Use Python's `yaml` library (already a dep of graphk)
- Idempotent: re-running ingest on the same dir updates existing nodes, does not duplicate
- Node identity: if a node with the same `id` exists in graph.yaml, update its fields
- Do not remove nodes that are no longer present in the archive (use `status: archived` instead if needed)

### 5. Run and verify

```bash
graphk ingest ~/.claude-work/memory/
graphk sync
```

Verify:
- `graphk query type feedback` returns at least 3 nodes
- `graphk query type reference` returns at least 3 nodes
- `graphk query node <some-feedback-id>` returns full node detail
- No errors in `graphk sync` output

### 6. Tests

In the graphk source, add tests for the ingest command:
- `test_ingest_creates_node_from_frontmatter`: given a tmp dir with one .md file
  with valid frontmatter, assert the node is written to graph.yaml
- `test_ingest_skips_archive_dir`: files under `archive/` are not ingested
- `test_ingest_idempotent`: running ingest twice on the same dir produces the same
  node count (no duplicates)
- `test_ingest_skips_missing_type`: .md files without `type:` in frontmatter are
  skipped with a warning

## Constraints

- Do NOT modify body content of any memory file — frontmatter only
- `graphk ingest` must be idempotent
- `graphk sync` must complete without errors after ingest
- Existing graph.yaml nodes must not be removed or corrupted

## Completion gate

```bash
graphk ingest ~/.claude-work/memory/
graphk sync
graphk query type feedback     # must return ≥3 results
graphk query type reference    # must return ≥3 results
python -m pytest tests/test_ingest.py   # all pass
```
