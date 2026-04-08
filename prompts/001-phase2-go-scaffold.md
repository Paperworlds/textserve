---
id: "001"
title: "Phase 2 — Go module + registry package + server.yaml migration"
phase: "phase-2"
repo: "mcp-fleet"
model: "sonnet"
depends_on: ["000"]
budget_usd: 3.00
---

# Phase 2 — Go Module & Registry

Set up the Go module, write the `internal/registry` package, and migrate all
`servers/<name>/server.yaml` files to the new declarative schema (env vars,
volumes, extra_args — no more shell hooks for credential injection).

## Context

The project is switching from bash to Go. `registry.yaml` (already written) is
the fleet-level source of truth. `servers/<name>/server.yaml` are per-server
overrides with richer operational config. The existing `hook.sh` files in each
server dir do credential injection via shell variable exports — that logic moves
into `server.yaml` as structured data. Hook files may remain for pure side-effect
scripts (port-forwards, pre-start checks), but credential injection is now
declarative.

Read the existing hook files at `~/projects/personal/skills/locals/mcp-hooks/`
to understand what env vars each server needs and where they come from.

## Tasks

### 1. Go module setup

Create `go.mod` with module `github.com/pdonorio/mcp-fleet`, Go 1.22+.

Dependencies (add to go.mod/go.sum):
- `gopkg.in/yaml.v3` — YAML parsing
- `github.com/spf13/cobra` — CLI framework
- `github.com/olekukonko/tablewriter` — status table output

Directory structure to create:
```
cmd/mcpf/          # main package (stub main.go for now)
internal/
  registry/        # registry + server config parsing
  op/              # 1Password credential fetching + caching
```

### 2. server.yaml schema (new format)

Each `servers/<name>/server.yaml` must be updated to the full declarative schema.
The `env` block replaces all credential injection from hook.sh.

**Env var source types:**
- `value: "literal"` — static value
- `value_template: "${OTHER_VAR}/suffix"` — expanded after all other env vars resolved
- `op: "op://Vault/Item/field"` + `cache: "service/field"` — read from 1Password,
  cache at `~/.cache/mcp-<cache>` (create dir 700, file 600)
- `cache_file: "service/field"` — read from `~/.cache/<cache_file>` (pre-populated
  externally, e.g. paradex-db)

**Full schema:**
```yaml
# servers/<name>/server.yaml
image: ""                    # docker image (omit for native/stdio)
transport: http              # http | stdio | native
port: 0                      # host port
container_port: 0            # container port (omit if same as port)
endpoint_path: /mcp          # URL path for claude mcp add
tags: []
network: ""                  # optional: "host"
env:
  - name: MY_VAR
    value: "static"
  - name: MY_SECRET
    op: "op://Private/Item/field"
    cache: "service/fieldname"
  - name: MY_TEMPLATE
    value_template: '{"key":"${MY_SECRET}"}'
  - name: FROM_CACHE
    cache_file: "service/fieldname"
volumes:
  - host: "${HOME}/path"
    container: "/container/path"
    readonly: true
    resolve_symlinks: true   # resolve symlink before mounting (docker limitation)
extra_args: []               # appended after image in docker run
pre_start: ""                # path to shell script for side effects (port-forward etc)
pid_file: ""                 # for native transport only
native_cmd: ""               # for native transport: path to executable
native_args: []              # args for native command
native_venv: ""              # if set, prepend <venv>/bin to PATH
deps:
  - cmd: ""
    hint: ""
health:
  endpoint: /health          # omit for stdio servers
  probe: ""                  # "tool-list" for stdio
  timeout: 5
managed_by: ""               # "claude" for stdio servers
```

### 3. Update all 12 server.yaml files

Migrate each server's hook logic into the new schema. Read the existing
`~/projects/personal/skills/locals/mcp-hooks/<name>.sh` for each server.

Key mappings per server:

**jenkins** — env: JENKINS_URL (static), JENKINS_USERNAME (op cached), JENKINS_TOKEN (op cached);
extra_args: `--transport streamable-http --jenkins-url ${JENKINS_URL} --jenkins-username ${JENKINS_USERNAME} --jenkins-password ${JENKINS_TOKEN}`

**snowflake** — env: SNOWFLAKE_ACCOUNT/USER/PASSWORD (all op cached via item ID
`op://Private/REDACTED`); volume: `${HOME}/.mcp/snowflake-tools.yaml →
/app/services/tools_config.yaml:ro`

**grafana** — env: GRAFANA_URL (static: https://paradigmconnect.grafana.net),
GRAFANA_SERVICE_ACCOUNT_TOKEN (op cached); extra_args: `-transport streamable-http -address 0.0.0.0:8000`

**grafana-pdx** — same as grafana but GRAFANA_URL=https://paradex.grafana.net,
op item "Grafana Paradex", cache key "grafana-pdx/token"

**notion** — env: NOTION_TOKEN (op cached), OPENAPI_MCP_HEADERS
(value_template: `{"Authorization":"Bearer ${NOTION_TOKEN}","Notion-Version":"2022-06-28"}`)

**airbyte** — env: AIRBYTE_API_URL (static: http://host.docker.internal:18001);
pre_start: servers/airbyte/pre-start.sh (see below); deps: kubectl check

**slack** — env: SLACK_BOT_TOKEN (op cached), SLACK_TEAM_ID (static: T072ZU3U7),
AUTH_TOKEN (static: mcp-slack-local); extra_args: `--transport http --port 9000`

**slack-search** — env: SLACK_USER_TOKEN (op cached: `op://Private/Slack Bot Token/user-token`,
cache: slack-search/user-token)

**datadog** — env: DD_API_KEY/DD_APP_KEY (op cached), DD_SITE=datadoghq.com,
MCP_TRANSPORT=http, MCP_PORT=9000, MCP_HOST=0.0.0.0

**paradex-db** — env: DB_HOST/DB_USER (cache_file from paradex-db-mcp/), DB_NAME=paradex,
DB_PORT=5432, AWS_REGION=ap-northeast-1, AWS_PROFILE=paradex-prod.basic, MCP_MODE=sse;
volumes: ${HOME}/.aws/config (resolve_symlinks:true), ${HOME}/.aws/sso; network: host;
deps: WARP check

**airflow** — transport: native; native_cmd: python; native_venv:
${HOME}/projects/personal/skills/locals/docker/mcp-airflow/.venv;
native_args: [${HOME}/projects/personal/skills/locals/docker/mcp-airflow/server.py];
pid_file: /tmp/.mcp-airflow.pid; env: AWS_PROFILE, AWS_REGION, MWAA_MCP_READONLY=true,
FASTMCP_LOG_LEVEL=ERROR, MCP_PORT=9894; deps: WARP check + AWS SSO check

**sentry** — transport: stdio; managed_by: claude; no env needed (Claude manages it)

Also create `servers/airbyte/pre-start.sh`:
```bash
#!/usr/bin/env bash
# Start kubectl port-forward for Airbyte before docker run
AIRBYTE_PF_PORT="${AIRBYTE_PF_PORT:-18001}"
AIRBYTE_PF_PIDFILE="/tmp/.airbyte-pf.pid"
[[ -f "$AIRBYTE_PF_PIDFILE" ]] && kill "$(cat "$AIRBYTE_PF_PIDFILE")" 2>/dev/null || true
kubectl port-forward svc/airbyte-helmv2-airbyte-server-svc "${AIRBYTE_PF_PORT}:8001" \
  -n airbyte-helmv2 &>/dev/null &
echo $! > "$AIRBYTE_PF_PIDFILE"
sleep 2
kill -0 "$(cat "$AIRBYTE_PF_PIDFILE")" 2>/dev/null || { echo "port-forward failed" >&2; exit 1; }
```

### 4. internal/registry package

`internal/registry/registry.go`:
- `ServerConfig` struct with yaml tags matching the server.yaml schema above
- `FleetRegistry` struct: `Servers map[string]RegistryEntry` matching registry.yaml
- `Load(path string) (*FleetRegistry, error)` — parse registry.yaml
- `LoadServer(repoRoot, name string) (*ServerConfig, error)` — parse servers/<name>/server.yaml
- `ListNames() []string`
- `FilterByTag(tag string) []string`

`internal/registry/registry_test.go`:
- Test Load parses all 12 servers from registry.yaml
- Test FilterByTag("docker") returns 10 servers (not airflow, not sentry)
- Test LoadServer("jenkins") has port 9887 and 3 env entries

### 5. internal/op package

`internal/op/op.go`:
- `Read(uri string) (string, error)` — exec `op read <uri>`
- `Cached(service, field, opURI string) (string, error)` — read from
  `~/.cache/mcp-<service>/<field>`, populate from `op read` if missing
- `CacheFileRead(cacheRelPath string) (string, error)` — read from
  `~/.cache/<cacheRelPath>` (for cache_file type vars)

### 6. Update Justfile and install.sh

Justfile:
```
build:
    go build -o bin/mcpf ./cmd/mcpf

test:
    go test ./...

lint:
    go vet ./...

install: build
    ln -sf $(pwd)/bin/mcpf ~/.local/bin/mcpf
```

install.sh: build then symlink (keep as bash wrapper for convenience).

### 7. Cleanup Phase 1 bash artifacts

Delete: `lib/registry.sh`, `lib/docker.sh`, `lib/claude.sh`, `lib/health.sh`,
`lib/preflight.sh`, `tests/test_registry.sh`, `tests/test_health.sh`.
The `lib/` directory can be removed if empty.

## Completion gate

Run `go test ./...` — all tests must pass.
Run `go vet ./...` — no errors.
Run `just build` — binary must compile cleanly.
