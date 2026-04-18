---
id: "004"
title: "Phase 5 — docs, server READMEs, skills migration"
phase: "phase-5"
repo: "mcp-fleet"
model: "sonnet"
depends_on: ["003"]
budget_usd: 2.00
---

# Phase 5 — Docs & Migration

Write per-server READMEs, update the skills repo memory files to point at
mcp-fleet as the source of truth, and clean up deprecated skills files.

## Tasks

1. Write `servers/<name>/README.md` for each of the 12 servers:
   - Tools exposed (list them)
   - Auth setup (env vars, 1Password paths if applicable)
   - Known issues / preconditions
   - Example usage

2. Write top-level `README.md`:
   - What mcp-fleet is
   - Quick start (install.sh, mcpf status, mcpf start <name>)
   - CLI reference table
   - Registry schema reference
   - Adding a new server (mcpf add)

3. Update skills repo memory files:
   - `~/projects/personal/skills/claude-code/memory/mcp-setup.md` —
     add note that mcp-fleet is now the source of truth; point to registry.yaml
   - `~/projects/personal/skills/claude-code/memory/skills_setup.md` —
     update MCP tooling section to reference mcpf instead of mcp-manage

4. Remove deprecated files from skills repo (after 2-week shim period):
   - List files to remove but DO NOT delete them — leave a checklist for the user.

## Constraints

- Do not delete any skills repo files — only update pointers and add deprecation notes.
