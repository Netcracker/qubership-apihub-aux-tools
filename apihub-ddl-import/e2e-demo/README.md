# apihub-ddl-import e2e demo

Local infrastructure for demoing the full `apihub-ddl-import` pipeline
(GitLab sources → merge → APIHUB publish → groups → exports).

All credentials in this folder are throwaway values for a local demo instance —
never reuse them anywhere real.

## Step 1 — local GitLab

```bash
docker compose up -d     # first boot takes ~5 min (container turns "healthy")
```

| What | Value |
|---|---|
| Web UI / API | http://localhost:8929 |
| Login | `root` / `Otter-Quartz-9271!` |
| Root PAT (scope `api`) | `glpat-apihub-e2e-demo-token01` |
| SSH (unused by the demo) | port `2224` |

The root personal access token is created once after first boot with:

```bash
docker exec apihub-e2e-gitlab gitlab-rails runner "t = User.find_by_username('root').personal_access_tokens.create!(scopes: ['api'], name: 'e2e-demo', expires_at: 364.days.from_now); t.set_token('glpat-apihub-e2e-demo-token01'); t.save!"
```

Smoke check:

```bash
curl -H "PRIVATE-TOKEN: glpat-apihub-e2e-demo-token01" http://localhost:8929/api/v4/version
```

Lifecycle:

```bash
docker compose stop      # keep data
docker compose start
docker compose down -v   # full reset — wipes data volumes, next boot re-seeds
```

Note: `initial_root_password` (and the PAT) only apply to a fresh instance;
after `down -v` re-run the PAT command above.

Gotchas learned the hard way:

- The root password must not contain `gitlab` (or other denylisted words) —
  the admin seed then fails with "Password must not contain commonly used
  combinations of words and letters" and the instance comes up with **zero
  users**. Recover without a reset:
  `docker exec -e GITLAB_ROOT_PASSWORD='<new>' -e GITLAB_ROOT_EMAIL='admin@example.com' -e FILTER=admin apihub-e2e-gitlab gitlab-rake db:seed_fu`
- The container turning `healthy` only means the web service is up; DB
  seeding can still be running for a minute or two after that.

## Step 2 — seed the DDL and comments repositories

Source data lives in `seed/`:

- `seed/ddl/apihub_ddl.sql` — the APIHUB backend schema dump (72 tables,
  492 columns, plus a view and functions the tool ignores).
- `seed/docs/comments-and-pfk.xlsx` — generated workbook covering **every**
  table and column, 13 domains (→ APIHUB groups), real PK marks, and six
  curated Constraint marks that exercise each merge rule once
  (3 resolved FKs / 1 unresolved / 1 ambiguous / 1 already-in-DDL, plus
  3 `COMMENT_EXISTS` columns). Regenerate after changing the DDL with:

  ```bash
  go run ./e2e-demo/gen-comments -ddl e2e-demo/seed/ddl -out e2e-demo/seed/docs/comments-and-pfk.xlsx
  ```

  The generator reuses the tool's own DDL parser and merge, and fails unless
  the merge outcome is exactly the expected six warnings.

Create the GitLab group/projects and (force-)push the seed content:

```bash
bash e2e-demo/seed-gitlab.sh
```

| Repo | Content |
|---|---|
| http://localhost:8929/apihub-demo/ddl | `db/ddl/apihub_ddl.sql` |
| http://localhost:8929/apihub-demo/docs | `docs/comments-and-pfk.xlsx` |

Verified: `--dry-run` with GitLab sources fetches, parses and merges cleanly:

```bash
GITLAB_TOKEN=glpat-apihub-e2e-demo-token01 go run . --dry-run \
  --version 2026.3 --previous-version none \
  --ddl-source-type gitlab --ddl-repo http://localhost:8929/apihub-demo/ddl \
  --ddl-branch main --ddl-path db/ddl \
  --comments-source-type gitlab --comments-repo http://localhost:8929/apihub-demo/docs \
  --comments-branch main --comments-path docs/comments-and-pfk.xlsx
```

## Step 3 — run the full pipeline against a dev APIHUB

`setup-packages.sh` and `run-demo.sh` default to `http://localhost:8090` with
a matching API key — point them at **whatever APIHUB backend + DDL-capable
build-task-consumer you already have running locally**. Override with
`APIHUB_URL`/`APIHUB_TOKEN` env vars if yours differs.

If you don't have one, `backend/start-backend.sh` stands up a second,
**isolated** instance on `:8095` (won't collide with anything on `:8090`),
using the actual `qubership-apihub-backend_3` checkout (not a container):

```bash
docker compose up -d                                              # 1. GitLab (already up from step 1)
BACKEND_REPO=/c/repo/qubership-apihub-backend_3 bash backend/start-backend.sh   # 2. isolated APIHUB backend on :8095 (foreground; needs go 1.25+, branch table_groups)
bash start-builder.sh                                              # 3. build-task-consumer (feature-ddl image) for :8095
APIHUB_URL=http://localhost:8095 APIHUB_TOKEN=apihub-e2e-zero-day-access-token-0123456789 bash setup-packages.sh
APIHUB_URL=http://localhost:8095 APIHUB_TOKEN=apihub-e2e-zero-day-access-token-0123456789 bash run-demo.sh 2026.3 none
```

Otherwise, against an existing backend on `:8090`:

```bash
bash setup-packages.sh              # DEMO workspace + DEMO.RDM package
bash run-demo.sh 2026.3 none        # publish first version
bash run-demo.sh 2026.4 2026.3      # publish a second version (exercises the changes export)
```

`start-backend.sh` generates a throwaway JWT key and `config.yaml` under
`backend/` (gitignored — never commit it), creates the `apihub_e2e` Postgres
database (reuses the `postgres` container from `docker-compose/DB` in the
backend repo, credentials `apihub`/`apihub`), and runs `go run .` from
`qubership-apihub-service/`.

Verified results (2026-09-10, both versions green end to end):

- **Publish**: `DEMO.RDM@2026.3` (baseline) and `@2026.4` (vs `2026.3`), both
  `status release`, build completes in ~8-10s.
- **72 DDL entities published** for both versions — confirms the backend
  builder does turn the `.sql` source into DDL entities and reads the
  generated `COMMENT ON` into their descriptions (this was flagged as an
  unverified risk when the tool was implemented; now confirmed).
- **13/13 groups created** each version (groups are scoped per package
  version, so a republish creates fresh groups rather than updating old
  ones).
- **Entities export**: 72 rows, `Group` column already filled by the
  backend for every row (`groupFilled: 0` in the report is the *good*
  outcome — nothing left for the tool to backfill, and it matches the
  workbook domains exactly, so zero `EXPORT_GROUP_MISMATCH` warnings).
- **Changes export** (2026.4 only, since 2026.3 had no previous version):
  72 rows with an `Analytics Severity` column added. Note: since the DDL and
  comments were byte-identical between 2026.3 and 2026.4, every table still
  shows 1 "non-breaking" change — that's the backend's own version
  comparison bookkeeping a re-publish as a trivial change, not something
  `apihub-ddl-import` controls.
- Merge outcome identical across both runs: 72/72 tables, 492/492 columns
  matched, exactly the 6 warnings the workbook was built to demonstrate (see
  step 2).
- Spot-checked the enriched DDL: `COMMENT ON TABLE activity_tracking IS
  '...'` and the generated FK `ALTER TABLE` blocks are present, each wrapped
  in `-- apihub-ddl-import:begin/end table=...` markers.

### Gotchas

- **Port collision on Olric's memberlist port.** If another APIHUB backend
  instance already runs locally with default `olric` ports, this backend
  panics at startup (`Failed to start TCP listener ... bind: Only one usage
  of each socket address...`). `config.template.yaml` pins non-default
  `bindPort`/`memberlistPort` (47385/47386) to avoid it — change them again
  if those also collide.
- `go.sum` is gitignored in the backend repo; `start-backend.sh` runs
  `go mod tidy` once if missing (cold: a couple of minutes).
- The backend needs branch **`table_groups`** (or a checkout with that work
  merged) — the `/ddl/groups` API and the exports' `Group` column only exist
  there. The build-task-consumer needs matching DDL support; confirmed
  working on both the `feature-ddl` image and a `:dev` image built from
  `develop` — check yours by publishing once and confirming DDL entities
  actually appear (`GET .../ddl/entities`), not just that the build completes.
- Republishing byte-identical DDL as a new version still produces a non-empty
  changes export (the backend's own version-comparison bookkeeping treats a
  republish as a trivial change) — not a bug in the tool, and the exact
  shape of that bookkeeping has varied between backend builds observed here
  (one gave every table a single "non-breaking" change; another gave zero
  rows for genuinely-unchanged tables). For a changes export worth actually
  reading, publish a version with a **real** DDL/comments diff — see step 4.

## Step 4 — simulate a schema change (next-version branches)

`seed/` (used by steps 2-3) is the **baseline** schema. `seed-v2/` is a
**follow-up** version with a small, realistic diff, pushed to a branch named
`2026.5` (not `main`) in both repos, so publishing it exercises real
structural changes instead of a no-op republish:

| Change | Demonstrates |
|---|---|
| New table `report_subscription`, domain `Reporting` | a brand-new DDL group created on publish |
| `report_subscription.build_id`, undeclared FK in DDL, `Constraint=FK` in the workbook | a new FK resolved purely by naming convention (`build_id` → `build`) |
| New column `build.retry_after` (nullable) | a non-breaking addition to an existing, already-published table |
| `endpoint_calls.count`: `integer` → `bigint` | a data-type change |
| `csv_dashboard_publication.message` dropped | a breaking column removal, DDL and workbook row both updated |

Regenerate after changing `seed-v2/ddl/apihub_ddl.sql` further:

```bash
go run ./e2e-demo/gen-comments-v2 -ddl e2e-demo/seed-v2/ddl -in e2e-demo/seed/docs/comments-and-pfk.xlsx -out e2e-demo/seed-v2/docs/comments-and-pfk.xlsx
```

`gen-comments-v2` edits the **baseline** workbook incrementally (add/update/
remove specific rows) rather than regenerating it from scratch — matching how
a real comments doc would be maintained alongside a schema change — and
self-verifies (73/73 tables, 499/499 columns matched, exactly 4 FKs
generated, the same 6 base warnings, domain `Reporting` present) before
writing the file.

Push the branch:

```bash
bash e2e-demo/seed-branch.sh 2026.5
```

Verified (2026-09-11): `--dry-run` fetching `--ddl-branch 2026.5
--comments-branch 2026.5` from the live GitLab parses and merges cleanly —
73/73 tables, 499/499 columns matched, 4 FKs generated (including the new
convention-resolved one), same 6 base warnings, 14 domains. `main` on both
repos is untouched.

Not yet published — to publish it as the next version:

```bash
DDL_BRANCH=2026.5 COMMENTS_BRANCH=2026.5 bash run-demo.sh 2026.5 2026.4
```

## Step 5 — GitLab CI job (import/export via "Run pipeline")

Gives analysts a GitLab-native interface for the tool instead of a local CLI:
a job living **in `apihub-demo/docs` itself**, triggered manually, with typed
"Run pipeline" input fields — including a real checkbox for
`export_custom_columns` (on = today's enriched Group/Analytics Severity
export, off = APIHUB's raw export, i.e. `--skip-enrichment`).

Bring up a runner and wire it to the docs project:

```bash
docker compose up -d gitlab-runner   # new service, same compose file as step 1
bash e2e-demo/register-runner.sh     # creates a group runner, sets APIHUB_API_KEY /
                                      # DDL_GITLAB_TOKEN / APIHUB_URL as masked CI/CD
                                      # variables on apihub-demo/docs
```

`register-runner.sh` uses the modern GitLab runner-creation API
(`POST /api/v4/user/runners`, group-scoped to `apihub-demo` so it also covers
`apihub-demo/ddl` if ever needed) rather than the legacy
`gitlab-runner register --registration-token` flow, and renders the runner
container's `config.toml` directly with the returned authentication token.
The runner uses the **docker executor bound to the host's docker socket** —
a local-demo simplification (documented in `docker-compose.yaml`); the
`.gitlab-ci.yml` template itself has no such dependency.

`.gitlab-ci.yml` itself is **seeded content**, not hand-pushed: it lives at
`seed/docs/.gitlab-ci.yml` and `seed-gitlab.sh` now pushes it to
`apihub-demo/docs` alongside the workbook. Re-run `seed-gitlab.sh` after the
runner is registered to make sure it's present:

```bash
bash e2e-demo/seed-gitlab.sh
```

Trigger a run: open http://localhost:8929/apihub-demo/docs/-/pipelines/new —
`export_custom_columns` renders as a checkbox next to the text/dropdown
inputs for `version`, `previous_version`, `status`, `ddl_repo`, `ddl_branch`,
`ddl_path`, `package_id`. Or via the API:

```bash
curl -s -X POST -H "PRIVATE-TOKEN: glpat-apihub-e2e-demo-token01" \
  --data-urlencode "ref=main" \
  --data-urlencode "inputs[version]=2026.5" \
  --data-urlencode "inputs[previous_version]=2026.4" \
  --data-urlencode "inputs[export_custom_columns]=false" \
  "http://localhost:8929/api/v4/projects/<docs-project-id>/pipeline"
```

Artifacts (`report.md`, `report.json`, `export/`, `enriched/`) are attached to
the job (`when: always`, so a failed run still uploads the report).

The job currently builds `apihub-ddl-import` from source on every run
(`git clone --branch ddl_export_import ... && go build`) — no release has
been cut for the tool yet. Once one exists, only the "Build the tool" script
step needs to change to a `curl` of the release binary; everything else
(inputs, secrets, artifacts) stays as-is.

### Verified (2026-09-21)

*(filled in after the live run below)*
