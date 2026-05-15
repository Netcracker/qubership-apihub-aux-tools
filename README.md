# qubership-apihub-aux-tools

Auxiliary **command-line and utility tools** for [Qubership APIHUB](https://github.com/Netcracker/qubership-apihub): API catalog, published specifications, operation groups, and portal data. This repository is **not** part of the core product; it hosts scripts and small programs for **automation**, **migration**, **support**, and **one-off operations** that would be awkward to do only through the UI.

Typical uses:

- Automate repetitive portal tasks (grouping operations, bulk copy between environments).
- Reduce manual work for administrators and integrators.
- Provide workarounds when a dedicated product feature is not available.

Each tool lives in its **own subdirectory** with its own `go.mod`, build instructions, and flags. Read the linked README before running a tool in production.

## Requirements

- **Go 1.23** (see [`.github/workflows/go.yml`](.github/workflows/go.yml)) to build from source.
- Appropriate **APIHUB permissions** (personal access token or API key, depending on the tool) and network access to the target instance.

## Tools

| Directory | Language | Purpose |
|-----------|----------|---------|
| [`apihub-op-group-creator`](./apihub-op-group-creator) | Go | Create a REST **operation group** from operations filtered by a **custom tag**, then export the group spec (YAML/JSON) via the async export API. |
| [`apihub-portal-package-copy`](./apihub-portal-package-copy) | Go | **Copy published packages** (and optionally whole workspace subtrees) from one APIHUB instance to another using **original sources + publish config** REST APIs; supports resume, wildcards (`*` for versions or workspace scope), and **exclude lists**. |

### apihub-op-group-creator

**What it does:** Lists operations for a package version (REST), keeps those whose **custom tags** match `-x-key` / `-x-value`, creates or recreates an operation group, assigns those operations to the group, and waits for **export** of the aggregated document.

**Authentication:** **`X-Personal-Access-Token`** (personal access token), not package API keys.

**When to use:** You need a **reduced OpenAPI-like document** built from a tagged subset of operations (for review, downstream tools, or sharing outside the group UI).

Details, flags, and examples: **[apihub-op-group-creator/README.md](./apihub-op-group-creator/README.md)**.

Build:

```bash
cd apihub-op-group-creator && go build .
```

### apihub-portal-package-copy

**What it does:** Pulls **`/versions/{version}/sources`** (zip) and **`/versions/{version}/config`** from a **source** APIHUB, writes them under `--work-dir`, logs a **plan**, then ensures the **target** package tree exists and **publishes** versions in order so **`previousVersion`** chains and in-scope **`refs`** keep working after ID remapping.

**Authentication:** **`api-key`** header (Apihub API keys). Workspace-wide listing uses `GET /api/v2/packages` without `{packageId}` in the path; that path often requires an API key scoped to **`*`** (see tool README).

**When to use:** **Environment migration**, **staging → production** sync of catalog content, or **bulk duplication** where you control both instances and need faithful sources (not `/export`-style artifacts).

Features (see tool README):

- **`--retry-after-fail`** — resume publish from checkpoint without re-fetching.
- **`--force-refresh-fetch`** — re-download artifacts.
- **`--versions *`** / **`--source-root-id *`** / **`--exclude-packages`** — scope control.

Details: **[apihub-portal-package-copy/README.md](./apihub-portal-package-copy/README.md)**.

Build:

```bash
cd apihub-portal-package-copy && go build .
```

## CI and releases

- **Continuous integration:** [`.github/workflows/go.yml`](.github/workflows/go.yml) — `go build` and `go test` for each tool on pushes and pull requests to `main`.
- **Release binaries:** [`.github/workflows/release.yml`](.github/workflows/release.yml) — on **tag** push, builds Linux and Windows artifacts for both tools and attaches them to a GitHub release.

## Adding a new tool

1. Add a **new top-level directory** (own `go.mod`, `README.md`, clear purpose).
2. Extend **`.github/workflows/go.yml`** and **`release.yml`** with build, test, and upload steps (follow existing tools).
3. Register the tool in the **table** and the short section above in this file.

## License

See [LICENSE](./LICENSE).
