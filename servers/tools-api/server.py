"""tools-api — unified HTTP API gateway with pluggable adapters."""

import json
import os
import pathlib
import threading
import time
from contextlib import asynccontextmanager
from datetime import datetime, timezone
from typing import Any

import uvicorn
import yaml
from fastapi import FastAPI
from fastapi.responses import JSONResponse
from pydantic import BaseModel

from adapters.athena import AthenaAdapter
from adapters.datadog import DatadogAdapter
from adapters.jenkins import JenkinsAdapter
from adapters.pagerduty import PagerDutyAdapter
from adapters.sentry import SentryAdapter
from adapters.snowflake import SnowflakeAdapter

PORT = int(os.environ.get("TOOLS_API_PORT", "10893"))
_LOG_PATH = pathlib.Path(
    os.environ.get("TOOLS_API_LOG", os.path.expanduser("~/.local/log/tools-api.log"))
)

_ADAPTER_CLASSES = {
    "athena": AthenaAdapter,
    "datadog": DatadogAdapter,
    "jenkins": JenkinsAdapter,
    "pagerduty": PagerDutyAdapter,
    "sentry": SentryAdapter,
    "snowflake": SnowflakeAdapter,
}

_TOOLS_YAML = pathlib.Path(__file__).parent / "tools.yaml"
_STARTED_AT = datetime.now(timezone.utc).isoformat(timespec="seconds")

_adapters: dict[str, Any] = {}
_adapters_lock = threading.Lock()

_log_lock = threading.Lock()


def _log(event: str, **fields) -> None:
    record = {"ts": datetime.now(timezone.utc).isoformat(timespec="milliseconds"),
              "event": event, **fields}
    line = json.dumps(record, default=str)
    try:
        _LOG_PATH.parent.mkdir(parents=True, exist_ok=True)
        with _log_lock:
            with _LOG_PATH.open("a") as f:
                f.write(line + "\n")
    except OSError:
        pass


def _load_adapters() -> dict[str, Any]:
    with open(_TOOLS_YAML) as f:
        config = yaml.safe_load(f)

    result = {}
    for name, cfg in config.get("tools", {}).items():
        cls = _ADAPTER_CLASSES[cfg["adapter"]]
        params = {k: os.path.expandvars(str(v)) for k, v in cfg.get("params", {}).items()}
        result[name] = cls(
            readonly=cfg.get("readonly", True),
            readonly_exceptions=cfg.get("readonly_exceptions", []),
            **params,
        )
    return result


@asynccontextmanager
async def lifespan(app: FastAPI):
    with _adapters_lock:
        _adapters.update(_load_adapters())
    yield


app = FastAPI(title="Tool Gateway", lifespan=lifespan)


class InvokeRequest(BaseModel):
    tool: str
    action: str
    params: dict = {}


@app.post("/invoke")
def invoke(req: InvokeRequest) -> JSONResponse:
    with _adapters_lock:
        adapter = _adapters.get(req.tool)

    if adapter is None:
        _log("invoke", tool=req.tool, action=req.action, status=404, error="unknown_tool")
        return JSONResponse(status_code=404, content={
            "tool": req.tool, "action": req.action, "error": "unknown_tool",
        })

    t0 = time.monotonic()
    try:
        result, cached = adapter.query(req.action, req.params)
        ms = round((time.monotonic() - t0) * 1000)
        _log("invoke", tool=req.tool, action=req.action, status=200, cached=cached, duration_ms=ms)
        return JSONResponse(content={
            "tool": req.tool, "action": req.action, "cached": cached, "result": result,
        })
    except NotImplementedError as exc:
        ms = round((time.monotonic() - t0) * 1000)
        _log("invoke", tool=req.tool, action=req.action, status=400, error=str(exc), duration_ms=ms)
        return JSONResponse(status_code=400, content={
            "tool": req.tool, "action": req.action, "error": str(exc),
        })
    except ValueError as exc:
        ms = round((time.monotonic() - t0) * 1000)
        _log("invoke", tool=req.tool, action=req.action, status=400, error=str(exc), duration_ms=ms)
        return JSONResponse(status_code=400, content={
            "tool": req.tool, "action": req.action, "error": str(exc),
        })
    except PermissionError as exc:
        ms = round((time.monotonic() - t0) * 1000)
        _log("invoke", tool=req.tool, action=req.action, status=403, error=str(exc), duration_ms=ms)
        return JSONResponse(status_code=403, content={
            "tool": req.tool, "action": req.action,
            "error": str(exc), "refresh_url": f"/auth/{req.tool}/refresh",
        })
    except Exception as exc:
        ms = round((time.monotonic() - t0) * 1000)
        _log("invoke", tool=req.tool, action=req.action, status=500, error=str(exc), duration_ms=ms)
        return JSONResponse(status_code=500, content={
            "tool": req.tool, "action": req.action, "error": str(exc),
        })


@app.post("/auth/{tool}/refresh")
def auth_refresh(tool: str) -> JSONResponse:
    with _adapters_lock:
        adapter = _adapters.get(tool)

    if adapter is None:
        _log("refresh", tool=tool, status=404, error="unknown_tool")
        return JSONResponse(status_code=404, content={"error": "unknown_tool", "tool": tool})

    t0 = time.monotonic()
    try:
        adapter.refresh()
        ms = round((time.monotonic() - t0) * 1000)
        _log("refresh", tool=tool, status=200, duration_ms=ms)
        return JSONResponse(content={"status": "ok", "tool": tool})
    except Exception as exc:
        ms = round((time.monotonic() - t0) * 1000)
        _log("refresh", tool=tool, status=500, error=str(exc), duration_ms=ms)
        return JSONResponse(status_code=500, content={
            "status": "failed", "tool": tool, "message": str(exc),
        })


@app.get("/health")
def health() -> JSONResponse:
    with _adapters_lock:
        snapshot = dict(_adapters)

    tools = {name: adapter.health() for name, adapter in snapshot.items()}
    overall = "ok" if all(t.get("status") in ("ok", "uninitialized") for t in tools.values()) else "degraded"
    return JSONResponse(content={"status": overall, "started_at": _STARTED_AT, "tools": tools})


@app.get("/permissions")
def permissions() -> JSONResponse:
    with _adapters_lock:
        snapshot = dict(_adapters)

    return JSONResponse(content={
        name: adapter.permissions() for name, adapter in snapshot.items()
    })


@app.get("/describe")
def describe() -> JSONResponse:
    """Return tool schemas and current status — entry point for agents discovering available tools."""
    with _adapters_lock:
        snapshot = dict(_adapters)

    return JSONResponse(content={
        "tools": {
            name: {**adapter.describe(), "status": adapter.health().get("status")}
            for name, adapter in snapshot.items()
        }
    })


class ModeRequest(BaseModel):
    readonly: bool


@app.post("/mode")
def set_mode_all(req: ModeRequest) -> JSONResponse:
    """Set readonly mode on all adapters at once."""
    with _adapters_lock:
        snapshot = dict(_adapters)

    for adapter in snapshot.values():
        adapter._readonly = req.readonly
    mode = "readonly" if req.readonly else "readwrite"
    _log("mode", tool="*", readonly=req.readonly)
    return JSONResponse(content={"mode": mode, "tools": list(snapshot)})


@app.post("/mode/{tool}")
def set_mode(tool: str, req: ModeRequest) -> JSONResponse:
    """Set readonly mode on a live adapter — resets to tools.yaml default on restart."""
    with _adapters_lock:
        adapter = _adapters.get(tool)

    if adapter is None:
        return JSONResponse(status_code=404, content={"error": "unknown_tool", "tool": tool})

    adapter._readonly = req.readonly
    mode = "readonly" if req.readonly else "readwrite"
    _log("mode", tool=tool, readonly=req.readonly)
    return JSONResponse(content={"tool": tool, "mode": mode})


@app.post("/reload")
def reload() -> JSONResponse:
    """Hot-reload tools.yaml — adds new tools without dropping existing ones."""
    try:
        new = _load_adapters()
        with _adapters_lock:
            added = [name for name in new if name not in _adapters]
            for name in added:
                _adapters[name] = new[name]
        return JSONResponse(content={"status": "ok", "added": added})
    except Exception as exc:
        return JSONResponse(status_code=500, content={"status": "failed", "message": str(exc)})


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=PORT)
