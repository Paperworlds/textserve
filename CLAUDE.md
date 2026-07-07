# textserve — development notes

## Commit rules
- GPG sign every commit (`git commit -S`)
- No `Co-Authored-By` lines
- Never commit secrets, op:// paths, or credentials

## server.yaml env vars
Empty `value: ""` entries cause `textserve` to fail with "no resolvable source" at start time — the resolver treats an empty value as missing. **Do not add env entries with `value: ""`** for optional/override-only vars. Leave them out of `server.yaml` entirely and handle the default in code (e.g. `os.environ.get("VAR", "default")`). Document any such env var in a comment in `server.yaml` if needed.

## tools-api mode flip
`POST /mode/{tool}` and `POST /mode` (bulk) set `adapter._readonly` directly on the live instance. All adapters expose `_readonly` as a plain attribute — no interface method needed. State resets to `tools.yaml` on restart. The `GET /permissions` response reflects the live value immediately.

## tools-api adapter rules
- All adapters must use TTL-based credential refresh — no infinite in-memory tokens
- Credentials read from env at startup; re-read from `op://` source on TTL expiry or explicit `POST /auth/{tool}/refresh`
- `JENKINS_TOKEN_SOURCE`, `DD_API_KEY_SOURCE`, `DD_APP_KEY_SOURCE` are passed via `literal_env` in `local.yaml` (not resolved by textserve — passed as-is for the adapter's own `op read` calls)
- Activity log: `~/.local/log/tools-api.log`, one JSON line per invoke/refresh event; path configurable via `TOOLS_API_LOG` env var
