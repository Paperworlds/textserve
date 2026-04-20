# Changelog

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
