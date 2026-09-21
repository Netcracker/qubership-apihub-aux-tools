#!/usr/bin/env bash
# Create the e2e-demo package hierarchy in APIHUB (idempotent).
set -euo pipefail

APIHUB_URL="${APIHUB_URL:-http://localhost:8090}"
TOKEN="${APIHUB_TOKEN:-8231f12d-054a-4a4b-be50-fda715694f3b}"

api() { curl -s -H "api-key: $TOKEN" -H "Content-Type: application/json" "$@"; }

ensure_package() {
  local id="$1" alias="$2" kind="$3" parent="$4" name="$5"
  local status
  status="$(api -o /dev/null -w '%{http_code}' "$APIHUB_URL/api/v2/packages/$id")"
  if [ "$status" = "200" ]; then
    echo "package $id exists"
    return
  fi
  api -X POST "$APIHUB_URL/api/v2/packages" \
    -d "{\"alias\":\"$alias\",\"parentId\":\"$parent\",\"kind\":\"$kind\",\"name\":\"$name\"}" >/dev/null
  echo "package $id created"
}

ensure_package "DEMO" "DEMO" "workspace" "" "Demo Workspace"
ensure_package "DEMO.RDM" "RDM" "package" "DEMO" "Reporting Data Mart"
