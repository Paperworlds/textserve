---
id: "002"
title: "Gitignore personal files, untrack from git"
phase: "oss-prep"
repo: "mcp-fleet"
model: "sonnet"
depends_on: ["001"]
budget_usd: 1.50
---

# Gitignore + Untrack Personal Files

## Tasks

### 1. Update .gitignore

Add:
```
registry.yaml
servers/*/
!servers/example-*/
LEAD.md
REPORT.md
pipeline.yaml
prompts/
features/
.claude/
```

### 2. Untrack personal files

Run `git rm --cached` on:
- `registry.yaml`
- All `servers/*/` directories (except `servers/example-*`)
- `LEAD.md`, `REPORT.md`
- `pipeline.yaml`
- `prompts/` (entire directory)
- `features/` (entire directory)
- `.claude/` if tracked

Do NOT delete files from disk — only untrack them.

### 3. Move registry.yaml to user config location

The user's personal `registry.yaml` should live at `~/.local/mcpf/registry.yaml`
(alongside the existing `~/.local/mcpf/config.yaml`). The `config.yaml` already
has a `root:` field — update it to point at where `registry.yaml` now lives,
or update `findRepoRoot()` to also check `~/.local/mcpf/` as a default location.

Check how other paperworlds text projects (textworld, textprompts) handle
user-specific config paths for the right pattern.

### 4. Verify

```bash
git status          # personal files should be untracked, not deleted
mcpf status         # fleet should still work (files on disk)
ls servers/         # personal server dirs still exist on disk
```

## Constraints

- Files must stay on disk — the user's fleet must keep working
- `mcpf` must still find registry.yaml after the move
