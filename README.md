# textserve

textserve is a local fleet manager for MCP (Model Context Protocol) servers. It provides a single CLI (`textserve`) to start, stop, and monitor a collection of Docker-based and native MCP servers, injecting credentials from 1Password at runtime and registering them with Claude Code automatically.

## Quick Start

```bash
just install        # build binary + install to ~/.local/bin/textserve
textserve status         # show all servers and their running state
textserve start slack    # start a single server and register it with Claude
```

## CLI Reference

| Command | Description |
|---------|-------------|
| `textserve start <name>` | Start a server and register it with Claude Code |
| `textserve start --tag <tag>` | Start all servers with a given tag |
| `textserve stop <name>` | Stop a server and deregister it |
| `textserve restart <name>` | Stop then start a server |
| `textserve logs <name> [-f]` | Show (or follow) container logs |
| `textserve list [--tag <tag>]` | List all (or filtered) server names |
| `textserve status` | Show all servers with running state and health |
| `textserve health <name>` | Run a health probe for one server |
| `textserve preflight --tags t1,t2 [--json]` | Check readiness of tagged servers |
| `textserve add <name> --transport http --image img` | Scaffold a new server entry |
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

Credentials are fetched from 1Password at start time and cached in `~/.cache/mcp-<name>/`. To force a refresh:

```bash
rm -rf ~/.cache/mcp-<name>/
textserve restart <name>
```

## Migration

The following files from the pre-fleet era are safe to remove after a 2-week shim period (approximately 2026-04-22):

- `~/projects/personal/skills/locals/mcp-hooks/*.sh` — all 12 per-server credential hooks
- `~/projects/personal/skills/locals/bin/mcp-<name>` — all 12 wrapper scripts
- `~/.config/mcp-servers.conf` — replaced by `registry.yaml`

Do **not** remove these until you have verified that `textserve` handles all servers correctly in your workflow.
