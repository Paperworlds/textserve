# tool-gateway

Unified HTTP gateway for Datadog and Snowflake queries. Single `POST /invoke` endpoint
dispatches to pluggable adapters defined in `tools.yaml`. Standalone servers (`datadog-api`,
`snowflake-api`) remain running until this is validated — do not retire them first.

## Usage

```bash
textserve up tool-gateway
```

Snowflake requires an explicit init before use (no SSO popup on startup):

```bash
curl -X POST http://localhost:10893/auth/snowflake/refresh   # opens browser
```

## Setup

Add to `~/.local/paperworlds/textserve/local.yaml`:

```yaml
servers:
  tool-gateway:
    env:
      DD_API_KEY: "op://Vault/Item/field"
      DD_APP_KEY: "op://Vault/Item/field"
      SNOWFLAKE_ACCOUNT: "op://Vault/Item/field"
      SNOWFLAKE_USER: "op://Vault/Item/field"
    literal_env:
      DD_API_KEY_SOURCE: "op://Vault/Item/field"
      DD_APP_KEY_SOURCE: "op://Vault/Item/field"
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/invoke` | Dispatch a query to a tool adapter |
| `POST` | `/auth/{tool}/refresh` | Re-authenticate a specific tool |
| `POST` | `/reload` | Hot-add new tools from tools.yaml without restart |
| `GET` | `/health` | Status of all adapters |
| `GET` | `/permissions` | Readonly flags and capabilities per tool |

### POST /invoke

```json
{
  "tool": "datadog",
  "action": "query",
  "params": {
    "type": "metrics",
    "query": "avg:system.cpu.user{*}",
    "from_ts": 1716000000,
    "to_ts": 1716003600
  }
}
```

Response envelope:

```json
{
  "tool": "datadog",
  "action": "query",
  "cached": false,
  "result": { ...native response... }
}
```

Error envelope (no `result` key):

```json
{
  "tool": "datadog",
  "action": "query",
  "error": "cred_expired",
  "refresh_url": "/auth/datadog/refresh"
}
```

### Snowflake params

```json
{
  "tool": "snowflake",
  "action": "query",
  "params": {
    "sql": "SELECT ...",
    "warehouse": "DATA_ANALYSIS",
    "role": "ANALYST",
    "database": "PROD",
    "schema_name": "PUBLIC"
  }
}
```

Snowflake adapter starts `uninitialized` — call `POST /auth/snowflake/refresh` first.

### POST /reload

Adds tools defined in `tools.yaml` that are not yet loaded. Does not replace or restart
existing adapters (their auth state is preserved). Use after adding a new tool entry.

## tools.yaml

Adapter registry loaded at startup. Each entry maps to a class in `adapters/`.

```yaml
tools:
  datadog:
    adapter: datadog
    readonly: true
    params:
      site: "${DD_SITE}"
      cred_ttl_hours: "${DD_CRED_TTL_HOURS}"

  snowflake:
    adapter: snowflake
    readonly: true
    readonly_exceptions:
      - "NIGHTLY*"
    params: {}
```

## Adding a new adapter

1. Add `adapters/<name>.py` implementing `query()`, `refresh()`, `health()`, `permissions()`
2. Register the class in `_ADAPTER_CLASSES` in `server.py`
3. Add an entry to `tools.yaml`
4. Call `POST /reload` or restart
