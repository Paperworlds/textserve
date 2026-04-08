# mcp-fleet — Lead Prompt

Use this to start a new implementation chat. Paste the contents into a fresh session
pointed at `~/projects/personal/mcp-fleet/`.

---

We are building `mcp-fleet` — a standalone Go CLI (`mcpf`) for managing a fleet
of 12 MCP servers. The full spec is in `~/.local/projects/mcp-fleet.md`.

## What exists

- Repo at `~/projects/personal/mcp-fleet/` (private GitHub: pdonorio/mcp-fleet)
- Phase prompts in `prompts/` define the 5 implementation phases (Phase 1 DONE)
- `registry.yaml` — fleet registry with all 12 servers (complete)
- `servers/<name>/server.yaml` — per-server stubs (need env/volumes migration in Phase 2)
- `servers/<name>/hook.sh` — existing bash credential hooks (being superseded)
- Pipeline registered with `pp` — run `pp status` to see task states

## Architecture decisions (already made)

- **Language: Go** (module `github.com/pdonorio/mcp-fleet`, Go 1.22+)
- **No bash libs** — all logic in Go packages under `internal/`
- **Declarative credentials** — env vars declared in `server.yaml` with op:// URIs,
  not injected via hook.sh. See Phase 2 prompt for full schema.
- **Hook files** — kept only for pre-start side effects (kubectl port-forwards etc),
  not credential injection
- **Tests** — `go test ./...` not bats. Each phase has a completion gate.

## Source material to read before starting

1. `~/.local/projects/mcp-fleet.md` — full spec
2. `~/.config/mcp-servers.conf` — original flat registry (migrated to registry.yaml)
3. `~/projects/personal/skills/claude-code/memory/mcp-setup.md` — per-server auth details
4. `~/projects/personal/skills/locals/mcp-hooks/` — existing credential hooks to migrate
5. `prompts/000-phase1-scaffold.md` — what Phase 1 built

## Lead responsibilities

Your role is to **lead, not implement directly**. Specifically:

1. **Read source material** before each phase — understand existing code before delegating
2. **Review prompts** before running — catch spec gaps, wrong counts, missing constraints
3. **Patch prompts proactively** — fix issues in upcoming phases while current phase runs
4. **Run `pp run`** to execute phases via the pipeline, not implement manually
5. **Verify after each phase** — run `go test ./...`, `go vet ./...`, `just build`,
   smoke-test the binary. If tests fail, fix before moving on.
6. **Report issues** — surface spec ambiguities, architectural tradeoffs, missing edge cases
7. **Do not implement** unless fixing a test gate failure or a prompt patch that needs
   a code correction

## Constraints

- Go only — no Python, no bash libs (pre-start hooks are the only remaining shell)
- `go vet ./...` must pass after every phase
- `go test ./...` must pass after every phase — fix failures before `pp run` next phase
- `mcpf` must work as a single entrypoint — no per-server scripts on PATH after migration
- 12 servers: jenkins, snowflake, grafana, grafana-pdx, notion, airbyte, slack,
  slack-search, datadog, paradex-db, airflow, sentry

## Current state

Run `pp status` to see which phase is next.
Run `pp run` to execute the next pending phase.
Run `go test ./...` + `just build` to verify current state.

## Known issues to watch for in remaining phases

- Phase 3: `docker.ResolveEnv` must process env entries in order (value_template
  references earlier vars). Order matters — enforce it.
- Phase 3: `claude.Register` URL must include `endpoint_path` from server.yaml
  (varies per server: snowflake=/snowflake-mcp, paradex-db=/sse, most others=/mcp)
- Phase 3: airflow is native (not docker) — needs separate `internal/native` package
- Phase 4: `mcpf status` must write `~/.files/states/mcp-fleet.json` on every run
  (statusline integration)
- Phase 4: health probes run concurrently with per-server timeouts
- Phase 5: preflight JSON schema is a hard contract — knowledge-harvest depends on it
