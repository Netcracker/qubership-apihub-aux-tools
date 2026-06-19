# qubership-apihub-aux-tools

Auxiliary **command-line and utility tools** for [Qubership APIHUB](https://github.com/Netcracker/qubership-apihub): API catalog, published specifications, operation groups, and portal data. This repository is **not** part of the core product; it hosts scripts and small programs for **automation**, **migration**, **support**, and **one-off operations** that would be awkward to do only through the UI.

Typical uses:

- Automate repetitive portal tasks (grouping operations, bulk copy between environments).
- Reduce manual work for administrators and integrators.
- Provide workarounds when a dedicated product feature is not available.

Each tool lives in its **own subdirectory** with its own build instructions and flags. Read the linked README before running a tool in production.

## Requirements

- **Go 1.23** and **Node.js ≥20** to build tools from source (see [CI](#ci-and-releases)).
- Appropriate **APIHUB permissions** (personal access token or API key, depending on the tool) and network access to the target instance.

## Tools

| Directory | Language | Purpose |
|-----------|----------|---------|
| [`apihub-op-group-creator`](./apihub-op-group-creator) | Go | Create a REST **operation group** from operations filtered by a **custom tag**, then export the group spec (YAML/JSON) via the async export API. |
| [`apihub-portal-package-copy`](./apihub-portal-package-copy) | Go | **Copy published packages** (and optionally whole workspace subtrees) from one APIHUB instance to another using **original sources + publish config** REST APIs; supports resume, wildcards (`*` for versions or workspace scope), and **exclude lists**. |
| [`apihub-build-config-diff`](./apihub-build-config-diff) | Go | Compare build config JSON `refs` to quickly identify added, removed, and changed references. |
| [`apihub-api-diff`](./apihub-api-diff) | Node.js | CLI and local MCP server for categorized diff of OpenAPI, AsyncAPI, and GraphQL specifications. |

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

### apihub-build-config-diff

**What it does:** Compares the `refs` arrays from two APIHUB build config JSON files and reports added, removed, and changed refs. It can print a human-readable report or stable JSON output.

**When to use:** Release/config review where you need to quickly see new and deleted `refId` values, plus version or parent changes for refs that exist in both configs.

Details: **[apihub-build-config-diff/README.md](./apihub-build-config-diff/README.md)**.

Build:

```bash
cd apihub-build-config-diff && go build .
```

### apihub-api-diff

**What it does:** Compares two API description files (OpenAPI, AsyncAPI, GraphQL SDL, etc.) and produces a categorized changelog (breaking, risky, non-breaking) using the same rules as APIHUB api-processor. Can run as a CLI or as a local MCP server for IDE agents.

**When to use:** API migration impact review, CI checks on spec changes, or agent-assisted diff summaries in Cursor and other MCP clients.

Details, flags, and MCP setup: **[apihub-api-diff/README.md](./apihub-api-diff/README.md)**.

Build (requires npm access to `@netcracker/*` packages on GitHub Packages):

```bash
cd apihub-api-diff && npm install && npm run build
apihub-api-diff previous.yaml current.yaml --format md
apihub-api-diff mcp
```

## CI and releases

This repository is a **monorepo**: each tool is versioned and released independently.

### Continuous integration

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on pushes and pull requests to `main`. Only tools **touched in the change** are built and tested (via path filters). Changing the CI workflow itself runs all tools.

| Tool | Trigger paths |
|------|---------------|
| Go tools | `apihub-op-group-creator/**`, `apihub-portal-package-copy/**`, `apihub-build-config-diff/**` |
| apihub-api-diff | `apihub-api-diff/**` |

`apihub-api-diff` CI requires repository secret **`NPMRC`** with GitHub Packages auth for `@netcracker/*`:

```
@netcracker:registry=https://npm.pkg.github.com/
//npm.pkg.github.com/:_authToken=<github-token>
registry=https://registry.npmjs.org/
```

### Manual releases

Releases are **not** triggered by tags. Run the workflow for the tool you need from GitHub Actions (**Run workflow**), enter a semver (e.g. `v1.0.0` or `1.0.0`), and the workflow creates tag `<tool>/vX.Y.Z` plus a GitHub Release with binaries.

| Tool | Tag prefix | Run release |
|------|------------|-------------|
| apihub-op-group-creator | `apihub-op-group-creator/v*` | [Release apihub-op-group-creator](https://github.com/Netcracker/qubership-apihub-aux-tools/actions/workflows/release-apihub-op-group-creator.yml) |
| apihub-portal-package-copy | `apihub-portal-package-copy/v*` | [Release apihub-portal-package-copy](https://github.com/Netcracker/qubership-apihub-aux-tools/actions/workflows/release-apihub-portal-package-copy.yml) |
| apihub-build-config-diff | `apihub-build-config-diff/v*` | [Release apihub-build-config-diff](https://github.com/Netcracker/qubership-apihub-aux-tools/actions/workflows/release-apihub-build-config-diff.yml) |
| apihub-api-diff | `apihub-api-diff/v*` | [Release apihub-api-diff](https://github.com/Netcracker/qubership-apihub-aux-tools/actions/workflows/release-apihub-api-diff.yml) |

Go tools publish Linux and Windows binaries. `apihub-api-diff` also publishes a macOS binary.

### Dependency updates (Dependabot)

[`.github/dependabot.yml`](.github/dependabot.yml) tracks npm dependencies in `apihub-api-diff`, including `@netcracker/qubership-apihub-api-processor` from GitHub Packages. Dependabot opens PRs on a weekly schedule when new versions are published.

Requirements:

1. Dependabot is enabled for this repository (GitHub **Settings → Code security → Dependabot**).
2. Add Dependabot secret **`NPM_READ_TOKEN`** (**Settings → Secrets and variables → Dependabot**) — a GitHub PAT with `read:packages` scope for resolving `@netcracker/*` on `npm.pkg.github.com`.

## Adding a new tool

1. Add a **new top-level directory** (own `go.mod` or `package.json`, `README.md`, clear purpose).
2. Extend [`.github/workflows/ci.yml`](.github/workflows/ci.yml) with a path filter and build job; add [`.github/workflows/release-<tool>.yml`](.github/workflows/) with `workflow_dispatch`.
3. Register the tool in the **table**, the short section above, and the **Manual releases** table in this file.

## License

See [LICENSE](./LICENSE).
