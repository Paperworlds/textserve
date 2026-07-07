import os
import subprocess
import threading
import time
from typing import Any

import httpx

_CACHE_TTL = 10.0


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


class SentryAdapter:
    def __init__(self, readonly: bool = True, readonly_exceptions: list = [],
                 cred_ttl_hours: str = "4"):
        self._url = os.environ.get("SENTRY_URL", "https://sentry.io").rstrip("/")
        self._org = os.environ["SENTRY_ORG"]
        self._creds = CredStore(
            token=os.environ["SENTRY_TOKEN"],
            token_source=os.environ["SENTRY_TOKEN_SOURCE"],
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
        return {"Authorization": f"Bearer {token}"}

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
        resp = httpx.get(f"{self._url}{path}", headers=self._headers(),
                         params=params, timeout=30)
        if resp.status_code in (401, 403):
            raise PermissionError("auth_expired")
        resp.raise_for_status()
        return resp.json()

    def _resolve_project(self, slug_or_id: str) -> str:
        """Return numeric project ID, resolving slug if needed."""
        if slug_or_id.isdigit():
            return slug_or_id
        cache_key = ("list_projects",)
        projects = self._cache_get(cache_key)
        if projects is None:
            data = self._get(f"/api/0/organizations/{self._org}/projects/")
            projects = {p["slug"]: p["id"] for p in data}
            self._cache_set(cache_key, projects)
        pid = projects.get(slug_or_id)
        if pid is None:
            raise ValueError(f"unknown project {slug_or_id!r} — use list_projects to see valid slugs")
        return str(pid)

    @staticmethod
    def _slim_issue(i: dict) -> dict:
        return {
            "id": i.get("id"),
            "shortId": i.get("shortId"),
            "title": i.get("title"),
            "level": i.get("level"),
            "status": i.get("status"),
            "substatus": i.get("substatus"),
            "priority": i.get("priority"),
            "project": i.get("project", {}).get("slug"),
            "firstSeen": i.get("firstSeen"),
            "lastSeen": i.get("lastSeen"),
            "count": i.get("count"),
            "userCount": i.get("userCount"),
            "culprit": i.get("culprit"),
            "permalink": i.get("permalink"),
            "assignedTo": i.get("assignedTo"),
            "isUnhandled": i.get("isUnhandled"),
        }

    def query(self, action: str, params: dict) -> tuple[Any, bool]:
        if action == "list_issues":
            project = params.get("project", "")
            query = params.get("query", "is:unresolved")
            limit = int(params.get("limit", 25))
            cache_key = ("list_issues", project, query, limit)
            if hit := self._cache_get(cache_key):
                return hit, True
            api_params: dict[str, Any] = {"query": query, "limit": limit}
            if project:
                api_params["project"] = self._resolve_project(project)
            data = self._get(f"/api/0/organizations/{self._org}/issues/", params=api_params)
            result = {"issues": [self._slim_issue(i) for i in data]}
            self._cache_set(cache_key, result)
            return result, False

        elif action == "get_issue":
            issue_id = params.get("issue_id", "")
            if not issue_id:
                raise ValueError("issue_id is required")
            cache_key = ("get_issue", str(issue_id))
            if hit := self._cache_get(cache_key):
                return hit, True
            data = self._get(f"/api/0/issues/{issue_id}/")
            result = self._slim_issue(data)
            self._cache_set(cache_key, result)
            return result, False

        elif action == "get_issue_events":
            issue_id = params.get("issue_id", "")
            if not issue_id:
                raise ValueError("issue_id is required")
            limit = int(params.get("limit", 10))
            cache_key = ("get_issue_events", str(issue_id), limit)
            if hit := self._cache_get(cache_key):
                return hit, True
            data = self._get(f"/api/0/issues/{issue_id}/events/", params={"full": "true"})
            data = data[:limit]
            events = []
            for e in data:
                entry = {
                    "id": e.get("id"),
                    "dateCreated": e.get("dateCreated"),
                    "message": e.get("message"),
                    "tags": {t["key"]: t["value"] for t in e.get("tags", [])},
                }
                # include exception frames if present
                exc_values = (
                    e.get("entries", [{}])[0].get("data", {}).get("values", [])
                    if e.get("entries") else []
                )
                if exc_values:
                    entry["exception"] = [
                        {
                            "type": v.get("type"),
                            "value": v.get("value"),
                            "frames": [
                                {"filename": f.get("filename"), "lineno": f.get("lineNo"),
                                 "function": f.get("function")}
                                for f in (v.get("stacktrace") or {}).get("frames", [])[-5:]
                            ],
                        }
                        for v in exc_values
                    ]
                events.append(entry)
            result = {"events": events}
            self._cache_set(cache_key, result)
            return result, False

        elif action == "list_projects":
            cache_key = ("list_projects_slim",)
            if hit := self._cache_get(cache_key):
                return hit, True
            data = self._get(f"/api/0/organizations/{self._org}/projects/")
            # warm the slug→id cache used by _resolve_project
            self._cache_set(("list_projects",), {p["slug"]: p["id"] for p in data})
            result = {
                "projects": [
                    {"slug": p["slug"], "name": p["name"], "platform": p.get("platform")}
                    for p in data
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
            self._get(f"/api/0/organizations/{self._org}/")
            return {"status": "ok", "org": self._org}
        except PermissionError:
            return {"status": "auth_error", "org": self._org,
                    "requires": "POST /auth/sentry/refresh"}
        except Exception as exc:
            return {"status": "degraded", "error": str(exc), "org": self._org}

    def permissions(self) -> dict:
        return {
            "readonly": self._readonly,
            "actions": ["list_issues", "get_issue", "get_issue_events", "list_projects"],
        }

    def describe(self) -> dict:
        return {
            "description": "Query Sentry — list and inspect issues and error events",
            "auth": "Bearer token (pre-loaded; refresh via POST /auth/sentry/refresh if expired)",
            "actions": {
                "list_issues": {
                    "params": {
                        "query": {
                            "type": "string",
                            "default": "is:unresolved",
                            "example": "is:unresolved level:error",
                            "note": "Sentry search syntax; see https://docs.sentry.io/product/sentry-basics/search/",
                        },
                        "project": {
                            "type": "string",
                            "example": "paradex",
                            "note": "project slug; omit to search all projects in the org",
                        },
                        "limit": {"type": "integer", "default": 25, "example": 10},
                    }
                },
                "get_issue": {
                    "params": {
                        "issue_id": {
                            "type": "string",
                            "required": True,
                            "example": "7407094074",
                            "note": "numeric issue ID or short ID like PARADEX-7BR",
                        },
                    }
                },
                "get_issue_events": {
                    "params": {
                        "issue_id": {"type": "string", "required": True, "example": "7407094074"},
                        "limit": {
                            "type": "integer",
                            "default": 10,
                            "note": "number of recent events to return; each includes tags and stack frames",
                        },
                    }
                },
                "list_projects": {
                    "params": {},
                    "note": "returns all projects in the org with slug, name, and platform",
                },
            },
        }
