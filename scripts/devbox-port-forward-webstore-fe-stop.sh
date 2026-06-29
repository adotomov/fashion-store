#!/usr/bin/env bash
# Stops the background kubectl port-forward started by
# scripts/devbox-port-forward-webstore-fe.sh.

set -euo pipefail

PID_FILE="/tmp/fashion-store-devbox-port-forward-webstore-fe.pid"

if [ ! -f "$PID_FILE" ]; then
	echo "No port-forward pid file found at $PID_FILE"
	exit 0
fi

PID="$(cat "$PID_FILE")"
if kill -0 "$PID" 2>/dev/null; then
	kill "$PID"
	echo "Stopped port-forward (pid $PID)"
else
	echo "Port-forward (pid $PID) was not running"
fi

rm -f "$PID_FILE"
