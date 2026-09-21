#!/usr/bin/env bash
# Seed the e2e-demo GitLab with the DDL and comments repositories.
# Idempotent: re-running commits+pushes the current seed/ content as a new
# commit on top of whatever's already on main (a no-op commit is skipped).
# Deliberately does NOT force-push or touch branch protection — GitLab
# auto-protects a project's default branch (Maintainer-only push, no force
# push) as soon as it's first created, and root's Owner/Maintainer access is
# enough for an ordinary fast-forward push.
#
# Creates (group and projects are private):
#   apihub-demo/ddl   -> db/ddl/apihub_ddl.sql
#   apihub-demo/docs  -> docs/comments-and-pfk.xlsx, .gitlab-ci.yml
set -euo pipefail

GITLAB_URL="${GITLAB_URL:-http://localhost:8929}"
TOKEN="${GITLAB_TOKEN:-glpat-apihub-e2e-demo-token01}"   # demo-only local token
GROUP="apihub-demo"
SEED_DIR="$(cd "$(dirname "$0")/seed" && pwd)"
HOSTPART="${GITLAB_URL#*://}"

api() { curl -s -H "PRIVATE-TOKEN: $TOKEN" "$@"; }
first_id() { grep -oE '"id":[0-9]+' | head -1 | cut -d: -f2; }

# ---- group ----------------------------------------------------------------
gid="$(api "$GITLAB_URL/api/v4/groups/$GROUP" | first_id || true)"
if [ -z "$gid" ]; then
  gid="$(api -X POST "$GITLAB_URL/api/v4/groups" \
    --data-urlencode "name=$GROUP" --data-urlencode "path=$GROUP" \
    --data-urlencode "visibility=private" | first_id)"
  echo "group $GROUP created (id $gid)"
else
  echo "group $GROUP exists (id $gid)"
fi
[ -n "$gid" ] || { echo "FATAL: cannot create or resolve group $GROUP" >&2; exit 1; }

# ---- projects -------------------------------------------------------------
ensure_project() {
  local proj="$1"
  local pid
  pid="$(api "$GITLAB_URL/api/v4/projects/$GROUP%2F$proj" | first_id || true)"
  if [ -z "$pid" ]; then
    pid="$(api -X POST "$GITLAB_URL/api/v4/projects" \
      --data-urlencode "name=$proj" --data-urlencode "path=$proj" \
      --data-urlencode "namespace_id=$gid" \
      --data-urlencode "visibility=private" \
      --data-urlencode "initialize_with_readme=false" | first_id)"
    echo "project $GROUP/$proj created (id $pid)"
  else
    echo "project $GROUP/$proj exists (id $pid)"
  fi
  [ -n "$pid" ] || { echo "FATAL: cannot create project $GROUP/$proj" >&2; exit 1; }
}
ensure_project ddl
ensure_project docs

# ---- push content ---------------------------------------------------------
push_repo() {
  local proj="$1" msg="$2"; shift 2   # rest: src=dst pairs
  local work
  work="$(mktemp -d)"
  git clone -q "http://root:$TOKEN@$HOSTPART/$GROUP/$proj.git" "$work"
  git -C "$work" checkout -q -B main
  local pair
  for pair in "$@"; do
    local src="${pair%%=*}" dst="${pair#*=}"
    mkdir -p "$work/$(dirname "$dst")"
    cp "$SEED_DIR/$src" "$work/$dst"
  done
  git -C "$work" add -A
  if git -C "$work" diff --cached --quiet; then
    echo "$GROUP/$proj: no changes vs current main content"
  else
    git -C "$work" -c user.name="e2e-demo seeder" -c user.email="e2e-demo@example.com" \
      commit -q -m "$msg"
    git -C "$work" push -q origin main
    echo "pushed $GROUP/$proj"
  fi
  rm -rf "$work"
}

push_repo ddl  "Seed APIHUB DDL dump"          "ddl/apihub_ddl.sql=db/ddl/apihub_ddl.sql"
push_repo docs "Seed DDL comments workbook and CI job" \
  "docs/comments-and-pfk.xlsx=docs/comments-and-pfk.xlsx" \
  "docs/.gitlab-ci.yml=.gitlab-ci.yml"

# ---- verify the way the import tool fetches -------------------------------
echo
echo "verify: tree of $GROUP/ddl (path db/ddl):"
api "$GITLAB_URL/api/v4/projects/$GROUP%2Fddl/repository/tree?path=db/ddl&recursive=true&ref=main" \
  | grep -oE '"path":"[^"]+"' || { echo "FATAL: ddl tree listing failed" >&2; exit 1; }

sql_bytes="$(api "$GITLAB_URL/api/v4/projects/$GROUP%2Fddl/repository/files/db%2Fddl%2Fapihub_ddl.sql/raw?ref=main" | wc -c)"
xlsx_bytes="$(api "$GITLAB_URL/api/v4/projects/$GROUP%2Fdocs/repository/files/docs%2Fcomments-and-pfk.xlsx/raw?ref=main" | wc -c)"
echo "raw fetch: apihub_ddl.sql $sql_bytes bytes, comments-and-pfk.xlsx $xlsx_bytes bytes"
[ "$sql_bytes" -gt 10000 ] && [ "$xlsx_bytes" -gt 10000 ] || { echo "FATAL: raw fetch too small" >&2; exit 1; }
echo "seed OK"
