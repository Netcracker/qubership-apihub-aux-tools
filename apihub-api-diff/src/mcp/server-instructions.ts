/**
 * Shown to MCP clients in `initialize` (when supported) so agents know how to use this server effectively.
 * Keep in English for broad model compatibility.
 */
export const SERVER_INSTRUCTIONS = `You have access to APIHUB API Diff: a specialized diff engine for API descriptions (OpenAPI/Swagger YAML or JSON, AsyncAPI, GraphQL SDL, etc.). It compares a "before" document to an "after" document and returns a categorized changelog (severity, action, scope) using the same rules as APIHUB's api-processor.

## When to use this tool
- The user asks what changed between two API specs, whether a change is breaking, or for a migration/porting impact summary.
- You need a structured diff richer than plain text diff of files.
- CI or review context: classify changes (breaking vs non-breaking) across operations and schemas.

## When not to use it
- Files are not API descriptions, or comparison is not "previous version vs current version" of the same API surface.
- You only need git line diff (use git/read tools instead).

## How to call the tool effectively
1. Paths must be absolute or resolvable on the machine running the MCP server (the IDE/host filesystem).
2. Default report format is Markdown (md): best balance for you to read and for turning into a clear, structured answer. Prefer md for user-facing summaries with headings and tables.
3. Use JSON format when you need exact machine-readable structure, counts, severity buckets, or downstream filtering. Combine JSON with includeValues: true when the user asks for concrete before/after examples (request/response schemas, parameters, examples).
4. Set includeValues: true for the deepest detail (raw before/after values appear in JSON; Markdown also adds an extra detail section when values are present). Expect larger output; summarize for the user.
5. Use HTML only when the user explicitly wants a single-file interactive report to open in a browser. It is awkward to quote verbatim in chat.
6. Set title when the user or task names a release, branch, or ticket (e.g. "v2.3 → v2.4 — PLT-4410") so headings stay traceable.

## How to present results to the user
- Lead with a short verdict (any breaking changes? how many operations affected?).
- Then group by severity or by operation; use the Markdown sections as your outline.
- Quote only the most relevant lines; avoid dumping huge JSON unless asked.
- If notifications appear in the report, surface them—they are parser or resolution warnings.

## Pairing with other tools
- Read or obtain the two file paths first; then call apihub_api_diff once per pair.`

