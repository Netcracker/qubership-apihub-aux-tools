#!/usr/bin/env bash
# Start the e2e-demo APIHUB backend (a second, isolated instance on :8095, so it
# does not collide with any backend you already run locally on :8090).
#
# Usage: BACKEND_REPO=/path/to/qubership-apihub-backend_3 bash start-backend.sh
set -euo pipefail

BACKEND_REPO="${BACKEND_REPO:-}"
[ -n "$BACKEND_REPO" ] || { echo "FATAL: set BACKEND_REPO=/path/to/qubership-apihub-backend_3" >&2; exit 1; }
[ -f "$BACKEND_REPO/qubership-apihub-service/go.mod" ] || { echo "FATAL: $BACKEND_REPO does not look like the backend repo" >&2; exit 1; }

DIR="$(cd "$(dirname "$0")" && pwd)"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-apihub_e2e}"

# ---- JWT key (generated once, gitignored) ----------------------------------
if [ ! -f "$DIR/jwt_private_key" ]; then
  openssl genpkey -out "$DIR/rsakey.pem" -algorithm RSA -pkeyopt rsa_keygen_bits:2048
  base64 "$DIR/rsakey.pem" | tr -d '\n' > "$DIR/jwt_private_key"
  rm -f "$DIR/rsakey.pem"
  echo "generated $DIR/jwt_private_key"
fi

# ---- database ----------------------------------------------------------------
# Requires a running postgres reachable at $DB_HOST:$DB_PORT with a
# superuser role named "postgres" and role "apihub"/password "apihub" already
# present (see docs/local_development/docker-compose/DB in the backend repo).
if ! docker exec postgres psql -U postgres -tc "select 1 from pg_database where datname='$DB_NAME'" 2>/dev/null | grep -q 1; then
  docker exec postgres psql -U postgres -c "create database $DB_NAME owner apihub"
  echo "created database $DB_NAME"
fi

# ---- config.yaml (gitignored, holds the JWT key) ----------------------------
mkdir -p "$DIR/ephemeral"
KEY="$(cat "$DIR/jwt_private_key")"
sed -e "s#{{JWT_KEY}}#$KEY#" -e "s#{{BACKEND_REPO}}#$BACKEND_REPO#" \
  "$DIR/config.template.yaml" > "$DIR/config.yaml"

# ---- run ---------------------------------------------------------------------
cd "$BACKEND_REPO/qubership-apihub-service"
[ -f go.sum ] || go mod tidy   # go.sum is gitignored in the backend repo
echo "starting backend on :8095 (Ctrl-C to stop)…"
APIHUB_CONFIG_FOLDER="$DIR" exec go run .
