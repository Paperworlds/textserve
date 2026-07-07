# open-source — Feature Lead

Feature of: mcp-fleet
Target: `github.com/paperworlds/mcp-fleet`

## What this feature does

Prepare mcp-fleet for open-sourcing. Separate the generic Go CLI engine from
personal fleet configuration (1Password items, company URLs, AWS profiles).

The Go code is already generic — no personal data in `.go` files. All personal
data lives in `registry.yaml`, `servers/*/server.yaml`, and docs/prompts.

## Strategy

1. Change module path to `github.com/paperworlds/mcp-fleet`
2. Create example configs showing the schema (registry.yaml.example, servers/example-*/)
3. Gitignore personal files (registry.yaml, servers/*, LEAD.md, prompts/, features/, etc.)
4. Untrack them with `git rm --cached` — files stay on disk, fleet keeps working
5. Rewrite README for OSS audience
6. Add MIT LICENSE
7. Squash to clean orphan branch (no secrets in history)

## Constraints

- Personal files must stay on disk after untracking — the user's fleet must keep working
- `go vet ./...` and `go test ./...` must pass after module rename
- Example configs must parse correctly with `mcpf status`

## Running pp in the background

Always use the Bash tool's `run_in_background: true` parameter when launching
`pp run <id>` — never use shell `&`.
