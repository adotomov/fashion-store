#!/usr/bin/env bash
# Applies apps/api/db/seeds/dev_catalog.sql to the devbox database.
#
# Reads DATABASE_URL off the api Deployment (the same value the migrate init
# container uses) and pipes the seed through psql inside the shared postgres
# pod, so no local psql client or port-forward is needed.
#
# Usage:
#   scripts/devbox-seed-catalog.sh
#
# Safe to re-run — the seed skips rows whose slug already exists.

set -euo pipefail

NAMESPACE="${DEVBOX_NAMESPACE:-api}"
DEPLOYMENT="${DEVBOX_DEPLOYMENT:-api}"
CONTAINER="${DEVBOX_CONTAINER:-api}"
PG_NAMESPACE="${DEVBOX_PG_NAMESPACE:-shared}"
PG_POD="${DEVBOX_PG_POD:-postgres-postgresql-0}"

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
seed_file="$repo_root/apps/api/db/seeds/dev_catalog.sql"

if [ ! -f "$seed_file" ]; then
	echo "error: seed file not found at $seed_file" >&2
	exit 1
fi

db_url=$(kubectl get deployment -n "$NAMESPACE" "$DEPLOYMENT" \
	-o jsonpath="{.spec.template.spec.containers[?(@.name==\"$CONTAINER\")].env[?(@.name==\"DATABASE_URL\")].value}")

if [ -z "$db_url" ]; then
	echo "error: container \"$CONTAINER\" on deployment \"$DEPLOYMENT\" has no DATABASE_URL env var" >&2
	exit 1
fi

echo "Seeding dev catalog into $DEPLOYMENT's database via $PG_NAMESPACE/$PG_POD…"

kubectl exec -i -n "$PG_NAMESPACE" "$PG_POD" -- \
	env PGURL="$db_url" sh -c 'psql "$PGURL" -v ON_ERROR_STOP=1 -q' <"$seed_file"

echo "Done."
