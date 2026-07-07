# Changelog

## v0.10.1

- **Refactor: split `cmd/textserve-mcp/main.go` (485 lines) into per-surface files.** Pure intra-package code movement, no behavior change:
  - `main.go` keeps wiring: package doc, vars/consts, `main()`, `registerMetaTools()`, `loadRegistry()`, `findRepoRoot()`
  - `textmap.go` — `ProxyResult`/`proxyResult()` and the three textmap proxies (`search`, `propose`, `detail`) with their input types
  - `bundles_meta.go` — `bundles.list` and `bundles.toggle` handlers with their types
- Documented load-bearing complexity that a prior refactor pass had flagged for removal but which is actually needed: `findRepoRoot`'s three-path resolution (tests / dev / prod-symlink) and `proxyResult`'s dual purpose (client outputSchema workaround + genuinely-untyped proxied output)
- Verified: `go build`, 84 tests green, installed binary lists 7 meta-tools

## v0.10.0

- **Live bundle reconciliation.** Approving a `bundles.toggle` request via `textserve toggle-inbox approve` no longer requires a session reconnect — the running `textserve-mcp` now watches the overlay file (`~/.local/textserve/bundles.yaml`) and reconciles on change
- New `bundleSupervisor` (`cmd/textserve-mcp/bundles.go`) owns the dynamic registration: tracks bundle-owned tool names per server, calls `Server.AddTool` for newly-enabled servers and `Server.RemoveTools` for newly-disabled ones. Both paths auto-emit `notifications/tools/list_changed` (go-sdk handles the wire notification)
- Background watcher: 2s mtime poll on `togglestate.OverlayPath()`. Cheap (one `stat` per tick), survives the overlay file not existing
- Meta-tool collisions still blocked; bundle ownership is keyed by server so re-enabling a server cleanly re-registers its tools without duplicating

## v0.9.2

- `bundles.toggle` now does a read-back: after `Request()` writes the inbox file, the handler calls `Get(id)` and verifies bundle/action/status match what was written. Guards against silent loss on flaky filesystems and against a racing process clearing the inbox between write and response
- Response gains an `inbox_path` field — the absolute path of the queued YAML — so callers can locate the file without guessing
- New exported `toggleinbox.EntryPath(id)` (thin wrapper over the existing unexported helper)

## v0.9.1

- `textmap.init_session` is now real (was a stub). Aggregates five `query_type` calls against textmap-mcp — workspaces (all), active intents, open problems, active goals, recent decisions — and returns each as a section with `count` (total) and `nodes` (capped at 20). Optional `query` field runs a parallel `search` and includes its raw hits as `search_hits`
- Per-section failures land in a `warnings` array rather than failing the whole call; the bootstrap stays useful even if one query type breaks
- Implementation in `cmd/textserve-mcp/init_session.go`. `extractNodes` handles both structured-content (map) and text (string) payloads to absorb future textmap-mcp output shape changes
- Spike 5 of the textserve-mcp arc — closes the original 5-spike plan.

## v0.9.0

- `bundles.example` is now real (was a stub). Reads thread YAMLs under `$TEXTFORUMS_ROOT` (default `~/.local/paperworlds/textforums`), filters by `meta.tags` matching the requested tool name, returns up to 5 threads with slug/title/status/tags/entry_count and a 240-char preview of the first entry
- Convention: tag threads with the exact tool name (e.g. `textmap.search`). Falls back to the server-name prefix (e.g. `textmap`) when no exact-match thread exists
- Implementation in `cmd/textserve-mcp/examples.go`; no new dependencies (already-vendored `gopkg.in/yaml.v3`)
- Smoke-tested: `bundles.example {"tool":"textmap"}` returns 3 real open threads from this user's textforums
- Spike 4 of the textserve-mcp arc.

## v0.8.3

- New `internal/backends/toolsapi.go` — `ToolsAPIBackend` wraps the FastAPI gateway at `:10893`. Not MCP: speaks the gateway's custom REST shape (`GET /describe`, `POST /invoke {tool, action, params}`). Each `(adapter, action)` pair from `/describe` is exposed as a separate MCP tool named `<adapter>.<action>`; the bundle layer prefixes with the registry server name, yielding e.g. `tools-api.snowflake.query`
- `bundles.go` dispatch on `runtime=process` wires the new backend (was: stderr notice deferring to 3c-iv). With the gateway running locally, `/describe` returns 6 adapters (pagerduty, sentry, jenkins, athena, datadog, snowflake)
- Param schemas are translated from the gateway's `{type, required, example}` form into JSON Schema `{type, properties, required, examples}`
- Sub-spike 3c-iv. The full 3c arc is now complete: stdio / docker-HTTP-MCP / tools-api gateway all route through one MCP entry.

## v0.8.2

- **Bundle gating** (`cmd/textserve-mcp/bundles.go`): walk enabled bundles at startup, resolve their servers, build a backend per `runtime`, list its tools, register each as `<server>.<upstream>` via low-level `Server.AddTool` (upstream input schema passes through verbatim). 5s timeout on `ListTools` so a hung child doesn't block startup. Collisions with meta-tools and dark bundles' tools are both omitted from `tools/list`
- **Docker backend** (`internal/backends/docker.go`): MCP streamable-HTTP client against `http://localhost:<port><endpoint_path>` from `server.yaml`. Lazy-connect, `DisableStandaloneSSE: true`. Containers assumed already running; failed connects → stderr warning, no crash
- `runtime=process` still emits a stderr notice deferring to 3c-iv
- `textmap` excluded by name (covered by curated meta-tools)
- Sub-spikes 3c-ii + 3c-iii.

## v0.8.0

- New `internal/backends/` package — `Backend` interface (`Name`, `ListTools`, `CallTool`, `Close`) over upstream MCP-compatible servers
- `StdioBackend` factored out of the old `textmapclient`; builds from a parsed `server.yaml`'s `native_cmd`/`native_args`, lazy-connects on first use
- `cmd/textserve-mcp` switched from `textmapclient.Client` to `*backends.StdioBackend` for the textmap proxies; observable behaviour unchanged
- Removed `internal/textmapclient/` (folded into backends)
- Sub-spike 3c-i. Bundle gating (3c-ii), docker backend (3c-iii), and tools-api backend (3c-iv) follow.

## v0.7.4

- New `textmap.detail` meta-tool — multiplexed proxy that dispatches on an `action` field to one of six textmap read tools: `expand`, `list_labels`, `query_node`, `query_relation`, `query_type`, `query_why`. Compresses the textmap-mcp read surface from six tools to one, with per-action argument validation.
- This brings `textserve-mcp` to feature parity with the standalone `textmap` MCP registration. Next step: deprecate the direct `textmap` MCP entry and route all callers through textserve-mcp.

## v0.7.3

- Fix `tools/list` schema rejection in Claude Code's MCP client. `textmap.search` and `textmap.propose` returned `Result any`, which the go-sdk emitted as an empty outputSchema; the validator flagged it as "Invalid input". Replaced with `result_json` (string) — proxied tool output is now JSON-encoded text. Callers (Claude) parse the string.

## v0.7.2

- `Justfile` builds and installs `bin/textserve-mcp` alongside `textserve`; symlinked to `~/.local/bin/textserve-mcp` on `just install`
- Registered `textserve-mcp` in user-scope MCP config alongside existing `textmap` server (Plan B: coexistence). The 6 meta-tools surface in live Claude sessions without removing prior registrations.
- 3c (tools-api HTTP backend + bundle gating) deferred — the registry's `coding` bundle and tools-api adapters overlap; revisit once we decide whether tools-api adapters move into the bundle surface.

## v0.7.1

- New `internal/textmapclient/` — singleton MCP client wrapping a lazily-spawned `uv run python -m textmap_mcp` child process. Command line read from `servers/textmap/server.yaml` (single source of truth shared with the orchestrator).
- `textmap.search` and `textmap.propose` are now real proxies (no longer stubs). `textmap.init_session` remains a stub (spike 5).
- `textmap.propose` exposes typed input matching textmap-mcp's `propose_node`: type, id, description, body, labels, workspace, connections.
- Verified end-to-end: `tools/call textmap.search {"intent":"textserve bundles"}` through the daemon returns real graph nodes from the live textmap-mcp child.
- Sub-spike 3b of the textserve-mcp arc; 3c wires the tools-api HTTP backend + bundle gating.

## v0.7.0

- New binary `cmd/textserve-mcp/` — the single MCP surface anticipated by the bundles arc; speaks MCP over stdio via `github.com/modelcontextprotocol/go-sdk` v1.6.1
- Six always-on meta-tools registered: `bundles.list`, `bundles.toggle`, `bundles.example`, `textmap.search`, `textmap.propose`, `textmap.init_session`
- `bundles.list` returns full bundle state (name, description, resolved servers, effective enabled, source overlay|registry)
- `bundles.toggle` writes a fire-and-forget request into `toggleinbox` with `requester:"mcp"`; rejects unknown bundles to keep the inbox clean
- `bundles.example`, `textmap.*` registered as stubs returning `{status:"not_implemented"}` — real proxies land in sub-spikes 3b, 4, 5
- `findRepoRoot` for the daemon — `$TEXTSERVE_ROOT` first, then walk up from the binary, then the orchestrator's config file fallback (no cwd dependency since Claude may launch from anywhere)
- 4 new unit tests on the meta-tools; full suite stays green
- Sub-spike 3a of the textserve-mcp arc; 3b wires the textmap stdio backend

## v0.6.0

- New `togglestate` package — per-host overlay at `~/.local/textserve/bundles.yaml` records explicit enable/disable decisions on top of `registry.yaml` defaults (`$TEXTSERVE_STATE_DIR` overrides the location)
- New `toggleinbox` package — file-based queue at `$TEXTSERVE_STATE_DIR/toggle-inbox/<id>.yaml` for bundle-toggle requests; append-only JSON-line audit at `~/.local/log/textserve-toggle.log` (override via `$TEXTSERVE_LOG_PATH`)
- CLI: `textserve toggle-inbox {list,approve,deny,request}` — humans approve/deny pending bundle toggles; `approve` mutates the overlay state, `deny` does not
- `textserve bundle list` shows effective enabled state (overlay wins over registry); `yes*` / `no*` marker when the overlay overrides the registry default
- `textserve bundle show <name>` reports `[enabled|disabled via overlay|registry]`
- Fire-and-forget toggle semantics: `Request()` returns immediately with a pending entry; a human (or future MCP meta-tool) resolves it asynchronously
- Spike #2 of the textserve-bundles arc (`textserve/docs/PLANS.yaml`)

## v0.5.0

- Rename `profiles:` → `bundles:` in `registry.yaml`; rename `ProfileEntry` → `BundleEntry`, `FleetRegistry.Profiles` → `Bundles`, `ResolveProfile` → `ResolveBundle` (`internal/registry/registry.go`)
- New `enabled: bool` axis on each bundle (default `false`; `minimal` bundle ships `enabled: true`); foundation for the upcoming `bundles.toggle` MCP meta-tool
- CLI: `textserve bundle {list,show,use}`; `textserve profile` kept as a hidden alias for one release; `list` output gains an `ENABLED` column
- Backwards-compat shim: registry.yaml with a legacy `profiles:` key still loads, prints a stderr deprecation warning on load
- Spec: `docs/textserve-bundles.yaml` (monorepo root) + `textserve/docs/PLANS.yaml` capture the full multi-spike arc

## v0.4.0

- BREAKING: storage root moves from `~/.config/paperworlds/textserve/` to `~/.local/paperworlds/textserve/`. `configFilePath()` (cmd/textserve/main.go) and `localconfig.Path()` (internal/localconfig) both follow. README snippets for `tools-api`, `snowflake`, `_archived/datadog-api` updated.
- Single-user move — no migration command. `config.yaml` + `local.yaml` moved by hand from the prior root.

## v0.3.6

- Add `textserve permissions <server>` — show live readonly/readwrite mode and available actions for each adapter in a running server (GETs `/permissions` endpoint)

## v0.3.5

- Add `pagerduty` adapter to `tools-api` — query PagerDuty incidents, alerts, and services via `POST /invoke {tool: "pagerduty", action: ...}`
- Actions: `list_incidents` (status/urgency/service/time filters), `get_incident`, `list_alerts` (per incident), `list_services`
- Auth: Token auth from 1Password (`op://Paradigm/Pagerduty api key/password`), TTL-based refresh via `CredStore`; refresh via `POST /auth/pagerduty/refresh`
- `PAGERDUTY_TOKEN` declared in `server.yaml` with `cache: "pagerduty/token"`; `PAGERDUTY_TOKEN_SOURCE` in `local.yaml` `literal_env`

## v0.3.4

- Add `sentry` adapter to `tools-api` — query Sentry issues and events via `POST /invoke {tool: "sentry", action: ...}`
- Actions: `list_issues` (with Sentry query syntax and project slug filter), `get_issue`, `get_issue_events` (with stack frames), `list_projects`
- Auth: Bearer token from 1Password (`op://Private/Sentry MCP/SENTRY_AUTH_TOKEN`), TTL-based refresh via `CredStore`; refresh via `POST /auth/sentry/refresh`
- Project slug → numeric ID auto-resolved in `list_issues`; `SENTRY_ORG` defaults to `paradigmco`

## v0.3.3

- Add `POST /mode` (bulk) — set readonly/readwrite on all adapters at once
- Add `textserve mode <server> --all readonly|readwrite` — CLI bulk flip

## v0.3.2

- Add `POST /mode/{tool}` to `tools-api` — set readonly/readwrite on a live adapter without restart; resets to `tools.yaml` default on next restart
- Add `textserve mode <server> <tool> readonly|readwrite` CLI command

## v0.3.1

- Fix `jenkins` adapter `trigger` action: use `buildWithParameters` endpoint when `build_params` dict is provided; falls back to `/build` for non-parameterized jobs
- Set `jenkins` adapter `readonly: false` in `tools.yaml`

## v0.3.0

Milestone: unified tool gateway with complete adapter set, secure TTL-based credential management, and structured activity logging.

- `tools-api` adapters: datadog, snowflake, athena, jenkins — all validated end-to-end
- All adapters use TTL-based credential refresh from 1Password; no infinite in-memory tokens
- Structured JSON activity log at `~/.local/log/tools-api.log` (configurable via `TOOLS_API_LOG`)
- Single `POST /invoke`, `GET /describe`, `GET /health`, `GET /permissions`, `POST /auth/{tool}/refresh`

## v0.2.11

- Add structured activity logging to `~/.local/log/tools-api.log` — one JSON line per `invoke` and `refresh` event: `ts`, `event`, `tool`, `action`, `status`, `cached`, `duration_ms`, `error`
- Log path configurable via `TOOLS_API_LOG` env var

## v0.2.10

- Fix jenkins adapter: TTL-based credential refresh via `CredStore` — token re-read from 1Password after `JENKINS_CRED_TTL_HOURS` (default 4 h); `JENKINS_TOKEN_SOURCE` required

## v0.2.9

- Add `jenkins` adapter to `tools-api` — query Jenkins CI via `POST /invoke {tool: "jenkins", action: ...}`
- Actions: `list_jobs` (with optional folder), `get_job`, `get_build`, `get_log` (tail with `max_chars`), `trigger` (requires `readonly: false`)
- Auth: Basic token loaded from 1Password at startup via `JENKINS_TOKEN`; re-readable via `POST /auth/jenkins/refresh` using `JENKINS_TOKEN_SOURCE`
- 10 s response cache; `JENKINS_URL` and `JENKINS_USER` configurable in `server.yaml`

## v0.2.8

- Retire `datadog-api` (port 10892) and `snowflake-api` (port 10891) — archived to `servers/_archived/`; functionality absorbed by `tools-api`

## v0.2.7

- Fix process stop: use `Setpgid: true` on start so SIGTERM reaches all child processes (fixes uvicorn port-not-released on restart); fall back to single-PID SIGTERM for legacy processes; wait up to 2s for exit before returning
- Archive `airflow` server → `servers/_archived/airflow`; remove from registry

## v0.2.6

- Add `started_at` (ISO 8601 UTC) to `GET /health` response in `tools-api`
- `textserve status` uptime for process servers: prefer `started_at` from health endpoint over `ps` when available

## v0.2.5

- Fix `textserve register`: skip servers with `no_register: true` instead of silently registering them

## v0.2.4

- Fix `textserve restart`: respect `no_register` — always deregisters stale entries on stop but skips re-registration for `no_register: true` servers
- Fix `textserve status`: REG column shows `-` for `no_register: true` servers instead of `✓`/`✗`

## v0.2.3

- Auto-run `uv sync` on `textserve start`/`up` when `native_venv` is configured but the venv directory is missing

## v0.2.2

- Fix `textserve down`: always deregister on stop, regardless of `no_register` — cleans up stale Claude config entries left from before the flag was set
- Fix `textserve up`: servers with `no_register: true` that are running-but-not-registered now correctly return "skipped" instead of "registered"
- Rename `tool-gateway` → `tools-api` (directory, registry key, pid file, env var `TOOLS_API_PORT`)

## v0.2.1

- Fix athena `POST /auth/athena/refresh`: run `aws sso login` without `--profile` (SSO session is shared; user manages auth externally if preferred)
- Add `region_overrides` param to athena adapter — same `"env=region,..."` format as `role_overrides`; configured via `ATHENA_REGION_OVERRIDES` env var: `paradigm-dev/testnet/prod → eu-west-2`, `paradigm-data → us-east-1`
- Fix athena role override for `paradigm-data`: `analyst` → `admin`
- Remove `ATHENA_SSO_PROFILE` env var from server.yaml

## v0.2.0

Unified tool gateway — replaces per-tool standalone HTTP servers with a single pluggable adapter architecture.

- `servers/tool-gateway` — FastAPI gateway on port 10893; single `POST /invoke {tool, action, params}` endpoint
- Standard response envelope: `{tool, action, cached, result}` / `{tool, action, error, refresh_url}`
- `POST /auth/{tool}/refresh` per-adapter re-auth; `POST /reload` hot-adds tools without restart
- `GET /describe` — agent-readable tool schemas, param definitions, examples, and current status
- `GET /health`, `GET /permissions` across all adapters

**Adapters:**
- `datadog` — token-cached auth via 1Password (`op read`), 10 s response cache, metrics/logs/events queries
- `snowflake` — SSO lazy init (no browser popup on startup), read-only with NIGHTLY* exception
- `athena` — AWS SSO lazy init (shared session across all profiles), `query_logs` with partition-aware time filter and k8s namespace filter, `raw_query` for custom SQL; covers all clusters (paradex-dev/testnet/prod, paradigm-dev/testnet/prod/data) via configurable `role` and `role_overrides`

Standalone `datadog-api` (port 10892) and `snowflake-api` (port 10891) remain running pending retirement decision.

## v0.1.26

- Fix athena `query_logs`: column values are stored as `key=value` strings — switch `app_name` filter to `LIKE '%=<name>'`
- Fix athena workgroup: `paradex_dev_vector_logs` not `paradex_dev_access_logs`; workgroup enforces output — drop `ResultConfiguration` from default queries
- Add `k8s_env` param to `query_logs` — filters by `meta_env` (k8s namespace, e.g. `nightly`)
- Make `app_name` optional in `query_logs` — omit to search across all services
- Add `meta_app_name` to `query_logs` SELECT output

## v0.1.25

- Fix athena adapter: drop hardcoded prefix/envs — `environment` param is now the full cluster name (e.g. `paradex-testnet`, `paradigm-data`); database/workgroup derived via slug replace
- Add `role` param (default `admin`) — profile becomes `{env}.{role}`; override per-entry in tools.yaml for clusters where only `analyst`/`engineer` access is available
- `sso_profile` now explicit in tools.yaml

## v0.1.24

- Add `athena` adapter to tool-gateway — query application logs from AWS Athena via `POST /invoke {tool: "athena", action: "query_logs" | "raw_query"}`
- Single adapter covers all environments (testnet/staging/prod) via per-env boto3 sessions; `environment` is a query param
- Lazy init like snowflake — `POST /auth/athena/refresh` runs `aws sso login` once (SSO session is shared across all profiles); no AWS prompts on gateway startup
- `query_logs` action: high-level (app_name, search_pattern, hours_ago, limit) — maps to `paradex_<env>_vector_logs` table with partition-aware time filter
- `raw_query` action: pass arbitrary SQL with optional workgroup/output_location overrides
- 10 s response cache per (action, env, params)

## v0.1.23

- Add `GET /describe` to tool-gateway — returns tool schemas, param definitions, examples, and current status per adapter; entry point for agents discovering available tools

## v0.1.22

- Add `servers/tool-gateway` — unified FastAPI gateway with pluggable adapters
- Single `POST /invoke {tool, action, params}` endpoint dispatches to registered adapters
- Adapters: `datadog` (token-cached, 10 s response cache) and `snowflake` (SSO, lazy init)
- Standard response envelope: `{tool, action, cached, result}` / `{tool, action, error, refresh_url}`
- `POST /auth/{tool}/refresh` per-adapter re-auth; `POST /reload` hot-adds new tools from `tools.yaml`
- Snowflake adapter starts uninitialized — no SSO popup on gateway startup
- `no_register: true`, port 10893; standalone `datadog-api` and `snowflake-api` remain running

## v0.1.21

- Add `servers/datadog-api` — FastAPI HTTP server for Datadog metrics, logs, and events queries
- Token-cached auth: API/App keys loaded from 1Password at startup, cached in memory, refreshable via `POST /auth/refresh`; TTL controlled by `DD_CRED_TTL_HOURS` (default 4 h)
- Endpoints: `POST /query` (metrics | logs | events), `POST /auth/refresh`, `GET /permissions`, `GET /health`
- `no_register: true` — starts/stops via textserve but never registered as a Claude MCP server
- Add `datadog-api` entry to `registry.yaml`

## v0.1.20

- Add `literal_env` section to `local.yaml` server overrides — values are passed to the server as-is without op:// resolution, for cases where the server itself needs to call `op read` (e.g. credential refresh paths)

## v0.1.19

- Add `no_register: true` flag to `server.yaml` — servers with this flag are started but never registered as MCP servers in the Claude config (`up`, `down`, `start` all respect it)
- Set `no_register: true` on `snowflake-api` (HTTP API server, not an MCP endpoint)

## v0.1.18

- Add `servers/airflow` — Airflow MWAA MCP server (Python); authenticates via `create_web_login_token`, exposes DAG/run/task/log/variable tools with optional read-only mode; runs streamable-http on port 9000
- Add `servers/notion` — Notion MCP server via `supergateway` wrapping `@notionhq/notion-mcp-server`; exposes streamable-http on port 3000
- Add `servers/paradex-db` — read-only PostgreSQL MCP server (Node.js) with automatic RDS IAM token refresh; supports SSE and stdio transports
- Add `servers/slack-search` — Slack search MCP server (Python); wraps `search.messages` and `search.files` with channel filtering and pagination
- Simplify `servers/snowflake/Dockerfile`: clone upstream `Snowflake-Labs/mcp` at build time instead of requiring a local source copy; drop lockfile pinning in favour of `uv sync --upgrade`

## v0.1.17

- Fix default Claude config path: was incorrectly falling back to `~/.claude-work/.claude.json`; now correctly uses `~/.claude.json` (respects `$CLAUDE_CONFIG_DIR` as before)

## v0.1.16

- Add `--profile <name>` persistent flag — resolves a textaccounts profile to a `CLAUDE_CONFIG_DIR` path before any command runs
- Reads `~/.textaccounts/profiles.yaml` directly (alias-aware); fails with a clear error if textaccounts is not configured or the profile is not found
- Applies to all subcommands: `up`, `down`, `start`, `stop`, `register`, `deregister`, `profile use`, etc.

## v0.1.15

- Add `textserve remove <name>` — remove an MCP server entry from a Claude config file
- Flags: `--global` (default), `--repo <path>` (project `.claude/settings.json`), `--all`, `--dry-run`
- Idempotent: exits 0 with a clear message when the entry is not found
- Atomic write via temp file + rename — invalid JSON is never left behind

## v0.1.14

- Add `textserve profile` command — `list`, `show`, and `use` subcommands
- `textserve profile use <name>` converges the fleet: brings up profile servers, brings down the rest
- Profiles defined in `registry.yaml` under `profiles:` — support explicit `servers:` lists and `tags:` expansion
- Fix `CLAUDE_CONFIG_DIR`: `register`/`deregister` now write to the active textaccounts profile's `claude.json` when `$CLAUDE_CONFIG_DIR` is set
- Extract `downServer()` helper (used by both `down` and `profile use`)

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
