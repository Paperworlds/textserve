# NEXT — things under consideration

Running inventory of ideas in flight for the textserve (mcp-fleet) project.
Entries link to prompt files when one exists; otherwise they're a short
description + rough shape.

## Active prompts (ready to run)

- **[005 — MCP context optimization](../.files/prompts/005-mcp-context-optimization.md)**
  Baseline + trim tool descriptions, disable rarely-used tools per server.
  Partly unblocked now that `disabled_tools:` ships in v0.1.7.
  Next step: run `/context`, record baseline, compare after re-registers.

- **[006 — Kubernetes/EKS MCP](../.files/prompts/006-kubernetes-mcp.md)**
  Add read-only EKS visibility without paying for Lens Pro.
  Recommended: `awslabs/eks-mcp-server` native (read-only by default).

## Backlog ideas (no prompt yet)

- **Dynamic MCP gateway via `notifications/tools/list_changed`**
  The MCP 2024-11-05 spec includes `notifications/tools/list_changed` — a server
  can signal mid-session that its tool list changed; the client re-fetches and
  future messages only carry the updated schema. If Claude Code handles this
  notification, textserve could become a single registered gateway that starts
  with 2 tools (`load_profile`, `list_profiles`), then expands on demand. Zero
  token cost for unloaded servers; no session restart required.

  **Unknowns requiring investigation:**
  1. Does Claude Code actually handle `notifications/tools/list_changed`? Test
     with a minimal Go MCP server that emits the notification after a tool call.
  2. Does the system prompt reflect the *current* tool list or the *initial* one?
     (i.e. does removing tools from the schema actually reduce tokens mid-session?)
  3. If Claude Code doesn't handle it natively — explore internal API. Look at
     `~/.claude/` internals, the MCP client source (Claude Code is Electron; devtools
     may expose the IPC/WebSocket the frontend uses to communicate with the MCP
     subprocess manager). A documented internal API path here could unlock dynamic
     loading even before Anthropic ships it officially.

  Investigation path: build a toy MCP server → test notification handling → if
  blocked, crack open Claude Code devtools and map the internal MCP session API.

- **CLAUDE_CONFIG_DIR awareness in mcpf**
  One-liner: `internal/claude/claude.go:configPath()` should read
  `$CLAUDE_CONFIG_DIR` before falling back to `~/.claude-work/.claude.json`.
  Unblocks textworkspace profile switching.

- **Tool consolidation / "Code Mode"** (from Cloudflare enterprise MCP post)
  For servers we control, collapse N per-resource tools into 1-2 dispatch
  tools. Highest-leverage context reduction technique. Datadog MCP is the
  obvious candidate if/when we fork it.

- **Dockerize EKS MCP**
  Follow-up to 006. Custom image based on awslabs + aws CLI + `~/.aws`
  mount. Only worth doing if we want process isolation.

- **AWS API MCP (broader)**
  `awslabs/aws-api-mcp-server` for non-EKS AWS ops. Scope read-only via
  IAM on a dedicated profile. Open question: tool count vs. utility.

## Recently shipped (for context)

- **v0.1.7** — `disabled_tools:` in server.yaml → `disabledTools` in
  claude.json. 23 tools disabled across datadog/jenkins/sentry.
- **v0.1.6** — sentry stdio registration fixes.
- **v0.1.5** — `mcpf register` command + local config for op secrets.
- **v0.1.4** — status table redesign.
