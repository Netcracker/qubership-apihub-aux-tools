#!/usr/bin/env bash
# Start the DDL-capable build-task-consumer wired to the e2e-demo backend
# (start-backend.sh, listening on :8095). Idempotent: recreates the container
# if it already exists.
set -euo pipefail

IMAGE="${BUILDER_IMAGE:-ghcr.io/netcracker/qubership-apihub-build-task-consumer:feature-ddl}"
TOKEN="${APIHUB_TOKEN:-apihub-e2e-zero-day-access-token-0123456789}"

docker rm -f apihub-e2e-builder >/dev/null 2>&1 || true
docker run -d --name apihub-e2e-builder \
  -e APIHUB_BACKEND_ADDRESS=host.docker.internal:8095 \
  -e APIHUB_API_KEY="$TOKEN" \
  -e LOG_LEVEL=INFO \
  "$IMAGE" >/dev/null
echo "apihub-e2e-builder started ($IMAGE)"
