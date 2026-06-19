# Output Schema

The JSON report is the canonical output. Markdown and HTML reports are rendered from the same model.

Top-level fields:

- `schemaVersion` - CLI output contract version.
- `tool` - CLI name and version.
- `engine` - api-processor engine metadata.
- `inputs` - previous and current file metadata.
- `summary` - aggregated counts by severity, action, API type, and operation count.
- `comparisons` - raw comparison metadata from api-processor.
- `changes` - operation-level changes with categorized diff messages.
- `notifications` - parser/build notifications reported by api-processor.

Raw before/after values are omitted by default. Pass `--include-values` to include JSON-safe `previousValue` and `currentValue` fields in each diff message.
