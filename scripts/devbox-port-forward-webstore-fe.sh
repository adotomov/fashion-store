#!/usr/bin/env bash
# Forwards the devbox webstore-fe Service to http://localhost:3000.
#
# This exists only because Google's OAuth client console will not accept
# devbox's default *.localhost ingress hostnames (e.g.
# http://webstore-fe.api.localhost) as an Authorized JavaScript origin — it
# only accepts the literal host "localhost" (optionally with a port). The
# api service is unaffected and stays on its normal devbox ingress URL,
# since it's never opened directly in a browser.
#
# Usage:
#   scripts/devbox-port-forward-webstore-fe.sh [port]
#
# Runs `kubectl port-forward` detached in the background. Re-running this
# script restarts the forward (stopping any previous one first). Use
# scripts/devbox-port-forward-webstore-fe-stop.sh to stop it.

set -euo pipefail

NAMESPACE="${DEVBOX_NAMESPACE:-api}"
SERVICE="${DEVBOX_SERVICE:-webstore-fe}"
CONTAINER_PORT="${DEVBOX_CONTAINER_PORT:-3000}"
PORT="${1:-3000}"

PID_FILE="/tmp/fashion-store-devbox-port-forward-webstore-fe.pid"
LOG_FILE="/tmp/fashion-store-devbox-port-forward-webstore-fe.log"

if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
	echo "Stopping existing port-forward (pid $(cat "$PID_FILE"))"
	kill "$(cat "$PID_FILE")" 2>/dev/null || true
	rm -f "$PID_FILE"
fi

nohup kubectl port-forward -n "$NAMESPACE" "svc/$SERVICE" "$PORT:$CONTAINER_PORT" \
	>"$LOG_FILE" 2>&1 &
echo $! >"$PID_FILE"

sleep 1
if ! kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
	echo "error: port-forward failed to start, see $LOG_FILE" >&2
	cat "$LOG_FILE" >&2
	exit 1
fi

echo "Forwarding http://localhost:$PORT -> svc/$SERVICE:$CONTAINER_PORT in namespace $NAMESPACE (pid $(cat "$PID_FILE"))"
echo "Stop with: scripts/devbox-port-forward-webstore-fe-stop.sh"
