import subprocess
import threading
import time
from datetime import datetime, timedelta, timezone
from typing import Any

import boto3

_CACHE_TTL = 10.0


class AthenaAdapter:
    def __init__(self, region: str = "ap-northeast-1", sso_profile: str = "",
                 role: str = "poweruser", role_overrides: str = "",
                 region_overrides: str = "",
                 readonly: bool = True, readonly_exceptions: list = []):
        self._region = region
        self._sso_profile = sso_profile
        self._role = role
        self._role_overrides: dict[str, str] = {}
        self._region_overrides: dict[str, str] = {}
        for overrides_str, target in [(role_overrides, self._role_overrides),
                                      (region_overrides, self._region_overrides)]:
            for pair in overrides_str.split(","):
                if "=" in pair:
                    k, v = pair.split("=", 1)
                    target[k.strip()] = v.strip()
        self._readonly = readonly
        self._sessions: dict[str, boto3.Session] = {}
        self._initialized = False
        self._init_lock = threading.Lock()
        self._cache: dict[tuple, tuple[float, Any]] = {}
        self._cache_lock = threading.Lock()

    @staticmethod
    def _slug(env: str) -> str:
        return env.replace("-", "_")

    def _workgroup(self, env: str) -> str:
        return f"{self._slug(env)}_vector_logs"

    def _database(self, env: str) -> str:
        return f"{self._slug(env)}_vector_logs"

    def _table(self, env: str) -> str:
        return f"{self._slug(env)}_vector_logs"

    def _client(self, env: str):
        if env not in self._sessions:
            role = self._role_overrides.get(env, self._role)
            region = self._region_overrides.get(env, self._region)
            self._sessions[env] = boto3.Session(
                profile_name=f"{env}.{role}",
                region_name=region,
            )
        return self._sessions[env].client("athena")

    def _cache_get(self, key: tuple) -> Any | None:
        with self._cache_lock:
            entry = self._cache.get(key)
            if entry and time.monotonic() - entry[0] < _CACHE_TTL:
                return entry[1]
            return None

    def _cache_set(self, key: tuple, data: Any) -> None:
        with self._cache_lock:
            self._cache[key] = (time.monotonic(), data)

    def _is_auth_error(self, exc: Exception) -> bool:
        msg = str(exc).lower()
        return any(k in msg for k in ("sso", "token", "unauthorized", "credential", "expired"))

    @staticmethod
    def _time_clauses(hours_ago: int) -> str:
        now = datetime.now(timezone.utc)
        start = now.replace(minute=0, second=0, microsecond=0) - timedelta(hours=hours_ago)
        seen: set[tuple] = set()
        points: list[tuple] = []
        t = start
        while t <= now:
            key = (t.year, t.month, t.day, t.hour)
            if key not in seen:
                seen.add(key)
                points.append(key)
            t += timedelta(hours=1)
        if len(points) == 1:
            y, mo, d, h = points[0]
            return f"year = {y} AND month = {mo} AND day = {d} AND hour = {h}"
        clauses = " OR ".join(
            f"(year = {y} AND month = {mo} AND day = {d} AND hour = {h})"
            for y, mo, d, h in points
        )
        return f"({clauses})"

    def _wait_for_query(self, client, query_id: str, timeout: int = 60) -> None:
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            resp = client.get_query_execution(QueryExecutionId=query_id)
            state = resp["QueryExecution"]["Status"]["State"]
            if state == "SUCCEEDED":
                return
            if state in ("FAILED", "CANCELLED"):
                reason = resp["QueryExecution"]["Status"].get("StateChangeReason", state)
                raise RuntimeError(f"athena query {state}: {reason}")
            time.sleep(2)
        raise TimeoutError(f"athena query {query_id} did not complete in {timeout}s")

    def _run_sql(self, env: str, sql: str, workgroup: str | None = None, output: str | None = None) -> dict:
        client = self._client(env)
        kwargs: dict = {
            "QueryString": sql,
            "WorkGroup": workgroup or self._workgroup(env),
        }
        if output:
            kwargs["ResultConfiguration"] = {"OutputLocation": output}
        resp = client.start_query_execution(**kwargs)
        query_id = resp["QueryExecutionId"]
        self._wait_for_query(client, query_id)
        results = client.get_query_results(QueryExecutionId=query_id)
        rows = results["ResultSet"]["Rows"]
        if not rows:
            return {"columns": [], "rows": [], "row_count": 0, "query_id": query_id}
        columns = [col.get("VarCharValue", "") for col in rows[0]["Data"]]
        data_rows = [
            [col.get("VarCharValue", "") for col in row["Data"]]
            for row in rows[1:]
        ]
        return {"columns": columns, "rows": data_rows, "row_count": len(data_rows), "query_id": query_id}

    def _run_query(self, action: str, env: str, sql: str,
                   workgroup: str | None = None, output: str | None = None) -> tuple[Any, bool]:
        cache_key = (action, env, sql, workgroup, output)
        if hit := self._cache_get(cache_key):
            return hit, True
        try:
            result = self._run_sql(env, sql, workgroup=workgroup, output=output)
            self._cache_set(cache_key, result)
            return result, False
        except Exception as exc:
            if self._is_auth_error(exc):
                with self._init_lock:
                    self._initialized = False
                raise PermissionError("auth_expired") from exc
            raise

    def query(self, action: str, params: dict) -> tuple[Any, bool]:
        if not self._initialized:
            raise PermissionError("not_initialized")

        env = params.get("environment", "")
        if not env:
            raise ValueError("environment is required")

        # Fail loud on typo'd/unknown params. Without this the filters
        # below silently no-op (e.g. `service`/`query`/`hours` instead of
        # `app_name`/`search_pattern`/`hours_ago`) and the caller gets an
        # env-wide dump that looks like a successful targeted query.
        allowed_params = {
            "query_logs": {"environment", "app_name", "search_pattern",
                           "k8s_env", "hours_ago", "limit"},
            "raw_query": {"environment", "sql", "workgroup", "output_location"},
        }
        if action in allowed_params:
            unknown = set(params) - allowed_params[action]
            if unknown:
                raise ValueError(
                    f"unknown param(s) for {action}: {', '.join(sorted(unknown))}; "
                    f"allowed: {', '.join(sorted(allowed_params[action]))}"
                )

        if action == "query_logs":
            app_name = params.get("app_name", "")
            pattern = params.get("search_pattern", "")
            k8s_env = params.get("k8s_env", "")
            hours_ago = int(params.get("hours_ago", 1))
            limit = int(params.get("limit", 50))
            db = self._database(env)
            tbl = self._table(env)
            time_filter = self._time_clauses(hours_ago)

            def esc(s: str) -> str:
                return s.replace("'", "''")

            filters = [time_filter]
            if app_name:
                filters.append(f"meta_app_name LIKE '%={esc(app_name)}'")
            if k8s_env:
                filters.append(f"meta_env LIKE '%={esc(k8s_env)}'")
            if pattern:
                filters.append(f"message LIKE '%{esc(pattern)}%'")

            sql = (
                f"SELECT timestamp, message, meta_app_name, meta_pod_name "
                f"FROM {db}.{tbl} "
                f"WHERE {' AND '.join(filters)} "
                f"ORDER BY timestamp DESC "
                f"LIMIT {limit}"
            )
            return self._run_query(action, env, sql)

        elif action == "raw_query":
            sql = params.get("sql", "").strip()
            if not sql:
                raise ValueError("sql is required")
            return self._run_query(action, env, sql,
                                   workgroup=params.get("workgroup"),
                                   output=params.get("output_location"))

        else:
            raise NotImplementedError(f"unknown action: {action}")

    _LOGIN_MAX_ATTEMPTS = 3
    _LOGIN_TIMEOUT_S = 120

    def refresh(self) -> None:
        cmd = ["aws", "sso", "login"]
        if self._sso_profile:
            cmd += ["--profile", self._sso_profile]
        # `aws sso login` blocks on an interactive browser device flow, so a
        # bare run() can hang forever. Bound each attempt with a timeout and
        # give up after 3 tries instead of wedging the gateway.
        last_err: Exception | None = None
        for attempt in range(1, self._LOGIN_MAX_ATTEMPTS + 1):
            try:
                subprocess.run(cmd, check=True, timeout=self._LOGIN_TIMEOUT_S)
                with self._init_lock:
                    self._initialized = True
                return
            except (subprocess.CalledProcessError, subprocess.TimeoutExpired) as exc:
                last_err = exc
        raise RuntimeError(
            f"aws sso login failed after {self._LOGIN_MAX_ATTEMPTS} attempts: {last_err}"
        )

    def health(self) -> dict:
        if not self._initialized:
            return {"status": "uninitialized", "requires": "POST /auth/athena/refresh"}
        return {"status": "ok"}

    def permissions(self) -> dict:
        return {"readonly": self._readonly, "initialized": self._initialized}

    def describe(self) -> dict:
        return {
            "description": "Query application logs from AWS Athena — requires POST /auth/athena/refresh before first use",
            "actions": {
                "query_logs": {
                    "params": {
                        "environment": {"type": "string", "required": True,
                                        "example": "paradex-testnet",
                                        "note": "full cluster name — paradex-dev, paradex-testnet, paradex-prod, paradigm-dev, paradigm-testnet, paradigm-prod, paradigm-data"},
                        "app_name": {"type": "string", "example": "position-management-service",
                                     "note": "filter by service name; omit to search across all apps"},
                        "k8s_env": {"type": "string", "example": "nightly",
                                    "note": "filter by k8s namespace/env (meta_env column)"},
                        "search_pattern": {"type": "string", "example": "No price available",
                                           "note": "substring match against log message"},
                        "hours_ago": {"type": "integer", "default": 1, "example": 4},
                        "limit": {"type": "integer", "default": 50, "example": 100},
                    }
                },
                "raw_query": {
                    "params": {
                        "environment": {"type": "string", "required": True,
                                        "example": "paradex-testnet"},
                        "sql": {"type": "string", "required": True,
                                "example": "SELECT timestamp, message FROM paradex_testnet_vector_logs.paradex_testnet_vector_logs WHERE year=2026 AND month=5 AND day=19 AND hour=12 LIMIT 10"},
                        "workgroup": {"type": "string"},
                        "output_location": {"type": "string"},
                    }
                },
            },
        }
