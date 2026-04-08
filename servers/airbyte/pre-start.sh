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
