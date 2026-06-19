# apihub-portal-package-copy

Copies published **APIHUB Portal** packages between two deployments over the REST API using **published sources archives** (`/sources`) plus **stored publish config** (`/config`). Recreates hierarchy on the target, remaps identifiers in config, and publishes versions respecting `previousVersion` order where possible within the copied set.

## CLI arguments

**Required:**

| Flag | Description |
|------|--------------|
| `--work-dir` | Working directory: stores `manifest.json` and `data/` snapshot (needed for reproducible runs and `--retry-after-fail`). |
| `--source-url` | Base URL of the source APIHUB. |
| `--source-api-key` | Source Apihub **API key** (sent as `api-key`). |
| `--source-workspace-id` | Source **workspace** package id (namespace for scope checks). |
| `--source-root-id` | Copy root: concrete **package id**, **group id**, or literal `*` (all `kind=package` descendants of `--source-workspace-id`). May be written as a **suffix** under that workspace — if the id is not already prefixed with `{workspace}.`, `{workspace}.` is prepended (composite Portal ids look like nested segments). |
| `--versions` | Comma-separated **logical** version labels (e.g. **`2025.4`**), optional **exact catalogue strings** with revision (e.g. **`2025.4@2`**), or **`*` alone** for every row **`GET /api/v3/packages/{id}/versions`** returns (see behaviour for **`@`** rules). **`*` cannot be combined with explicit names. |
| `--target-url` | Base URL of the target APIHUB. |
| `--target-api-key` | Target Apihub **API key** (`api-key`). |
| `--target-workspace-id` | Target workspace package id where the mirrored tree will be anchored. |

**Optional:**

| Flag | Description |
|------|--------------|
| `--exclude-packages` | Comma-separated package or group ids on the **source** to **omit**, with descendants (prefix rule; see below). Entries are resolved under `--source-workspace-id` like `--source-root-id` (suffix → full composite id). Empty if omitted. |
| `--retry-after-fail` | After a partial apply, skips re-fetching (if snapshot is already complete); continues publishing from checkpoint in `manifest.json`. |
| `--force-refresh-fetch` | Downloads `sources`/`config` again even when items were previously fetched successfully. |
| `--insecure-skip-tls-verify` | Skip TLS certificate verification for **both** source and target HTTPS URLs. **Insecure** (MITM risk); prefer installing the issuer CA locally. Typical error without it: `x509: certificate signed by unknown authority`. |
| `--no-color` | Disable ANSI colors in log output. Colors are also off when **`NO_COLOR`** is set or in **`CI`**. |
| `--debug` | **Verbose troubleshooting:** dumps **HTTP responses** on stderr (JSON truncated or summarized; binaries as byte counts only) plus **exact version string** comparisons when fetch skips a `--versions` entry. |

**Notes:** Omitting any required flag or passing an illegal `--versions` mix exits with usage. Paths and URLs should be quoted in your shell **where needed** (e.g. `--source-root-id '*'`). Logs go to **stderr** with phase headers, tags (`INFO` / `WARN` / `OK` / …), timestamps, and magenta `DBG` lines when **`--debug`** is set.

## Behaviour (semantics without repeating defaults)

**Source roots and exclusions.** If `--source-root-id` or an `--exclude-packages` entry is not already prefixed with `{source-workspace}.`, `{source-workspace}.` is prepended so you can pass a path relative to the workspace (segments after the workspace id).

**Exclusions.** For each exclusion entry `rule`, packages are dropped when `packageID == rule` or `packageID` has prefix `rule + "."`.

**Reuse / manifest.** A second run uses the existing `manifest.json` only when `source/target` workspace ids, `--source-root-id`, `--versions` (canonical logical list **as passed on the CLI** — without implicit `@`), and `--exclude-packages` match. Change scope or exclusion → new `--work-dir` or delete the manifest first.

**Published version identifiers (`@revision`).** Portal stores **composed** catalogue keys such as **`2025.3@2`** ( **`logical@revision`** ). The discovery API lists those strings; **`GET …/sources`**, **`GET …/config`**, etc. **must use the composed path segment**. When you pass **`--versions`** without **`@`**, the tool resolves it to **one matching API row** (normally there is only the latest revision per logical version). If multiple rows collide (unlikely with default Portal sorting), the **highest numeric revision suffix** wins. **`manifest.json`** work items persist the **resolved** catalogue string after fetch.

**Target publish.** The multipart **`POST …/publish`** accepts publish JSON **without** **`@`** on **`version`** / **`previousVersion`** — APIHUB assigns a fresh revision server-side; this tool strips **`@suffix`** from those fields while copying configs onto the target.

**Stale `previousVersion`.** Stored publish configs may reference a previous logical version that you **did not include** in this run (e.g. **`2025.2`** chains from **`2025.1`** but **`2025.1`** is omitted). Such pointers are **cleared** before publish (and when building the plan graph) so the target is not asked to attach to a non-existent row.

**Workspace snapshot folders.** `--work-dir/data/` stores each fetched package/version under **`{sanitized-package-id}/{sanitized-version-id}/`** — readable nested directories (illegal FS characters → **`_`** ; very long segments shortened with a **hash suffix** to stay within path limits). Older tool builds used opaque **base64** folders; after upgrading, start from a **new** **`--work-dir`** (or delete **`data/`**).

**Published version catalogue (pagination).** The tool walks every page of **`GET /api/v3/packages/{packageId}/versions`** (**`limit=100`**) until exhaustion. If **`--debug`** still logs a mismatch, compare the **`DBG`** **`%q`** dumps to the raw JSON from the API.

Runs through three steps: fetch snapshot from source → write plan to log → create missing nodes on target and publish in dependency order (`Fetch` / `plan` output / `Apply`).

## Authentication

Uses the **`api-key`** header on both instances. Listing packages by `parentId` (`GET /api/v2/packages?...`) has **no** `{packageId}` in the URL, so scoped keys sometimes fail; **`*`-scoped** keys are usually needed for workspace-wide discovery, group subtree listing, or `source-root-id *`.

Package-scoped workflows that only call paths containing `{packageId}` may work with a prefix-scoped key; see backend rules for your deployment.

## Build

Examples use **GNU/Bash-style** continuation (`\`) and **CMD** continuation (`^`). Build from repo root (`apihub-portal-package-copy`), then invoke `apihub-portal-package-copy` (Linux/macOS) or `apihub-portal-package-copy.exe` (Windows `go build .`).

<table>
<tr>
<th align="left" width="50%">Bash</th>
<th align="left" width="50%">Windows CMD</th>
</tr>
<tr valign="top">
<td>

```bash
cd apihub-portal-package-copy && go build .
```

</td>
<td>

```bat
cd apihub-portal-package-copy
go build .
```

</td>
</tr>
</table>

### Debugging in Cursor / VS Code

Use [.vscode/launch.json](.vscode/launch.json) (two entries: **`APIHUB-ALL`** monorepo vs **tool-only** workspace root). Set the API key in the profile’s **`env.AHPC_API_KEY`** field (recommended for temporary local edits; do **not** commit secrets) **or** define `AHPC_API_KEY` in your user/shell environment. The snapshot **`--work-dir`** is `./.debug/run1` (ignored via [`.gitignore`](.gitignore)). Install the **[Go extension](https://marketplace.visualstudio.com/items?itemName=golang.Go)** (`dlv`). Configurations pass **`--debug`** for verbose HTTP/trace output.

## Example

### Full workspace (`*`), all versions (`*`), two excludes, then retry

<table>
<tr>
<th align="left" width="50%">Bash</th>
<th align="left" width="50%">Windows CMD</th>
</tr>
<tr valign="top">
<td>

```bash
apihub-portal-package-copy \
  --work-dir ./run1 \
  --source-url https://source-ap.example \
  --source-api-key "${SRC_API_KEY:?}" \
  --source-workspace-id ws.source \
  --source-root-id '*' \
  --versions '*' \
  --exclude-packages ws.source.skip-this,ws.source.other.branch \
  --target-url https://target-ap.example \
  --target-api-key "${TGT_API_KEY:?}" \
  --target-workspace-id ws.target
```

</td>
<td>

```bat
apihub-portal-package-copy.exe ^
  --work-dir .\run1 ^
  --source-url https://source-ap.example ^
  --source-api-key %SRC_API_KEY% ^
  --source-workspace-id ws.source ^
  --source-root-id "*" ^
  --versions "*" ^
  --exclude-packages ws.source.skip-this,ws.source.other.branch ^
  --target-url https://target-ap.example ^
  --target-api-key %TGT_API_KEY% ^
  --target-workspace-id ws.target
```

</td>
</tr>
</table>

Set secrets first, e.g. `export SRC_API_KEY=...` / `export TGT_API_KEY=...` (Bash) or `set SRC_API_KEY=...` / `set TGT_API_KEY=...` (CMD).

### Retry after failed apply

Repeat the **same** arguments as your last fetch run (`--work-dir`, workspaces, `--source-root-id`, `--versions`, `--exclude-packages`, URLs, keys), append **`--retry-after-fail`**, and rerun. Fetch already finished is skipped while the snapshot in `--work-dir` is valid.

<table>
<tr>
<th align="left" width="50%">Bash</th>
<th align="left" width="50%">Windows CMD</th>
</tr>
<tr valign="top">
<td>

```bash
apihub-portal-package-copy \
  --work-dir ./run1 \
  --source-url https://source-ap.example \
  --source-api-key "${SRC_API_KEY:?}" \
  --source-workspace-id ws.source \
  --source-root-id '*' \
  --versions '*' \
  --exclude-packages ws.source.skip-this,ws.source.other.branch \
  --target-url https://target-ap.example \
  --target-api-key "${TGT_API_KEY:?}" \
  --target-workspace-id ws.target \
  --retry-after-fail
```

</td>
<td>

```bat
apihub-portal-package-copy.exe ^
  --work-dir .\run1 ^
  --source-url https://source-ap.example ^
  --source-api-key %SRC_API_KEY% ^
  --source-workspace-id ws.source ^
  --source-root-id "*" ^
  --versions "*" ^
  --exclude-packages ws.source.skip-this,ws.source.other.branch ^
  --target-url https://target-ap.example ^
  --target-api-key %TGT_API_KEY% ^
  --target-workspace-id ws.target ^
  --retry-after-fail
```

</td>
</tr>
</table>

### Named versions on one package subtree

Same URLs, keys, workspaces, and `--work-dir` pattern as **Full workspace** (no literal `*` in `--versions`); set **`--source-root-id`** to one package or group and list **`--versions`**. Omit **`--exclude-packages`** if you do not need it.

<table>
<tr>
<th align="left" width="50%">Bash</th>
<th align="left" width="50%">Windows CMD</th>
</tr>
<tr valign="top">
<td>

```bash
apihub-portal-package-copy \
  --work-dir ./run1 \
  --source-url https://source-ap.example \
  --source-api-key "${SRC_API_KEY:?}" \
  --source-workspace-id ws.source \
  --target-url https://target-ap.example \
  --target-api-key "${TGT_API_KEY:?}" \
  --target-workspace-id ws.target \
  --source-root-id ws.source.mygroup.onlypkg \
  --versions 1.0.0,1.2.0
```

</td>
<td>

```bat
apihub-portal-package-copy.exe ^
  --work-dir .\run1 ^
  --source-url https://source-ap.example ^
  --source-api-key %SRC_API_KEY% ^
  --source-workspace-id ws.source ^
  --target-url https://target-ap.example ^
  --target-api-key %TGT_API_KEY% ^
  --target-workspace-id ws.target ^
  --source-root-id ws.source.mygroup.onlypkg ^
  --versions 1.0.0,1.2.0
```

</td>
</tr>
</table>

## Backend endpoints used

List/describe: `GET /api/v2/packages/{id}`, `GET /api/v2/packages`, `GET /api/v3/packages/{id}/versions`.  
Snapshots: `GET /api/v2/packages/.../versions/{v}/sources`, `GET /api/v2/packages/.../versions/{v}/config`. Here **`{v}` is the full catalogue id** (including **`@revision`** when applicable).  
Target: `POST /api/v2/packages`, `POST /api/v2/packages/{id}/publish`, poll `GET /api/v2/packages/{id}/publish/{publishId}/status`.

Not used for source artifacts: **`/export`** flows.

### Roadmap / TODOs

- **Revision-aware copy (future):** optionally clone **all** revisions per logical version in **correct order** (not only the row returned by default `/versions`), including cross-revision metadata and any API extensions required for full fidelity.
