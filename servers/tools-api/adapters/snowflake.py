import os
import re
import threading
from typing import Any

import snowflake.connector

_ALLOWED_STMT = {"select", "describe", "desc", "show", "use", "with"}
_AUTH_ERROR_CODES = {390100, 390114, 390195, 250001}


class SnowflakeAdapter:
    def __init__(self, readonly: bool = True, readonly_exceptions: list = []):
        self._account = os.environ["SNOWFLAKE_ACCOUNT"]
        self._user = os.environ["SNOWFLAKE_USER"]
        self._readonly = readonly
        self._exception_patterns = [
            re.compile(r"\b" + p.replace("*", r"\w*") + r"\b", re.IGNORECASE)
            for p in readonly_exceptions
        ]
        self._conn: snowflake.connector.SnowflakeConnection | None = None
        self._conn_lock = threading.Lock()
        self._initialized = False
        self._auth_expired = False

    def _connect(self) -> snowflake.connector.SnowflakeConnection:
        return snowflake.connector.connect(
            account=self._account,
            user=self._user,
            authenticator="externalbrowser",
        )

    def _get_conn(self) -> snowflake.connector.SnowflakeConnection:
        with self._conn_lock:
            if self._conn is None:
                self._conn = self._connect()
            return self._conn

    def _drop_conn(self) -> None:
        with self._conn_lock:
            if self._conn is not None:
                try:
                    self._conn.close()
                except Exception:
                    pass
                self._conn = None
            self._auth_expired = True

    def _is_auth_error(self, exc: Exception) -> bool:
        code = getattr(exc, "errno", None)
        if code in _AUTH_ERROR_CODES:
            return True
        msg = str(exc).lower()
        return any(k in msg for k in ("token expired", "session no longer exists", "authentication"))

    def _is_allowed(self, sql: str) -> bool:
        if not self._readonly:
            return True
        if any(p.search(sql) for p in self._exception_patterns):
            return True
        first = sql.strip().lstrip("(").split()[0].lower()
        return first in _ALLOWED_STMT

    def query(self, action: str, params: dict) -> tuple[Any, bool]:
        if action != "query":
            raise NotImplementedError(f"unknown action: {action}")
        if not self._initialized:
            raise PermissionError("not_initialized")
        if self._auth_expired:
            raise PermissionError("auth_expired")

        sql = params.get("sql", "")
        if not self._is_allowed(sql):
            raise ValueError("read_only_violation")

        try:
            conn = self._get_conn()
            cur = conn.cursor()
            for key, stmt in [("warehouse", "USE WAREHOUSE {}"), ("role", "USE ROLE {}"),
                               ("database", "USE DATABASE {}"), ("schema_name", "USE SCHEMA {}")]:
                if val := params.get(key):
                    cur.execute(stmt.format(val))
            cur.execute(sql)
            rows = cur.fetchall()
            columns = [d[0] for d in cur.description] if cur.description else []
            return {"columns": columns, "rows": [list(r) for r in rows],
                    "row_count": len(rows), "query_id": cur.sfqid}, False
        except Exception as exc:
            if self._is_auth_error(exc):
                self._drop_conn()
                raise PermissionError("auth_expired") from exc
            raise

    def refresh(self) -> None:
        self._drop_conn()
        self._auth_expired = False
        self._get_conn()
        self._initialized = True

    def health(self) -> dict:
        if not self._initialized:
            return {"status": "uninitialized", "requires": "POST /auth/snowflake/refresh"}
        if self._auth_expired:
            return {"status": "degraded", "auth_expired": True}
        connected = False
        if self._conn is not None:
            try:
                self._conn.cursor().execute("SELECT 1")
                connected = True
            except Exception:
                pass
        return {"status": "ok" if connected else "degraded", "snowflake_connected": connected}

    def permissions(self) -> dict:
        return {
            "readonly": self._readonly,
            "allowed_statements": sorted(_ALLOWED_STMT),
            "full_access_patterns": [p.pattern for p in self._exception_patterns],
            "initialized": self._initialized,
        }

    def describe(self) -> dict:
        return {
            "description": "Query Snowflake — requires POST /auth/snowflake/refresh before first use",
            "actions": {
                "query": {
                    "params": {
                        "sql": {"type": "string", "required": True,
                                "example": "SELECT * FROM prod.public.trades LIMIT 100"},
                        "warehouse": {"type": "string"},
                        "role": {"type": "string"},
                        "database": {"type": "string"},
                        "schema_name": {"type": "string"},
                    }
                }
            },
        }
