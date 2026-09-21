#!/usr/bin/env bash
# Run the full apihub-ddl-import pipeline against the e2e-demo stack: GitLab
# sources -> merge -> publish to the local APIHUB -> DDL groups -> exports.
#
# Prerequisites (each is its own script):
#   docker compose up -d              # GitLab (docker-compose.yaml)
#   bash backend/start-backend.sh     # APIHUB backend on :8095
#   bash start-builder.sh             # build-task-consumer
#   bash setup-packages.sh            # DEMO / DEMO.RDM packages
#   bash seed-gitlab.sh               # DDL + comments repos
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION="${1:-2026.4}"
PREVIOUS="${2:-2026.3}"
OUT_DIR="${OUT_DIR:-$DIR/../ddl-import-out}"

# `go run <dir>` does not reliably relocate module resolution to <dir> — it can
# still search from the caller's inherited working directory. cd first so the
# module is found regardless of where this script was invoked from.
cd "$DIR/.."

GITLAB_TOKEN="${GITLAB_TOKEN:-glpat-apihub-e2e-demo-token01}" \
APIHUB_API_KEY="${APIHUB_API_KEY:-8231f12d-054a-4a4b-be50-fda715694f3b}" \
go run . \
  --apihub-url "${APIHUB_URL:-http://localhost:8090}" \
  --package-id "${PACKAGE_ID:-DEMO.RDM}" \
  --version "$VERSION" --previous-version "$PREVIOUS" --status release \
  --version-labels e2e-demo \
  --ddl-source-type gitlab --ddl-repo "${GITLAB_URL:-http://localhost:8929}/apihub-demo/ddl" \
  --ddl-branch "${DDL_BRANCH:-main}" --ddl-path db/ddl \
  --comments-source-type gitlab --comments-repo "${GITLAB_URL:-http://localhost:8929}/apihub-demo/docs" \
  --comments-branch "${COMMENTS_BRANCH:-main}" --comments-path docs/comments-and-pfk.xlsx \
  --output-dir "$OUT_DIR"
