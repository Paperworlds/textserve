import os
import subprocess
import threading
import time
from typing import Any

import httpx

_CACHE_TTL = 10.0
_BASE_URL = "https://api.pagerduty.com"


class CredStore:
    def __init__(self, token: str, token_source: str, ttl: float):
        self._token = token
        self._token_source = token_source
        self._ttl = ttl
        self._loaded_at = time.monotonic()
        self._lock = threading.Lock()

    def _expired(self) -> bool:
        return time.monotonic() - self._loaded_at > self._ttl

    def _reload(self) -> None:
        self._token = subprocess.check_output(
            ["op", "read", self._token_source], text=True
        ).strip()
        self._loaded_at = time.monotonic()

    def get(self) -> str:
        with self._lock:
            if self._expired():
                self._reload()
            return self._token

    def refresh(self) -> None:
        with self._lock:
            self._reload()


class PagerDutyAdapter:
    def __init__(self, readonly: bool = True, readonly_exceptions: list = [],
                 cred_ttl_hours: str = "4"):
        self._creds = CredStore(
            token=os.environ["PAGERDUTY_TOKEN"],
            token_source=os.environ["PAGERDUTY_TOKEN_SOURCE"],
            ttl=float(cred_ttl_hours) * 3600,
        )
        self._readonly = readonly
        self._cache: dict[tuple, tuple[float, Any]] = {}
        self._cache_lock = threading.Lock()

    def _headers(self) -> dict[str, str]:
        try:
            token = self._creds.get()
        except subprocess.CalledProcessError as exc:
            raise PermissionError("cred_refresh_failed") from exc
        return {
            "Authorization": f"Token token={token}",
            "Accept": "application/vnd.pagerduty+json;version=2",
        }

    def _cache_get(self, key: tuple) -> Any | None:
        with self._cache_lock:
            entry = self._cache.get(key)
            if entry and time.monotonic() - entry[0] < _CACHE_TTL:
                return entry[1]
            return None

    def _cache_set(self, key: tuple, data: Any) -> None:
        with self._cache_lock:
            self._cache[key] = (time.monotonic(), data)

    def _get(self, path: str, params: dict | None = None) -> Any:
        resp = httpx.get(f"{_BASE_URL}{path}", headers=self._headers(),
                         params=params, timeout=30)
        if resp.status_code in (401, 403):
            raise PermissionError("auth_expired")
        resp.raise_for_status()
        return resp.json()

    def query(self, action: str, params: dict) -> tuple[Any, bool]:
        if action == "list_incidents":
            status = params.get("status", "triggered,acknowledged")
            urgency = params.get("urgency", "")
            service_ids = params.get("service_ids", [])
            limit = int(params.get("limit", 25))
            since = params.get("since", "")
            until = params.get("until", "")
            cache_key = ("list_incidents", status, urgency, tuple(service_ids), limit, since, until)
            if hit := self._cache_get(cache_key):
                return hit, True
            api_params: dict[str, Any] = {"limit": limit}
            if status:
                api_params["statuses[]"] = status.split(",")
            if urgency:
                api_params["urgencies[]"] = [urgency]
            if service_ids:
                api_params["service_ids[]"] = service_ids
            if since:
                api_params["since"] = since
            if until:
                api_params["until"] = until
            data = self._get("/incidents", params=api_params)
            result = {
                "incidents": [
                    {
                        "id": i.get("id"),
                        "incident_number": i.get("incident_number"),
                        "title": i.get("title"),
                        "status": i.get("status"),
                        "urgency": i.get("urgency"),
                        "created_at": i.get("created_at"),
                        "resolved_at": i.get("resolved_at"),
                        "service": i.get("service", {}).get("summary"),
                        "html_url": i.get("html_url"),
                        "assigned_to": [a.get("assignee", {}).get("summary")
                                        for a in i.get("assignments", [])],
                    }
                    for i in data.get("incidents", [])
                ]
            }
            self._cache_set(cache_key, result)
            return result, False

        elif action == "get_incident":
            incident_id = params.get("incident_id", "")
            if not incident_id:
                raise ValueError("incident_id is required")
            cache_key = ("get_incident", str(incident_id))
            if hit := self._cache_get(cache_key):
                return hit, True
            data = self._get(f"/incidents/{incident_id}")
            i = data.get("incident", {})
            result = {
                "id": i.get("id"),
                "incident_number": i.get("incident_number"),
                "title": i.get("title"),
                "status": i.get("status"),
                "urgency": i.get("urgency"),
                "created_at": i.get("created_at"),
                "resolved_at": i.get("resolved_at"),
                "service": i.get("service", {}).get("summary"),
                "html_url": i.get("html_url"),
                "assigned_to": [a.get("assignee", {}).get("summary")
                                 for a in i.get("assignments", [])],
                "body": i.get("body", {}).get("details", ""),
                "last_status_change_at": i.get("last_status_change_at"),
                "escalation_policy": i.get("escalation_policy", {}).get("summary"),
            }
            self._cache_set(cache_key, result)
            return result, False

        elif action == "list_alerts":
            incident_id = params.get("incident_id", "")
            if not incident_id:
                raise ValueError("incident_id is required")
            limit = int(params.get("limit", 25))
            cache_key = ("list_alerts", str(incident_id), limit)
            if hit := self._cache_get(cache_key):
                return hit, True
            data = self._get(f"/incidents/{incident_id}/alerts", params={"limit": limit})
            result = {
                "alerts": [
                    {
                        "id": a.get("id"),
                        "status": a.get("status"),
                        "severity": a.get("severity"),
                        "created_at": a.get("created_at"),
                        "resolved_at": a.get("resolved_at"),
                        "summary": a.get("summary"),
                        "body": a.get("body", {}).get("details", ""),
                    }
                    for a in data.get("alerts", [])
                ]
            }
            self._cache_set(cache_key, result)
            return result, False

        elif action == "list_services":
            limit = int(params.get("limit", 50))
            cache_key = ("list_services", limit)
            if hit := self._cache_get(cache_key):
                return hit, True
            data = self._get("/services", params={"limit": limit})
            result = {
                "services": [
                    {
                        "id": s.get("id"),
                        "name": s.get("name"),
                        "status": s.get("status"),
                        "description": s.get("description", ""),
                        "html_url": s.get("html_url"),
                    }
                    for s in data.get("services", [])
                ]
            }
            self._cache_set(cache_key, result)
            return result, False

        else:
            raise NotImplementedError(f"unknown action: {action}")

    def refresh(self) -> None:
        try:
            self._creds.refresh()
        except subprocess.CalledProcessError as exc:
            raise RuntimeError("op read failed — check 1Password access") from exc

    def health(self) -> dict:
        try:
            self._get("/abilities")
            return {"status": "ok"}
        except PermissionError:
            return {"status": "auth_error",
                    "requires": "POST /auth/pagerduty/refresh"}
        except Exception as exc:
            return {"status": "degraded", "error": str(exc)}

    def permissions(self) -> dict:
        return {
            "readonly": self._readonly,
            "actions": ["list_incidents", "get_incident", "list_alerts", "list_services"],
        }

    def describe(self) -> dict:
        return {
            "description": "Query PagerDuty — list and inspect incidents, alerts, and services",
            "auth": "Token auth (pre-loaded; refresh via POST /auth/pagerduty/refresh if expired)",
            "actions": {
                "list_incidents": {
                    "params": {
                        "status": {
                            "type": "string",
                            "default": "triggered,acknowledged",
                            "example": "triggered",
                            "note": "comma-separated: triggered, acknowledged, resolved",
                        },
                        "urgency": {
                            "type": "string",
                            "example": "high",
                            "note": "high or low; omit for all",
                        },
                        "service_ids": {
                            "type": "array",
                            "example": ["P1234AB"],
                            "note": "filter by service IDs; omit for all services",
                        },
                        "since": {
                            "type": "string",
                            "example": "2026-05-20T00:00:00Z",
                            "note": "ISO 8601 start time",
                        },
                        "until": {
                            "type": "string",
                            "example": "2026-05-21T00:00:00Z",
                            "note": "ISO 8601 end time",
                        },
                        "limit": {"type": "integer", "default": 25, "example": 10},
                    }
                },
                "get_incident": {
                    "params": {
                        "incident_id": {
                            "type": "string",
                            "required": True,
                            "example": "P1234AB",
                            "note": "PagerDuty incident ID",
                        },
                    }
                },
                "list_alerts": {
                    "params": {
                        "incident_id": {"type": "string", "required": True, "example": "P1234AB"},
                        "limit": {"type": "integer", "default": 25},
                    }
                },
                "list_services": {
                    "params": {
                        "limit": {"type": "integer", "default": 50},
                    },
                    "note": "returns all services with ID, name, and current status",
                },
            },
        }
