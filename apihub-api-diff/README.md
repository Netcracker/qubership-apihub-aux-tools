# APIHUB API Diff CLI and MCP

Standalone CLI and local MCP server for categorized API diffs powered by `@netcracker/qubership-apihub-api-processor`.

## Usage

```bash
apihub-api-diff previous.yaml current.yaml --format json
apihub-api-diff previous.yaml current.yaml --format md --output diff.md
apihub-api-diff previous.yaml current.yaml --format html --output diff.html
```

## Options

```text
apihub-api-diff <previous-file> <current-file> [options]

  -f, --format <json|md|html>       Output format. Default: json
  -o, --output <file>               Output file. Default: stdout for json/md, diff.html for html
      --output-dir <directory>      Directory for the default output file
      --fail-on <level>             Exit with code 2 on breaking, risky, any, or never. Default: never
      --include-values              Include raw before/after values in JSON output
      --title <text>                Report title
  -q, --quiet                       Suppress status messages
  -h, --help                        Show help
```

## Development

The repository resolves `@netcracker/*` packages from GitHub Packages. Configure an npm token with access to the organization before installing dependencies.

```bash
npm install
npm run typecheck
npm test
npm run build
```

Create a standalone binary for the current platform:

```bash
npm run build:binary
```

The binary is written to `dist/apihub-api-diff` or `dist/apihub-api-diff.exe`.

## MCP Server

The same binary can run as a local MCP server over stdio. On connect, the server returns **instructions** for the agent (when the client supports injecting them): when to diff API specs, preferred output formats, and how to summarize results for users.

```bash
apihub-api-diff mcp
```

Cursor MCP configuration example:

```json
{
  "mcpServers": {
    "apihub-api-diff": {
      "command": "apihub-api-diff",
      "args": [
        "mcp"
      ]
    }
  }
}
```

### Tool: `apihub_api_diff`

Compares two API description files on the **local machine** (paths must exist where the MCP server runs).

| Argument | Purpose |
| --- | --- |
| `previousPath` | Baseline spec (old version). |
| `currentPath` | Revised spec (new version). |
| `format` | `md` (default, best for LLM-readable summaries), `json` (machine-readable / counts / piping), or `html` (single-file browser report when the user wants that). |
| `includeValues` | `true` to attach concrete before/after values and richer Markdown detail; use for deep questions, expect larger output. |
| `title` | Optional heading (release, ticket, branch name) for traceability in the report.

**Agent-oriented usage:** Prefer Markdown + clear `title` for user-facing answers; use `json` with `includeValues: true` when the user needs exact delta evidence; reserve `html` for explicit “open in browser” requests.

## GitHub Actions

Add a repository secret named `NPMRC` with the full `.npmrc` content required to install `@netcracker/*` packages. Example:

```text
@netcracker:registry=https://npm.pkg.github.com/
//npm.pkg.github.com/:_authToken=<github-token>
registry=https://registry.npmjs.org/
```

Push a tag like `v0.1.0` to create a GitHub Release. The release workflow builds Linux, Windows, and macOS binaries and attaches them to the release.
