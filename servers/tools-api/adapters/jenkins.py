import base64
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


class JenkinsAdapter:
    def __init__(self, readonly: bool = True, readonly_exceptions: list = [],
                 cred_ttl_hours: str = "4"):
        self._url = os.environ["JENKINS_URL"].rstrip("/")
        self._user = os.environ["JENKINS_USER"]
        self._creds = CredStore(
            token=os.environ["JENKINS_TOKEN"],
            token_source=os.environ["JENKINS_TOKEN_SOURCE"],
            ttl=float(cred_ttl_hours) * 3600,
        )
        self._readonly = readonly
        self._cache: dict[tuple, tuple[float, Any]] = {}
        self._cache_lock = threading.Lock()

    def _auth_header(self) -> str:
        try:
            token = self._creds.get()
        except subprocess.CalledProcessError as exc:
            raise PermissionError("cred_refresh_failed") from exc
        creds = base64.b64encode(f"{self._user}:{token}".encode()).decode()
        return f"Basic {creds}"

    def _headers(self) -> dict[str, str]:
        return {"Authorization": self._auth_header()}

    def _cache_get(self, key: tuple) -> Any | None:
        with self._cache_lock:
            entry = self._cache.get(key)
            if entry and time.monotonic() - entry[0] < _CACHE_TTL:
                return entry[1]
            return None

    def _cache_set(self, key: tuple, data: Any) -> None:
        with self._cache_lock:
            self._cache[key] = (time.monotonic(), data)

    @staticmethod
    def _job_path(job: str) -> str:
        return "/job/" + "/job/".join(job.strip("/").split("/"))

    def _get(self, path: str, params: dict | None = None) -> dict:
        resp = httpx.get(f"{self._url}{path}", headers=self._headers(), params=params, timeout=30)
        if resp.status_code in (401, 403):
            raise PermissionError("auth_expired")
        resp.raise_for_status()
        return resp.json()

    def _get_text(self, path: str) -> str:
        resp = httpx.get(f"{self._url}{path}", headers=self._headers(), timeout=30)
        if resp.status_code in (401, 403):
            raise PermissionError("auth_expired")
        resp.raise_for_status()
        return resp.text

    def query(self, action: str, params: dict) -> tuple[Any, bool]:
        if action == "list_jobs":
            folder = params.get("folder", "").strip("/")
            cache_key = ("list_jobs", folder)
            if hit := self._cache_get(cache_key):
                return hit, True
            path = (self._job_path(folder) if folder else "") + "/api/json"
            data = self._get(path, params={
                "tree": "jobs[name,url,color,lastBuild[number,result,timestamp,duration]]"
            })
            result = {
                "jobs": [
                    {
                        "name": j.get("name"),
                        "url": j.get("url"),
                        "status": j.get("color", "unknown"),
                        "last_build": j.get("lastBuild"),
                    }
                    for j in data.get("jobs", [])
                ]
            }
            self._cache_set(cache_key, result)
            return result, False

        elif action == "get_job":
            job = params.get("job", "")
            if not job:
                raise ValueError("job is required")
            cache_key = ("get_job", job)
            if hit := self._cache_get(cache_key):
                return hit, True
            data = self._get(
                self._job_path(job) + "/api/json",
                params={"tree": "name,url,color,description,builds[number,result,timestamp,duration]{0,10}"},
            )
            self._cache_set(cache_key, data)
            return data, False

        elif action == "get_build":
            job = params.get("job", "")
            build = params.get("build", "lastBuild")
            if not job:
                raise ValueError("job is required")
            cache_key = ("get_build", job, str(build))
            if hit := self._cache_get(cache_key):
                return hit, True
            data = self._get(self._job_path(job) + f"/{build}/api/json")
            self._cache_set(cache_key, data)
            return data, False

        elif action == "get_log":
            job = params.get("job", "")
            build = params.get("build", "lastBuild")
            if not job:
                raise ValueError("job is required")
            cache_key = ("get_log", job, str(build))
            if hit := self._cache_get(cache_key):
                return hit, True
            text = self._get_text(self._job_path(job) + f"/{build}/consoleText")
            max_chars = int(params.get("max_chars", 10000))
            truncated = len(text) > max_chars
            if truncated:
                text = text[-max_chars:]
            result = {"log": text, "truncated": truncated}
            self._cache_set(cache_key, result)
            return result, False

        elif action == "trigger":
            if self._readonly:
                raise PermissionError("readonly — set readonly: false in tools.yaml to allow triggers")
            job = params.get("job", "")
            if not job:
                raise ValueError("job is required")
            build_params = params.get("build_params", {})
            if build_params:
                endpoint = self._job_path(job) + "/buildWithParameters"
                resp = httpx.post(
                    f"{self._url}{endpoint}",
                    headers=self._headers(),
                    data={k: str(v) for k, v in build_params.items()},
                    timeout=30,
                )
            else:
                resp = httpx.post(
                    f"{self._url}{self._job_path(job)}/build",
                    headers=self._headers(), timeout=30,
                )
            if resp.status_code in (401, 403):
                raise PermissionError("auth_expired")
            resp.raise_for_status()
            return {"triggered": True, "job": job, "queue_url": resp.headers.get("Location", "")}, False

        else:
            raise NotImplementedError(f"unknown action: {action}")

    def refresh(self) -> None:
        try:
            self._creds.refresh()
        except subprocess.CalledProcessError as exc:
            raise RuntimeError("op read failed — check 1Password access") from exc

    def health(self) -> dict:
        try:
            self._get("/api/json", params={"tree": "mode"})
            return {"status": "ok", "url": self._url}
        except PermissionError:
            return {"status": "auth_error", "url": self._url,
                    "requires": "POST /auth/jenkins/refresh"}
        except Exception as exc:
            return {"status": "degraded", "error": str(exc), "url": self._url}

    def permissions(self) -> dict:
        actions = ["list_jobs", "get_job", "get_build", "get_log"]
        if not self._readonly:
            actions.append("trigger")
        return {"readonly": self._readonly, "actions": actions}

    def describe(self) -> dict:
        return {
            "description": "Query Jenkins CI — list jobs, check build status, read logs",
            "auth": "static API token (pre-loaded; refresh via POST /auth/jenkins/refresh if expired)",
            "actions": {
                "list_jobs": {
                    "params": {
                        "folder": {
                            "type": "string",
                            "example": "paradex",
                            "note": "optional folder name; omit for top-level jobs",
                        },
                    }
                },
                "get_job": {
                    "params": {
                        "job": {
                            "type": "string",
                            "required": True,
                            "example": "paradex/my-pipeline",
                            "note": "job path — use / for nested jobs",
                        },
                    }
                },
                "get_build": {
                    "params": {
                        "job": {"type": "string", "required": True, "example": "paradex/my-pipeline"},
                        "build": {
                            "type": "string|integer",
                            "default": "lastBuild",
                            "example": 42,
                            "note": "build number or lastBuild / lastSuccessfulBuild / lastFailedBuild",
                        },
                    }
                },
                "get_log": {
                    "params": {
                        "job": {"type": "string", "required": True, "example": "paradex/my-pipeline"},
                        "build": {"type": "string|integer", "default": "lastBuild", "example": 42},
                        "max_chars": {
                            "type": "integer",
                            "default": 10000,
                            "note": "tail chars of console log returned",
                        },
                    }
                },
                "trigger": {
                    "params": {
                        "job": {"type": "string", "required": True, "example": "backend/mono/Meta job all backends from branch to env"},
                        "build_params": {
                            "type": "object",
                            "example": {"BRANCH": "main", "ENV": "staging", "DEPLOY_PROXIES": "true"},
                            "note": "required for parameterized jobs; uses buildWithParameters endpoint",
                        },
                    },
                    "note": "requires readonly: false in tools.yaml",
                },
            },
        }
