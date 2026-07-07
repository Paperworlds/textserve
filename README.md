# textserve

textserve is a local fleet manager for MCP (Model Context Protocol) servers. It provides a single CLI (`textserve`) to start, stop, and monitor a collection of Docker-based and native MCP servers, injecting credentials from 1Password at runtime and registering them with Claude Code automatically.

## Quick Start

```bash
just install        # build binary + install to ~/.local/bin/textserve
textserve status         # show all servers and their running state
textserve start slack    # start a single server and register it with Claude
```

## Session Profiles

Every registered MCP server injects its tool descriptions into Claude's system prompt on every message — whether or not those tools are relevant to the session. Profiles let you declare which servers are active for a given context and converge the fleet in one command.

```bash
textserve profile list
# PROFILE           SERVERS  DESCRIPTION
# -------           -------  -----------
# coding                  3  Code review and CI
# data                    5  Data analysis and observability
# comms                   2  Slack and notifications
# minimal                 1  Knowledge graph only

textserve profile use coding
# jenkins   → already running
# sentry    → started
# textmap   → already running
# grafana   → stopped
# snowflake → stopped
# slack     → stopped
```

Profiles are defined in `registry.yaml` alongside your servers. Each profile lists servers explicitly, by tag, or both:

```yaml
profiles:
  coding:
    description: "Code review and CI"
    servers: [jenkins, sentry]   # explicit names
    tags: [knowledge]            # expanded by tag

  data:
    description: "Data analysis and observability"
    tags: [data, monitoring]     # all servers with these tags
```

`profile use` brings up servers in the profile that aren't running, and brings down servers that aren't in the profile. Already-running servers in the profile are skipped. The `--force` flag on individual `start` calls still works if you need to restart a specific server.

If you use [textaccounts](https://github.com/paperworlds) to switch Claude Code profiles, textserve respects `$CLAUDE_CONFIG_DIR` automatically — so switching accounts and switching your MCP surface are coordinated:

```bash
textaccounts use work    # sets CLAUDE_CONFIG_DIR → ~/.claude-work
textserve profile use coding   # registers to ~/.claude-work/.claude.json
```

## CLI Reference

| Command | Description |
|---------|-------------|
| `textserve start <name>` | Start a server and register it with Claude Code |
| `textserve start --tag <tag>` | Start all servers with a given tag |
| `textserve stop <name>` | Stop a server and deregister it |
| `textserve restart <name>` | Stop then start a server |
| `textserve up [name\|--tag\|--all]` | Converge to running+registered (skip if already up) |
| `textserve down [name\|--tag\|--all]` | Stop and deregister |
| `textserve profile list` | List defined profiles |
| `textserve profile show <name>` | Show resolved server list for a profile |
| `textserve profile use <name>` | Converge fleet to a named profile |
| `textserve logs <name> [-f]` | Show (or follow) container logs |
| `textserve list [--tag <tag>]` | List all (or filtered) server names |
| `textserve status` | Show all servers with running state and health |
| `textserve health <name>` | Run a health probe for one server |
| `textserve preflight --tags t1,t2 [--json]` | Check readiness of tagged servers |
| `textserve add <name> --transport http --image img` | Scaffold a new server entry |
| `textserve mode <server> <tool> readonly\|readwrite` | Flip a live adapter's write access (resets on restart) |
| `textserve mode <server> --all readonly\|readwrite` | Flip all adapters in a server at once |
| `textserve doctor` | Full diagnostic: registry, configs, deps, port conflicts |

## Registry Schema

`registry.yaml` is the source of truth. Each server entry:

```yaml
servers:
  myserver:
    image: "my-docker-image"       # Docker image (omit for native/stdio)
    transport: http                 # http | native | stdio
    port: 9887                      # host port
    container_port: 9887            # port inside container
    endpoint_path: /mcp             # Claude registration URL path
    tags: [ci, docker]              # arbitrary tags for filtering
    deps: []                        # prerequisite checks (cmd + hint)
    health:
      endpoint: /health             # HTTP health endpoint
      timeout: 5                    # probe timeout (seconds)
```

Full per-server configuration lives in `servers/<name>/server.yaml` and supports `env`, `volumes`, `extra_args`, `pre_start`, and more. See an existing server for examples.

## Adding a New Server

Use `textserve add` to scaffold the directory, server.yaml, hook.sh, and README.md:

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

## Migration

The following files from the pre-fleet era are safe to remove after a 2-week shim period (approximately 2026-04-22):

- `~/projects/personal/skills/locals/mcp-hooks/*.sh` — all 12 per-server credential hooks
- `~/projects/personal/skills/locals/bin/mcp-<name>` — all 12 wrapper scripts
- `~/.config/mcp-servers.conf` — replaced by `registry.yaml`

Do **not** remove these until you have verified that `textserve` handles all servers correctly in your workflow.

## Part of Paperworlds

textserve is part of [Paperworlds](https://github.com/paperworlds) — an open org building tools and games around AI agents and text interfaces.

## License

Elastic License 2.0 — see [LICENSE](LICENSE).
