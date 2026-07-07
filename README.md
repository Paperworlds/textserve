# textserve

textserve is a local fleet manager for MCP (Model Context Protocol) servers, plus a single MCP entry-point (`textserve-mcp`) that multiplexes the whole fleet behind one stdio connection. It runs Docker-based and native MCP servers, injects 1Password credentials at runtime, and exposes a curated meta-tool surface to Claude with the rest of the fleet dark until a human approves a bundle.

Two binaries ship from this module:

- `textserve` — orchestrator CLI (start/stop/status/bundle/toggle-inbox/doctor)
- `textserve-mcp` — single MCP server registered in Claude; speaks stdio, dispatches to fleet backends

## Quick Start

```bash
just install                 # build both binaries → ~/.local/bin/{textserve,textserve-mcp}
textserve status             # show all servers and their running state
textserve start slack        # start a single server and register it with Claude
textserve bundle list        # show togglable bundles and their enabled state
```

## textserve-mcp — single MCP surface

`textserve-mcp` is the only MCP entry registered in `~/.claude.json`. Claude sees seven always-on meta-tools; every other tool is dark until a human approves a bundle.

| Meta-tool | Role |
|---|---|
| `bundles.list` | List bundles with their effective enabled state (overlay > registry default). |
| `bundles.toggle` | Request enable/disable of a bundle. Writes to `~/.local/textserve/toggle-inbox/` and returns a `request_id` + `inbox_path`. Human approves out-of-band. |
| `bundles.example` | Get example usage for a tool from textforums threads (`$TEXTFORUMS_ROOT`, default `~/.local/paperworlds/textforums`). |
| `textmap.search` | Keyword search across the textmap knowledge graph. |
| `textmap.propose` | Propose a new node — writes to the textmap inbox for human review. |
| `textmap.detail` | Multiplexed read: `expand`, `list_labels`, `query_node`, `query_relation`, `query_type`, `query_why`. |
| `textmap.init_session` | Bootstrap context: workspaces, active intents, open problems, active goals, recent decisions. Optional `query` adds search hits. |

### Toggle flow

```bash
# model calls bundles.toggle({bundle: "data", enable: true})  →  request queued
textserve toggle-inbox list             # see what's pending
textserve toggle-inbox approve <id>     # apply the overlay
```

The running `textserve-mcp` watches `~/.local/textserve/bundles.yaml` (2s mtime poll) and reconciles on change — newly-enabled servers' tools appear in Claude via `notifications/tools/list_changed` without reconnecting. Disabled bundles' tools are removed the same way.

Bundle-owned tools are prefixed: `<server>.<upstream>` (e.g. `jenkins.list_builds`). Upstream input schemas pass through verbatim.

## Bundles

A bundle is a named set of MCP servers that toggle as a unit. Defined in `registry.yaml`:

```yaml
bundles:
  coding:
    description: "Code review and CI"
    servers: [jenkins, sentry]

  data:
    description: "Data analysis and observability"
    tags: [data, monitoring]
```

```bash
textserve bundle list
textserve bundle show coding
textserve bundle use coding   # converge the docker/process fleet to this bundle
```

`bundle use` is orchestrator-side (start/stop containers). The MCP-tool gating is independent and lives in the toggle inbox + overlay. The `profile` subcommand alias still works for muscle memory.

## CLI Reference

| Command | Description |
|---------|-------------|
| `textserve start <name>` | Start a server and register it with Claude Code |
| `textserve start --tag <tag>` | Start all servers with a given tag |
| `textserve stop <name>` | Stop a server and deregister it |
| `textserve restart <name>` | Stop then start a server |
| `textserve up [name\|--tag\|--all]` | Converge to running+registered (skip if already up) |
| `textserve down [name\|--tag\|--all]` | Stop and deregister |
| `textserve bundle list` | List defined bundles with ENABLED column |
| `textserve bundle show <name>` | Show resolved server list for a bundle |
| `textserve bundle use <name>` | Converge fleet to a named bundle |
| `textserve toggle-inbox list [--all]` | List pending (or all) bundle toggle requests |
| `textserve toggle-inbox approve <id>` | Apply a toggle to the overlay |
| `textserve toggle-inbox deny <id> [reason]` | Reject a pending toggle |
| `textserve logs <name> [-f]` | Show (or follow) container logs |
| `textserve list [--tag <tag>]` | List all (or filtered) server names |
| `textserve status` | Show all servers with running state and health |
| `textserve health <name>` | Run a health probe for one server |
| `textserve preflight --tags t1,t2 [--json]` | Check readiness of tagged servers |
| `textserve add <name> --transport http --image img` | Scaffold a new server entry |
| `textserve mode <server> <tool> readonly\|readwrite` | Flip a live adapter's write access (resets on restart) |
| `textserve doctor` | Full diagnostic: registry, configs, deps, port conflicts |

## Registry Schema

`registry.yaml` is the source of truth. Each server entry:

```yaml
servers:
  myserver:
    image: "my-docker-image"       # Docker image (omit for native/stdio)
    protocol: http                 # http | stdio
    runtime: docker                # docker | process | claude
    port: 9887                     # host port
    container_port: 9887           # port inside container
    endpoint_path: /mcp            # HTTP MCP endpoint path
    tags: [ci, docker]             # arbitrary tags for filtering
    deps: []                       # prerequisite checks (cmd + hint)
    health:
      endpoint: /health            # HTTP health endpoint
      timeout: 5                   # probe timeout (seconds)
```

`runtime` controls how textserve-mcp reaches the upstream:
- `docker` → speaks MCP streamable-HTTP at `http://localhost:<port><endpoint_path>`
- `process` → speaks the tools-api gateway's `/describe` + `/invoke` REST
- `claude` → spawns a stdio child via `native_cmd` from the server's `server.yaml`

Full per-server configuration lives in `servers/<name>/server.yaml` and supports `env`, `volumes`, `extra_args`, `pre_start`, `native_cmd`, and more. See an existing server for examples.

## Adding a New Server

```bash
textserve add myserver --transport http --port 9899 --image my-image --tags ci,docker
```

Then:
1. Edit `servers/myserver/server.yaml` — fill in `env[]` for credentials, `deps[]` for preconditions.
2. Edit `servers/myserver/hook.sh` — add any side-effect setup (port-forwards, etc.).
3. Edit `servers/myserver/README.md` — document tools, auth, and usage.
4. Run `textserve start myserver` to test.

## Credential Rotation

Credentials are fetched from 1Password at start time. Process server logs live at `~/.cache/textserve/<name>.log`. To force a credential refresh on HTTP API servers (e.g. `tools-api`):

```bash
curl -X POST http://localhost:<port>/auth/<tool>/refresh
```

For MCP servers managed by Docker, restart picks up fresh credentials:

```bash
textserve restart <name>
```

## State Files

| Path | Owner | Purpose |
|---|---|---|
| `~/.local/textserve/bundles.yaml` | `toggle-inbox approve` | Overlay state — which bundles are effectively enabled |
| `~/.local/textserve/toggle-inbox/<id>.yaml` | `bundles.toggle` meta-tool | Pending requests awaiting human approval |
| `~/.local/log/textserve-toggle.log` | both | Append-only JSON-line audit (override via `$TEXTSERVE_LOG_PATH`) |
| `~/.cache/textserve/<name>.log` | server start | Per-server runtime logs |

Override state location with `$TEXTSERVE_STATE_DIR`; the textforums root for `bundles.example` with `$TEXTFORUMS_ROOT`.

## Part of Paperworlds

textserve is part of [Paperworlds](https://github.com/paperworlds) — an open org building tools and games around AI agents and text interfaces.

## License

Elastic License 2.0 — see [LICENSE](LICENSE).
