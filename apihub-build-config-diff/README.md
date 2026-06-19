# APIHUB Build Config Diff

`apihub-build-config-diff` compares the `refs` array from two APIHUB build config JSON files.

The utility is focused on quickly identifying:

- new `refId` values in the compared config;
- deleted `refId` values from the baseline config;
- existing `refId` values whose `version`, `parentRefId`, or `parentVersion` changed.

Refs are matched by exact `refId`. If the same `refId` has a different `version`, it is reported as changed, not as a removed and added ref.

## Usage

```shell
go run . --old 1.json --new 2.json
```

Use `--only added,removed` for a short report focused on new and deleted refs:

```shell
go run . --old 1.json --new 2.json --only added,removed
```

Write JSON output:

```shell
go run . --old 1.json --new 2.json --format json
```

Write the report to a file:

```shell
go run . --old old.json --new new.json --output refs-diff.txt
```

## Flags

- `--old`: required path to the baseline build config JSON.
- `--new`: required path to the compared build config JSON.
- `--format`: output format, either `text` or `json`; defaults to `text`.
- `--only`: optional comma-separated category filter: `added`, `removed`, `changed`.
- `--output`: optional output file path; defaults to stdout

## Development

```shell
go test ./...
go build ./...
```
