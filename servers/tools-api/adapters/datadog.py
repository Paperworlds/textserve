import os
import subprocess
import threading
import time
from datetime import datetime, timezone
from typing import Any

import httpx

_CACHE_TTL = 10.0
_AUTH_ERROR_CODES = {403}


class CredStore:
    def __init__(self, api_key: str, app_key: str, api_key_source: str, app_key_source: str, ttl: float):
        self._api_key_source = api_key_source
        self._app_key_source = app_key_source
        self._ttl = ttl
        self._api_key = api_key
        self._app_key = app_key
        self._loaded_at = time.monotonic()
        self._lock = threading.Lock()

    def _expired(self) -> bool:
        return time.monotonic() - self._loaded_at > self._ttl

    def _reload(self) -> None:
        self._api_key = subprocess.check_output(["op", "read", self._api_key_source], text=True).strip()
        self._app_key = subprocess.check_output(["op", "read", self._app_key_source], text=True).strip()
        self._loaded_at = time.monotonic()

    def get(self) -> tuple[str, str]:
        with self._lock:
            if self._expired():
                self._reload()
            return self._api_key, self._app_key

    def refresh(self) -> None:
        with self._lock:
            self._reload()


class DatadogAdapter:
    def __init__(self, site: str = "datadoghq.com", cred_ttl_hours: str = "4",
                 readonly: bool = True, readonly_exceptions: list = []):
        self._site = site
        self._readonly = readonly
        self._creds = CredStore(
            api_key=os.environ["DD_API_KEY"],
            app_key=os.environ["DD_APP_KEY"],
            api_key_source=os.environ["DD_API_KEY_SOURCE"],
            app_key_source=os.environ["DD_APP_KEY_SOURCE"],
            ttl=float(cred_ttl_hours) * 3600,
        )
        self._cache: dict[tuple, tuple[float, Any]] = {}
        self._cache_lock = threading.Lock()

    def _cache_get(self, key: tuple) -> Any | None:
        with self._cache_lock:
            entry = self._cache.get(key)
            if entry and time.monotonic() - entry[0] < _CACHE_TTL:
                return entry[1]
            return None

    def _cache_set(self, key: tuple, data: Any) -> None:
        with self._cache_lock:
            self._cache[key] = (time.monotonic(), data)

    def _headers(self) -> dict[str, str]:
        try:
            api_key, app_key = self._creds.get()
        except subprocess.CalledProcessError as exc:
            raise PermissionError("cred_refresh_failed") from exc
        return {"DD-API-KEY": api_key, "DD-APPLICATION-KEY": app_key}

    @staticmethod
    def _utc_iso(ts: int) -> str:
        return datetime.fromtimestamp(ts, tz=timezone.utc).isoformat()

    def query(self, action: str, params: dict) -> tuple[Any, bool]:
        if action != "query":
            raise NotImplementedError(f"unknown action: {action}")

        q_type = params.get("type")
        q = params.get("query", "")
        from_ts = params["from_ts"]
        to_ts = params["to_ts"]

        cache_key = (q_type, q, from_ts, to_ts)
        if hit := self._cache_get(cache_key):
            return hit, True

        headers = self._headers()
        base = f"https://api.{self._site}"
        try:
            if q_type == "metrics":
                resp = httpx.get(f"{base}/api/v1/query", headers=headers,
                                 params={"query": q, "from": from_ts, "to": to_ts}, timeout=30)
            elif q_type == "logs":
                resp = httpx.post(f"{base}/api/v2/logs/events/search", headers=headers,
                                  json={"filter": {"query": q, "from": self._utc_iso(from_ts),
                                                   "to": self._utc_iso(to_ts)}, "page": {"limit": 1000}},
                                  timeout=30)
            elif q_type == "events":
                resp = httpx.get(f"{base}/api/v1/events", headers=headers,
                                 params={"start": from_ts, "end": to_ts, "tags": q}, timeout=30)
            else:
                raise NotImplementedError(f"unknown type: {q_type}")

            if resp.status_code == 403:
                raise PermissionError("cred_expired")
            resp.raise_for_status()
            data = resp.json()
            self._cache_set(cache_key, data)
            return data, False
        except PermissionError:
            raise
        except httpx.HTTPStatusError as exc:
            raise RuntimeError(exc.response.text) from exc

    def refresh(self) -> None:
        self._creds.refresh()

    def health(self) -> dict:
        try:
            api_key, _ = self._creds.get()
            resp = httpx.get(f"https://api.{self._site}/api/v1/validate",
                             headers={"DD-API-KEY": api_key}, timeout=10)
            valid = resp.status_code == 200
        except Exception:
            valid = False
        return {"status": "ok" if valid else "degraded", "dd_api_valid": valid, "site": self._site}

    def permissions(self) -> dict:
        return {"readonly": self._readonly, "query_types": ["metrics", "logs", "events"]}

    def describe(self) -> dict:
        return {
            "description": "Query Datadog metrics, logs, and events",
            "actions": {
                "query": {
                    "params": {
                        "type": {"type": "string", "enum": ["metrics", "logs", "events"], "required": True},
                        "query": {"type": "string", "required": True,
                                  "examples": {"metrics": "avg:system.cpu.user{*}",
                                               "logs": "service:api status:error",
                                               "events": "env:prod"}},
                        "from_ts": {"type": "integer", "description": "Unix seconds", "required": True},
                        "to_ts": {"type": "integer", "description": "Unix seconds", "required": True},
                    }
                }
            },
        }
