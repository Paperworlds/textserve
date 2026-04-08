---
id: "002"
title: "Phase 3 — docker + claude packages + CLI core"
phase: "phase-3"
repo: "mcp-fleet"
model: "sonnet"
depends_on: ["001"]
budget_usd: 3.00
---

# Phase 3 — CLI Core

Implement the `bin/mcpf` entrypoint and the docker/claude lifecycle packages.
The binary must handle all three transport types: docker (most servers), native
(airflow), and stdio (sentry).

## Context

`server.yaml` files now contain declarative env/volumes config (from Phase 2).
The Go binary owns credential resolution (1Password via `internal/op`) and
assembles docker run flags — no shell hooks for credentials.

Read `internal/registry` and `internal/op` packages from Phase 2 before implementing.

## Tasks

### 1. internal/docker package

`internal/docker/docker.go`:

**`ResolveEnv(cfg *registry.ServerConfig) ([]string, error)`**
- Processes `cfg.Env` entries in order, building a `map[string]string`
- Source dispatch:
  - `value`: use as-is
  - `value_template`: expand `${VAR}` references using already-resolved vars
  - `op` + `cache`: call `op.Cached(service, field, uri)` — parse service/field from cache key
  - `cache_file`: call `op.CacheFileRead(path)`
- Returns slice of `"NAME=VALUE"` strings for docker `-e` flags

**`ResolveVolumes(cfg *registry.ServerConfig) ([]string, error)`**
- For each volume entry, expand `${HOME}` in host path
- If `resolve_symlinks: true`, call `filepath.EvalSymlinks` on host path
- Returns slice of `"host:container[:ro]"` strings for docker `-v` flags

**`Run(name string, cfg *registry.ServerConfig) error`**
- If `pre_start` is set, exec that script first and wait for it to complete
- Calls `docker rm -f mcp-<name>` (ignore error if not running)
- Assembles docker run command:
  ```
  docker run -d
    -p <port>:<container_port>
    --name mcp-<name>
    [--network <network>]
    [-e NAME=VALUE ...]
    [-v host:container ...]
    <image>
    [extra_args...]
  ```
- Env vars and volumes from ResolveEnv/ResolveVolumes
- Streams docker output to stderr for visibility

**`Stop(name string) error`**
- `docker rm -f mcp-<name>`

**`Status(name string) (string, error)`**
- `docker inspect --format {{.State.Status}} mcp-<name>`
- Returns "running", "stopped", "unknown"

**`Logs(name string, follow bool) error`**
- `docker logs [--follow] mcp-<name>` — streams to stdout

### 2. internal/native package

`internal/native/native.go` — handles `transport: native` servers (airflow):

**`Start(name string, cfg *registry.ServerConfig) error`**
- Resolves env via `docker.ResolveEnv` (same logic, reused)
- Builds PATH: prepend `<native_venv>/bin` if `native_venv` set
- Execs `native_cmd native_args...` in background with env
- Writes PID to `pid_file`

**`Stop(name string, cfg *registry.ServerConfig) error`**
- Read PID from `pid_file`, send SIGTERM, remove pid file

**`Status(name string, cfg *registry.ServerConfig) (string, error)`**
- Read PID from `pid_file`, check process is alive → "running" or "stopped"

### 3. internal/claude package

`internal/claude/claude.go`:

**`Register(name string, cfg *registry.ServerConfig) error`**
- Builds URL: `http://localhost:<port><endpoint_path>`
- Execs: `claude mcp add --transport http <name> <url>`

**`Deregister(name string) error`**
- Execs: `claude mcp remove <name>`

**Stdio servers** (`managed_by: "claude"`): Register/Deregister are no-ops, print
info message and return nil.

### 4. cmd/mcpf/main.go — cobra CLI

Subcommands:

**`start <name|--tag t>`**
- Resolve names: single name, or all names matching tag via registry.FilterByTag
- For each name in order:
  - If `transport: stdio` and `managed_by: claude`: print "sentry is managed by Claude — no action needed" and skip
  - If `transport: native`: native.Start + claude.Register
  - If `transport: http`: docker.Run + claude.Register
- Check deps first (run each `deps[].cmd`; on failure print hint and abort)

**`stop <name|--tag t>`**
- Reverse of start: claude.Deregister + docker.Stop (or native.Stop)

**`restart <name|--tag t>`**
- stop then start

**`logs <name> [-f]`**
- docker.Logs(name, follow)

**`list [--tag t]`**
- Print server names, one per line (filtered by tag if given)

### 5. Dependency checking

`internal/deps/deps.go`:

**`Check(deps []registry.Dep) error`**
- For each dep: run `bash -c "<cmd>"`, check exit code
- On failure: print the `hint` and return error naming which dep failed

Called by `start` before any container ops.

### 6. Tests

`internal/docker/docker_test.go`:
- Test ResolveEnv with static values and value_template expansion
- Test ResolveVolumes with ${HOME} expansion

`internal/deps/deps_test.go`:
- Test passing dep (cmd: "true")
- Test failing dep (cmd: "false") returns error with hint

`cmd/mcpf/main_test.go`:
- Test `list` command outputs expected server names
- Test `list --tag docker` excludes airflow and sentry

For docker.Run and native.Start: do not test docker/process execution directly —
these are integration paths. Test the flag assembly (ResolveEnv, ResolveVolumes)
and dispatch logic only.

## Constraints

- All scripts and the binary must handle missing `servers/<name>/server.yaml`
  gracefully (fall back to registry.yaml fields only)
- `go vet ./...` must pass
- Do not shell out to bash for anything except `deps[].cmd` checks and `pre_start`

## Completion gate

Run `go test ./...` — all tests must pass.
Run `go vet ./...` — no errors.
Run `just build` — binary compiles.
Smoke test: `./bin/mcpf list` prints all 12 servers.
Smoke test: `./bin/mcpf list --tag docker` prints 10 servers.
