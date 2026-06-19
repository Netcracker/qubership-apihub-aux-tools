# Usage

The CLI compares two API documents and emits a categorized diff report.

```bash
apihub-api-diff old.yaml new.yaml
```

By default the report is JSON and is written to stdout. Use `--format md` or `--format html` for human-readable reports.

For CI, use `--fail-on`:

```bash
apihub-api-diff old.yaml new.yaml --fail-on breaking
```

Exit codes:

- `0` means the command completed and the selected threshold did not match.
- `1` means the command failed.
- `2` means the diff matched the `--fail-on` threshold.

## MCP Mode

Run the same binary as a local MCP server over stdio:

```bash
apihub-api-diff mcp
```

The server advertises **`apihub_api_diff`**. On initialization it also sends **server instructions** (if your client surfaces them to the model) explaining when to diff API specs and how to choose formats:

- **`md` (default)** — structured, LLM-friendly report for turning into a clear user answer.
- **`json`** — exact structure, severity buckets; add **`includeValues: true`** for raw before/after field values.
- **`html`** — only when the user wants a standalone file to open in a browser.

Paths must be valid on the host where the MCP process runs. For the most detailed Markdown, use `includeValues: true` (adds pointer/value detail sections where available).
