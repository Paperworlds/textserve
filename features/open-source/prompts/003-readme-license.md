---
id: "003"
title: "Rewrite README for OSS, add LICENSE"
phase: "oss-prep"
repo: "mcp-fleet"
model: "sonnet"
depends_on: ["002"]
budget_usd: 2.00
---

# README Rewrite + LICENSE

## Tasks

### 1. Rewrite README.md

Target audience: someone who wants to manage their MCP servers with a single CLI.
No references to personal paths, company names, or specific 1Password items.

Structure:
- **What is mcpf** — one paragraph
- **Quick start** — clone, `cp registry.yaml.example registry.yaml`, edit, `mcpf start`
- **Installation** — `go install github.com/paperworlds/mcp-fleet/cmd/mcpf@latest` or `just install`
- **CLI reference** — table of all commands with one-line descriptions
- **Registry schema** — document `registry.yaml` fields with types and descriptions
- **Server config** — document `servers/<name>/server.yaml` fields
- **Configuration patterns** — three ways to set up config:
  1. In-repo: clone repo, put registry.yaml in root, mcpf finds it via CWD walk-up
  2. External dir: put config anywhere, set `root:` in `~/.local/mcpf/config.yaml`
  3. Private repo: `go install` mcpf, keep config in a separate private repo
- **Adding a server** — `mcpf add <name> --transport http --port 9899`
- **License** — MIT

### 2. Add LICENSE

Create `LICENSE` at repo root — MIT license, copyright paperworlds.

### 3. Verify

```bash
# README renders correctly
cat README.md | head -50

# No personal references leaked
grep -i "pdonorio\|paradigm\|paradex\|paulie\|Private/" README.md  # must return empty
```

## Constraints

- Zero personal references in README
- All CLI commands documented
- Registry schema fully documented with field types
