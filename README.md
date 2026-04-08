# mcp-fleet

mcp-fleet is a local fleet manager for MCP (Model Context Protocol) servers. It provides a single CLI (`mcpf`) to start, stop, and monitor a collection of Docker-based and native MCP servers, injecting credentials from 1Password at runtime and registering them with Claude Code automatically.

## Quick Start

```bash
just install        # build binary + install to ~/.local/bin/mcpf
mcpf status         # show all servers and their running state
mcpf start slack    # start a single server and register it with Claude
```

## CLI Reference

| Command | Description |
|---------|-------------|
| `mcpf start <name>` | Start a server and register it with Claude Code |
| `mcpf start --tag <tag>` | Start all servers with a given tag |
| `mcpf stop <name>` | Stop a server and deregister it |
| `mcpf restart <name>` | Stop then start a server |
| `mcpf logs <name> [-f]` | Show (or follow) container logs |
| `mcpf list [--tag <tag>]` | List all (or filtered) server names |
| `mcpf status` | Show all servers with running state and health |
| `mcpf health <name>` | Run a health probe for one server |
| `mcpf preflight --tags t1,t2 [--json]` | Check readiness of tagged servers |
| `mcpf add <name> --transport http --image img` | Scaffold a new server entry |
| `mcpf doctor` | Full diagnostic: registry, configs, deps, port conflicts |

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

Use `mcpf add` to scaffold the directory, server.yaml, hook.sh, and README.md:

```bash
mcpf add myserver --transport http --port 9899 --image my-image --tags ci,docker
```

Then:
1. Edit `servers/myserver/server.yaml` — fill in `env[]` for credentials, `deps[]` for preconditions.
2. Edit `servers/myserver/hook.sh` — add any side-effect setup (port-forwards, etc.).
3. Edit `servers/myserver/README.md` — document tools, auth, and usage.
4. Run `mcpf start myserver` to test.

## Credential Rotation

Credentials are fetched from 1Password at start time and cached in `~/.cache/mcp-<name>/`. To force a refresh:

```bash
rm -rf ~/.cache/mcp-<name>/
mcpf restart <name>
```

## Migration

The following files from the pre-fleet era are safe to remove after a 2-week shim period (approximately 2026-04-22):

- `~/projects/personal/skills/locals/mcp-hooks/*.sh` — all 12 per-server credential hooks
- `~/projects/personal/skills/locals/bin/mcp-<name>` — all 12 wrapper scripts
- `~/.config/mcp-servers.conf` — replaced by `registry.yaml`

Do **not** remove these until you have verified that `mcpf` handles all servers correctly in your workflow.
