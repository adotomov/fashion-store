#!/usr/bin/env bash
# Ensures the api Deployment in the devbox cluster has a "migrate" init
# container that runs goose against DATABASE_URL before the api container
# starts. devbox-cli's own Deployment template doesn't know about this
# container, so `devbox project up` never removes it once patched in — but
# if the namespace/deployment is ever deleted and recreated from scratch
# (devbox project rm, devbox reset, etc.), it needs to be reapplied.
#
# Usage:
#   scripts/devbox-patch-migrate.sh
#
# Run this after `devbox project up` has created the Deployment at least
# once, and after the api container's DATABASE_URL env var has been set
# (e.g. via `kubectl set env`). Safe to re-run; it overwrites any existing
# "migrate" init container with the current image/env.

set -euo pipefail

NAMESPACE="${DEVBOX_NAMESPACE:-api}"
DEPLOYMENT="${DEVBOX_DEPLOYMENT:-api}"
CONTAINER="${DEVBOX_CONTAINER:-api}"

if ! command -v jq >/dev/null 2>&1; then
	echo "error: jq is required" >&2
	exit 1
fi

deployment_json=$(kubectl get deployment -n "$NAMESPACE" "$DEPLOYMENT" -o json)

image=$(echo "$deployment_json" | jq -r --arg c "$CONTAINER" \
	'.spec.template.spec.containers[] | select(.name == $c) | .image')

if [ -z "$image" ] || [ "$image" = "null" ]; then
	echo "error: could not find container \"$CONTAINER\" on deployment \"$DEPLOYMENT\" in namespace \"$NAMESPACE\"" >&2
	exit 1
fi

db_url_env=$(echo "$deployment_json" | jq -c --arg c "$CONTAINER" \
	'.spec.template.spec.containers[] | select(.name == $c) | .env // [] | map(select(.name == "DATABASE_URL"))')

if [ "$(echo "$db_url_env" | jq 'length')" -eq 0 ]; then
	echo "error: container \"$CONTAINER\" has no DATABASE_URL env var set yet." >&2
	echo "Set it first, e.g.:" >&2
	echo "  kubectl set env deployment/$DEPLOYMENT -n $NAMESPACE DATABASE_URL=postgres://devbox:devbox@postgres-postgresql.shared.svc.cluster.local:5432/<db>?sslmode=disable" >&2
	exit 1
fi

patch=$(jq -n --arg image "$image" --argjson env "$db_url_env" '
  {
    spec: {
      template: {
        spec: {
          initContainers: [
            {
              name: "migrate",
              image: $image,
              command: ["/usr/local/bin/goose", "-dir", "/app/db/migrations", "postgres", "$(DATABASE_URL)", "up"],
              env: $env
            }
          ]
        }
      }
    }
  }
')

echo "$patch" | kubectl patch deployment -n "$NAMESPACE" "$DEPLOYMENT" --type merge --patch-file /dev/stdin

kubectl rollout status deployment/"$DEPLOYMENT" -n "$NAMESPACE" --timeout=120s
