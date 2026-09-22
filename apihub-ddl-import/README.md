# apihub-ddl-import

Merges a raw PostgreSQL DDL with an Excel workbook of table/column comments and PK/FK marks, publishes the enriched DDL to APIHUB as a package version, creates per-domain **DDL table groups**, and downloads the DDL xlsx exports enriched with **Group** and **Analytics Severity** columns.

One command — several artifacts:

```text
ddl-import-out/
├── raw/                      # inputs exactly as fetched (ddl/ + the workbook)
├── enriched/                 # DDL with generated COMMENT ON + FK statements (artifact #1)
├── report.md, report.json    # merge + pipeline report (artifact #2)
└── export/
    ├── DDLEntities_<pkg>_<ver>.xlsx        # enriched entities export (artifact #3)
    ├── DDLChanges_<pkg>_<ver>.xlsx         # enriched changes export (artifact #4)
    └── *.orig.xlsx                         # backend originals, kept for reference
```

## Pipeline

1. **Preflight** (skipped with `--dry-run`): the target version and the **required** `--previous-version` are checked in APIHUB (`404` on the previous version is fatal — fail fast before any fetching). Use `--previous-version none` for the very first publish.
2. **Fetch** DDL `.sql` files and the comments workbook from local paths or GitLab (`PRIVATE-TOKEN` auth, works for gitlab.com and self-hosted).
3. **Validate + parse.** The workbook must have sheets `List of Tables` (`Domain | Table Name | Table Description | External Table`) and `Tables Specifications` (`Domain | Table Name | Column Name | Data Type | IsPK | Constraint | Column Description | Deployment Release | RDB | ExternalDB`). Broken headers or a fully empty workbook abort the run (nothing is published). DDL is parsed with the **real PostgreSQL grammar** ([libpg_query](https://github.com/pganalyze/libpg_query) compiled to WebAssembly — no cgo).
4. **Merge** (rules below) → enriched DDL + report (console, `report.md`, `report.json`).
5. **Publish**: enriched files are zipped and posted to `POST /api/v2/packages/{id}/publish` (multipart `config` + `sources`), status `draft`/`release` per `--status`; the build status is polled until `complete`/`error`.
6. **Verify**: the published version must expose DDL entities (`GET .../ddl/entities`); zero entities is a hard error (the APIHUB builder likely does not parse `.sql` sources).
7. **Groups**: one DDL table group per workbook `Domain` (`POST .../ddl/groups`). On republish into an existing revision (duplicate group, code `8403`) the group membership is **replaced** via `PATCH`. **If not a single group ends up created** — whether the backend has no `/ddl/groups` at all (404/405, or 421 "Requested unknown endpoint", observed live on some deployments) or every individual domain fails for some other reason — that's a failed deliverable, not a warning: the run exits with `F_GROUPS_FAILED`. A partial result (some domains succeed, some don't) stays non-fatal (`GROUP_CREATE_FAILED` per domain). Use `--skip-groups` to publish without groups at all when that's intentional.
8. **Exports**: `GET .../ddl/export/entities` (always) and `GET .../ddl/export/changes` (when a previous version exists) are downloaded and enriched:
   - **Group** — filled from the workbook domain for rows the backend left empty; if the export has no Group column at all (older backend), the tool appends one.
   - **Analytics Severity** (changes only) — computed per row from the per-entity change list (`GET .../ddl/entities/{id}/changes`) using the mapping: `annotation`, `semi-breaking` (*requiring attention*), `breaking`, `deprecated` → **breaking**; `non-breaking` → **non-breaking**; `unclassified` → **unclassified**; **exception: any data-type change is always breaking** (detected by a configurable regex on the change description). When per-change data is unavailable the value falls back to the severity count columns (reported as `ANALYTICS_FALLBACK_COUNTS`).

## Merge rules

- **Matching is normalization-based**: names are trimmed, lowercased, and stripped of `_`/`-`/spaces, so `ledgerMoniker` = `ledger_moniker` = `LEDGER_MONIKER`. The `public.` schema prefix of the DDL is ignored for matching (the workbook has schema-less names).
- **Comments**: non-empty `Table Description` / `Column Description` become `COMMENT ON TABLE` / `COMMENT ON COLUMN` statements appended after each table's `CREATE TABLE` inside marker blocks:

  ```sql
  -- apihub-ddl-import:begin table=bc_sable_thicket
  COMMENT ON TABLE public.bc_sable_thicket IS 'The table stores ...';
  COMMENT ON COLUMN public.bc_sable_thicket.sable_thicket_id IS 'Unique identifier ...';
  -- apihub-ddl-import:end table=bc_sable_thicket
  ```

  Re-running the tool over its own output is a **no-op**: marker blocks are stripped before parsing and regenerated.
- **Primary keys**: the DDL `PRIMARY KEY` always wins and is **never rewritten**. Workbook `IsPK` marks that contradict it produce one `PK_MISMATCH` warning per table.
- **Foreign keys**: the workbook's `Constraint` column carries only `PK`/`FK`/`PFK` — no target. FKs are resolved **only by the `<table_stem>_id` naming convention** (`sable_thicket_id` → table `bc_sable_thicket`, version suffixes `_vN` ignored; same-domain tables win ties). Resolvable marks become `ALTER TABLE ... ADD CONSTRAINT ... FOREIGN KEY` statements in a per-file block; unresolvable ones are warnings. `PFK` without `IsPK=Y` is flagged as contradictory input.
- **Dirty input**: cell values are trimmed; `IsPK` accepts `Y/YES/1/TRUE` (any case), `Constraint` accepts any case. Exact duplicate rows are dropped with a warning; duplicate rows with **conflicting descriptions** produce **no comment at all** — the tool never silently picks one.
- **Fatal cases** (exit 1, nothing published): broken/renamed sheet headers, both sheets empty, unparsable DDL, previous version not found, publish failure.

Warnings never block publishing (use `--strict` to stop before publish when any warning exists).

## Usage

```bash
# Everything in the config file, per-run params on the command line:
apihub-ddl-import --version "2026.2" --previous-version "2026.1" --status draft

# First publish of a package (no baseline):
apihub-ddl-import --version "2026.1" --previous-version none --status draft

# Local files, no APIHUB calls — validate the merge and read the report:
apihub-ddl-import --dry-run --version test --previous-version none \
  --ddl-path ./db/ddl --comments-path ./docs/comments-and-pfk.xlsx

# Everything via flags:
apihub-ddl-import \
  --apihub-url https://apihub.example.com --package-id MY.WS.REPORTING-DM \
  --version "2026.2" --previous-version "2026.1" --status release \
  --ddl-source-type gitlab --ddl-repo https://gitlab.example.com/group/ddl-repo \
  --ddl-branch main --ddl-path db/ddl \
  --comments-source-type gitlab --comments-repo https://gitlab.example.com/group/docs \
  --comments-branch main --comments-path docs/comments-and-pfk.xlsx
```

### Configuration

Semi-static parameters live in `ddl-import.yaml` (auto-loaded from the working directory, or `--config <path>`), see [ddl-import.example.yaml](./ddl-import.example.yaml). **Precedence: flag > env > config file > default.**

Secrets via environment: `APIHUB_API_KEY`, `GITLAB_TOKEN` (both sources), `DDL_GITLAB_TOKEN` / `COMMENTS_GITLAB_TOKEN` (per-source, beat the shared one). Secrets are never printed or written into reports.

### Flags

| Flag | Meaning |
|------|---------|
| `--version` | Version to publish (**required**) |
| `--previous-version` | Previous published version, or `none` for the first publish (**required**; verified in APIHUB fail-fast) |
| `--status` | `draft` (default) or `release` |
| `--config` | Config file path (default `ddl-import.yaml` when present) |
| `--apihub-url`, `--apihub-api-key`, `--package-id` | APIHUB coordinates |
| `--ddl-source-type/-repo/-branch/-path/-token` | DDL source (`file` or `gitlab`; path = folder with `.sql` or one file) |
| `--comments-source-type/-repo/-branch/-path/-token` | Comments source (path = the `.xlsx` file) |
| `--output-dir` | Artifacts directory (default `./ddl-import-out`) |
| `--version-labels` | Comma-separated APIHUB version labels |
| `--publish-timeout` | Publish poll timeout (default `15m`) |
| `--dry-run` | Merge + report only, zero APIHUB calls |
| `--strict` | Exit 3 before publish when the merge has warnings |
| `--skip-groups`, `--skip-exports` | Skip the respective steps |
| `--skip-enrichment` | Keep APIHUB's raw exports — do not add the `Group`/`Analytics Severity` custom columns |
| `--insecure-skip-tls-verify` | Skip TLS verification (APIHUB and GitLab) |
| `--no-color`, `--debug` | Console output control |

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success (warnings allowed) |
| `1` | Fatal error — nothing published (or a later pipeline stage failed) |
| `2` | Usage / configuration error |
| `3` | `--strict` stop: merge produced warnings |

## Warning codes (report.json)

`TBL_DDL_ONLY`, `TBL_XLSX_ONLY`, `COL_DDL_ONLY` (columns of workbook-missing tables are counted too), `COL_XLSX_ONLY`, `DESC_EMPTY_TABLE`, `DESC_EMPTY_COLUMN`, `PK_MISMATCH`, `FK_UNRESOLVED`, `FK_AMBIGUOUS`, `FK_EXISTS`, `PFK_WITHOUT_PK`, `DUP_EXACT`, `DUP_CONFLICT`, `BAD_FLAG`, `SHEET_EMPTY`, `COMMENT_EXISTS`, `DUP_DDL_TABLE`, `GROUPS_API_UNAVAILABLE`, `GROUP_CREATE_FAILED`, `EXPORT_UNAVAILABLE`, `EXPORT_LAYOUT_UNKNOWN`, `EXPORT_GROUP_MISMATCH`, `ANALYTICS_FALLBACK_COUNTS`, `DESCRIPTIONS_MISSING`.

Fatal codes: `F_XLSX_HEADER`, `F_XLSX_EMPTY`, `F_NO_DDL_FILES`, `F_DDL_PARSE`, `F_PREV_VERSION_NOT_FOUND`, `F_PUBLISH_FAILED`, `F_APIHUB_UNREACHABLE`, `F_NO_DDL_ENTITIES`, `F_GROUPS_FAILED`.

## Backend endpoints used

| Purpose | Endpoint |
|---------|----------|
| Version preflight | `GET /api/v3/packages/{id}/versions/{version}` |
| Publish | `POST /api/v2/packages/{id}/publish` (multipart `config` + `sources`) |
| Publish status | `GET /api/v2/packages/{id}/publish/{publishId}/status` |
| DDL entities | `GET /api/v1/packages/{id}/versions/{v}/ddl/entities` |
| Per-entity changes | `GET /api/v1/packages/{id}/versions/{v}/ddl/entities/{ddlEntityId}/changes` |
| Table groups | `POST/PATCH /api/v1/packages/{id}/versions/{v}/ddl/groups[/{groupName}]` |
| Exports | `GET /api/v1/packages/{id}/versions/{v}/ddl/export/entities` and `.../export/changes` |

Auth: `api-key` header.

## Build and test

```bash
cd apihub-ddl-import
go build .
go test ./...
```

The test suite includes the ten fixture cases from the DDL-merge spec (`testdata/cases/case-01…10`) asserting exact warning counts, FK resolution (38 resolved / 11 unresolved / 3 contradictions in the FK case), the two fatal cases, and enrichment across three export layout generations.

### Manual smoke against a dev APIHUB

1. `apihub-ddl-import --version 2026.1 --previous-version none --status draft --ddl-path testdata/ddl --comments-path testdata/cases/case-01/comments-and-pfk.xlsx --apihub-url … --package-id … `
   — expect a clean merge, publish `complete`, 75 DDL entities, 12 groups, an enriched entities export.
2. Re-run with `--version 2026.2 --previous-version 2026.1` and the case-10 workbook — expect warnings in the report and the enriched **changes** export with the `Analytics Severity` column.

## Running as a GitLab CI job

For analysts who shouldn't need a local CLI, the tool can run as a GitLab pipeline **inside the comments/docs repository itself** — "Run pipeline" becomes the UI for the import+export action, with typed input parameters (including a real checkbox) and the exports published as job artifacts.

A ready-to-copy template lives at [`e2e-demo/seed/docs/.gitlab-ci.yml`](./e2e-demo/seed/docs/.gitlab-ci.yml) (also seeded into the local demo GitLab's `apihub-demo/docs` repo by `e2e-demo/seed-gitlab.sh` — see [e2e-demo/README.md](./e2e-demo/README.md) for a full working example, including how to stand up a GitLab Runner). Key points:

- Uses GitLab's [pipeline input parameters](https://docs.gitlab.com/ci/inputs/) (`spec:inputs`) for `version`, `previous_version`, `status`, the DDL repo coordinates, `package_id`, and an **`export_custom_columns` boolean** — GitLab renders `type: boolean` as an actual checkbox on the "Run pipeline" page. Unchecking it maps to `--skip-enrichment`.
- The comments workbook is read straight from the job's own checkout (`--comments-source-type file --comments-path docs/comments-and-pfk.xlsx`) — no GitLab token needed for that side.
- Secrets (`APIHUB_API_KEY`, a `DDL_GITLAB_TOKEN` for the external DDL repo) are project CI/CD variables, picked up automatically through the tool's existing environment-variable support — they never appear as pipeline inputs or in logs.
- The job currently builds the tool from source (`git clone` a pinned ref of this repo + `go build`) since no `apihub-ddl-import` release has been cut yet; once one exists, swap that one step for a `curl` of the release binary.
- `artifacts: { when: always, paths: [ddl-import-out/report.md, ddl-import-out/report.json, ddl-import-out/export/, ddl-import-out/enriched/] }` so both successful and failed runs leave the report and exports attached to the job.

## Known assumptions / limitations

- **The APIHUB builder must parse `.sql` sources into DDL contracts.** The tool verifies this after publish (`F_NO_DDL_ENTITIES` otherwise). Likewise, entity descriptions are expected to come from the generated `COMMENT ON` statements; if all published descriptions are empty the tool reports `DESCRIPTIONS_MISSING`.
- Generated FK constraints are **documentation** for APIHUB, not executable migrations: targets with composite PKs are referenced by a single column with an explanatory `-- note:` line.
- The changes export needs a stored comparison on the backend; when it is unavailable the tool reports `EXPORT_UNAVAILABLE` and continues.
- `deprecated` → breaking in the Analytics Severity mapping is an assumption (the documented mapping table does not mention it).
- The data-type-change detection regex is heuristic; override it via `analytics.dataTypeChangeRegex` when the builder's change wording differs.
