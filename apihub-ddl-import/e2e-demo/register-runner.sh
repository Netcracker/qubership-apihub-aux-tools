#!/usr/bin/env bash
# Register a GitLab Runner against the e2e-demo GitLab for the
# apihub-demo group, and set the CI/CD variables the .gitlab-ci.yml job
# needs on apihub-demo/docs. Idempotent: safe to re-run.
#
# Prerequisite: `docker compose up -d gitlab-runner` (see docker-compose.yaml)
# and seed-gitlab.sh already run (group/projects must exist).
set -euo pipefail

GITLAB_URL="${GITLAB_URL:-http://localhost:8929}"          # used by this script's own curl calls (runs on the host)
RUNNER_GITLAB_URL="${RUNNER_GITLAB_URL:-http://host.docker.internal:8929}"  # used by config.toml — the runner container
                                                             # can't reach GitLab via "localhost" (that's itself);
                                                             # host.docker.internal reaches the host's published port
                                                             # (Docker Desktop), same trick the job containers use.
TOKEN="${GITLAB_TOKEN:-glpat-apihub-e2e-demo-token01}"   # demo-only local token
GROUP="apihub-demo"
RUNNER_CONTAINER="${RUNNER_CONTAINER:-apihub-e2e-runner}"
APIHUB_API_KEY_VALUE="${APIHUB_API_KEY:-8231f12d-054a-4a4b-be50-fda715694f3b}"
APIHUB_URL_VALUE="${DEMO_APIHUB_URL:-http://host.docker.internal:8090}"

api() { curl -s -H "PRIVATE-TOKEN: $TOKEN" "$@"; }
first_id() { grep -oE '"id":[0-9]+' | head -1 | cut -d: -f2; }

# ---- CI job token signing key ----------------------------------------------
# Without this, EVERY job dispatch fails instantly with failure_reason
# "scheduler_failure" (server-side RuntimeError "CI job token signing key is
# not set" in lib/ci/job_token/jwt.rb) — the runner never even gets a chance
# to run anything, and produces no trace/log output at all, which makes this
# very non-obvious to diagnose from the runner or pipeline side. This is a
# one-time, instance-wide setting; the omnibus image doesn't auto-generate it
# for this minimal single-container setup.
docker exec "${GITLAB_CONTAINER:-apihub-e2e-gitlab}" \
  gitlab-rails runner "
    settings = Gitlab::CurrentSettings.current_application_settings
    if settings.ci_job_token_signing_key.blank?
      settings.update!(ci_job_token_signing_key: OpenSSL::PKey::RSA.new(2048).to_pem)
      puts 'ci_job_token_signing_key generated'
    else
      puts 'ci_job_token_signing_key already set'
    end
  " 2>&1 | tail -3

gid="$(api "$GITLAB_URL/api/v4/groups/$GROUP" | first_id || true)"
[ -n "$gid" ] || { echo "FATAL: group $GROUP not found — run seed-gitlab.sh first" >&2; exit 1; }
docs_pid="$(api "$GITLAB_URL/api/v4/projects/$GROUP%2Fdocs" | first_id || true)"
[ -n "$docs_pid" ] || { echo "FATAL: project $GROUP/docs not found — run seed-gitlab.sh first" >&2; exit 1; }

# ---- drop any runner(s) from a previous run of this script ----------------
# GitLab never re-exposes a runner's auth token after creation, so true
# idempotency means "replace", not "reuse": this script is the sole owner of
# runners on this group, so remove whatever's there before creating a fresh one.
for old_id in $(api "$GITLAB_URL/api/v4/groups/$gid/runners" | grep -oE '"id":[0-9]+' | cut -d: -f2 || true); do
  api -X DELETE "$GITLAB_URL/api/v4/runners/$old_id" >/dev/null
  echo "removed stale runner id $old_id"
done

# ---- runner: create a group-scoped runner, get its auth token -------------
# GitLab 16+ issues an authentication token (glrt-...) via this endpoint; the
# older `gitlab-runner register --registration-token` flow is not used here.
echo "creating/registering group runner for $GROUP (id $gid)..."
runner_resp="$(curl -s -X POST -H "PRIVATE-TOKEN: $TOKEN" \
  --data-urlencode "runner_type=group_type" \
  --data-urlencode "group_id=$gid" \
  --data-urlencode "description=apihub-ddl-import e2e docker runner" \
  --data-urlencode "run_untagged=true" \
  "$GITLAB_URL/api/v4/user/runners")"
runner_token="$(echo "$runner_resp" | grep -oE '"token":"[^"]+"' | head -1 | cut -d'"' -f4 || true)"
if [ -z "$runner_token" ]; then
  echo "FATAL: could not create/obtain a runner token. Response was:" >&2
  echo "$runner_resp" >&2
  echo "If this instance disabled the new-style runner API, enable it or" >&2
  echo "fall back to the legacy 'gitlab-runner register --registration-token'" >&2
  echo "flow (Admin Area > CI/CD > Runners > registration token)." >&2
  exit 1
fi
echo "runner token obtained"

# ---- render config.toml for the runner container --------------------------
CONFIG_TOML="$(cat <<EOF
concurrent = 4
check_interval = 3

[[runners]]
  name = "apihub-e2e-runner"
  url = "$RUNNER_GITLAB_URL"
  token = "$runner_token"
  executor = "docker"
  # GitLab tells job containers to check out sources from the server's own
  # external_url (http://localhost:8929, meaningless inside a job container —
  # that's the container itself). clone_url overrides just that, without
  # touching the server's public external_url.
  clone_url = "$RUNNER_GITLAB_URL"
  [runners.docker]
    image = "golang:1.25"
    privileged = false
    volumes = ["/var/run/docker.sock:/var/run/docker.sock", "/cache"]
    pull_policy = ["if-not-present"]
EOF
)"
docker exec -i "$RUNNER_CONTAINER" sh -c "cat > /etc/gitlab-runner/config.toml" <<EOF
$CONFIG_TOML
EOF
docker restart "$RUNNER_CONTAINER" >/dev/null
echo "runner config written, container restarted"

# ---- CI/CD variables on apihub-demo/docs -----------------------------------
set_var() {
  local key="$1" value="$2" masked="${3:-false}"
  # PUT if it exists, else POST — avoids "already exists" errors on re-run.
  if api "$GITLAB_URL/api/v4/projects/$docs_pid/variables/$key" | grep -q '"key"'; then
    api -X PUT --data-urlencode "value=$value" --data-urlencode "masked=$masked" \
      "$GITLAB_URL/api/v4/projects/$docs_pid/variables/$key" >/dev/null
  else
    api -X POST --data-urlencode "key=$key" --data-urlencode "value=$value" --data-urlencode "masked=$masked" \
      "$GITLAB_URL/api/v4/projects/$docs_pid/variables" >/dev/null
  fi
  echo "set CI/CD variable $key"
}
set_var "APIHUB_API_KEY" "$APIHUB_API_KEY_VALUE" true
set_var "DDL_GITLAB_TOKEN" "$TOKEN" true   # demo-only: reuses the root PAT; scope a read-only token for real use
set_var "APIHUB_URL" "$APIHUB_URL_VALUE" false

echo
echo "register-runner OK. Confirm the runner is online:"
echo "  curl -s -H 'PRIVATE-TOKEN: $TOKEN' $GITLAB_URL/api/v4/projects/$docs_pid/runners"
