#!/usr/bin/env bash
# Push a "next version" branch to the e2e-demo DDL and comments repositories,
# simulating a schema change ready to publish. Idempotent: re-running
# force-pushes the current seed-v2/ content to the same branch.
#
# Usage: bash seed-branch.sh [branch-name]   (default: 2026.5)
set -euo pipefail

GITLAB_URL="${GITLAB_URL:-http://localhost:8929}"
TOKEN="${GITLAB_TOKEN:-glpat-apihub-e2e-demo-token01}"   # demo-only local token
GROUP="apihub-demo"
BRANCH="${1:-2026.5}"
SEED_DIR="$(cd "$(dirname "$0")/seed-v2" && pwd)"
HOSTPART="${GITLAB_URL#*://}"

push_branch() {
  local proj="$1" msg="$2"; shift 2
  local work
  work="$(mktemp -d)"
  git clone -q "http://root:$TOKEN@$HOSTPART/$GROUP/$proj.git" "$work"
  git -C "$work" checkout -q -B "$BRANCH"
  local pair
  for pair in "$@"; do
    local src="${pair%%=*}" dst="${pair#*=}"
    mkdir -p "$work/$(dirname "$dst")"
    cp "$SEED_DIR/$src" "$work/$dst"
  done
  git -C "$work" add -A
  if git -C "$work" diff --cached --quiet; then
    echo "$GROUP/$proj@$BRANCH: no changes vs current branch content"
  else
    git -C "$work" -c user.name="e2e-demo seeder" -c user.email="e2e-demo@example.com" \
      commit -q -m "$msg"
    git -C "$work" push -q --force origin "$BRANCH"
    echo "pushed $GROUP/$proj@$BRANCH"
  fi
  rm -rf "$work"
}

push_branch ddl  "Simulate next version: retry_after, count->bigint, drop csv_dashboard_publication.message, add report_subscription" \
  "ddl/apihub_ddl.sql=db/ddl/apihub_ddl.sql"
push_branch docs "Simulate next version: workbook updated for the DDL change above" \
  "docs/comments-and-pfk.xlsx=docs/comments-and-pfk.xlsx"

echo
echo "verify: tree of $GROUP/ddl@$BRANCH (path db/ddl):"
curl -s -H "PRIVATE-TOKEN: $TOKEN" \
  "$GITLAB_URL/api/v4/projects/$GROUP%2Fddl/repository/tree?path=db/ddl&recursive=true&ref=$BRANCH" \
  | grep -oE '"path":"[^"]+"' || { echo "FATAL: ddl tree listing failed" >&2; exit 1; }
echo "seed-branch OK ($BRANCH)"
