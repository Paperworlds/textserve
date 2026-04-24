# Changelog

## v0.1.13

- Fix slack MCP 401: add `Authorization: Bearer` header to server registration
- Accept comma-separated server names in `up`, `down`, `start`, `stop` (e.g. `textserve up sentry,grafana`)

## v0.1.12

- Add `textmap` server entry (stdio/process runtime)
- Archive `airbyte` and `graph` servers → `servers/_archived/`
- Suppress Docker container ID from terminal output on `start`
- Redirect native process stdout/stderr to `~/.cache/textserve/<name>.log`
- Fix stopped containers showing last-start uptime in `status` — now shows `-`

## v0.1.11

- Add `textserve up` — converge server to running + registered state (skips if already healthy)
- Add `textserve down` — stop and deregister one or more servers
- Both commands accept `--tag` and `--all` flags

## v0.1.10

- `textserve start` is now idempotent: skips servers already running and registered
- Add `--force` flag to `start` to override skip and restart unconditionally

## v0.1.9

- Health-gated registration: wait for server to pass health probe before registering with Claude
- Configurable via `health_wait` in `server.yaml` (default 15 s); soft warning on timeout

## v0.1.8

- Hash-based re-register skip: compute SHA-256 of `server.yaml` after each registration
- Subsequent `start` skips Claude re-registration when config is unchanged
- Stored in `~/.cache/textserve/<name>.reg.hash`

## v0.1.7

- Archive `datadog-security` server — unusable without mandatory auth that phones home to Datadog
- Add `archived.yaml` convention and `servers/_archived/` for dormant servers
- Add CI workflow (`go vet`, `go test` on every push)

## v0.1.6

- Open-source release under Elastic License 2.0

## v0.1.5

- `statusIcon` coverage, `StatusRunning` constant, `HOME` consistency fixes

## v0.1.4

- Rename `mcp-fleet` → `textserve`; fix leftover paths

## v0.1.3

- Add `textserve add` — scaffold new server entry from CLI

## v0.1.2

- Add `textserve doctor` — full diagnostic: registry, configs, deps, port conflicts
- Add `textserve preflight` — readiness check for tagged servers

## v0.1.1

- `textserve register` / `deregister` — manage Claude registration without stopping servers
- Per-server `server.yaml` overrides registry entry defaults

## v0.1.0

- Initial release: `start`, `stop`, `restart`, `logs`, `list`, `status`, `health`
- Docker and native (stdio/process) runtimes
- 1Password credential injection at start time
- Auto-registration with Claude Code (`claude.json`)
