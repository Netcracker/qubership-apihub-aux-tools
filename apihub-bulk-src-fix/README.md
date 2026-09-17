# APIHUB Bulk Sources Fix

`apihub-bulk-src-fix` helps to fix published sources of many APIHUB revisions at once.

It automates everything around the manual part of the work:

- downloads the sources archive of every revision from a list;
- extracts each archive into its own folder;
- saves `manifest.json` so the upload step knows which revisions to replace;
- packs the folders back and replaces the sources of exactly those revisions.

The fix of the documents itself is done by hand, between the two commands.

A typical reason to use it is a defect that affects many published documents at once, for
example incorrect YAML after a parser update, when the fix has to be applied to the original
sources of specific revisions instead of being republished as new versions.

## Authentication

Both commands send the `X-Personal-Access-Token` header. The upload command calls
`/api/v2/admin/...`, so the token needs administrator rights.

## Ids file

A plain text file, one revision per line, package id and version separated by a space:

```text
gbl.pkg1 2026.3@3
gbl.pkg2 2026.3@1
```

**The revision (`@N`) is required.** Without it the API serves the latest revision at the moment
of the request, and that one can change while the documents are being fixed by hand — the upload
step would then replace the sources of a revision that was never downloaded. A line without a
revision is rejected with the file name and the line number.

Lines that do not consist of exactly two words are skipped.

## Usage

### 1. Download

```shell
apihub-bulk-src-fix download -url https://apihub.example.com -pat <token> -ids ids.txt
```

Downloads and extracts the sources, then writes `manifest.json`. Revisions that failed to
download are reported in the log and are not written to the manifest.

### 2. Fix the documents by hand

Edit the files under `output/extracted/<packageId>-<version>@<revision>/`.

Do not run the download command again after this point: it overwrites the extracted folders
and the manual changes are lost.

### 3. Upload

```shell
apihub-bulk-src-fix upload -url https://apihub.example.com -pat <token>
```

Packs every folder listed in `manifest.json` and replaces the sources of the corresponding
revision. This is a destructive operation on published data — the original archives stay in
`output/downloaded/` and can be used to restore.

## Flags

`download`:

- `-url`: APIHUB base URL.
- `-pat`: personal access token.
- `-ids`: path to the ids file.
- `-output`: working directory; defaults to `./output`.

`upload`:

- `-url`: APIHUB base URL.
- `-pat`: personal access token.
- `-output`: working directory; defaults to `./output`.

## Working directory layout

```text
output/
  downloaded/                          original archives as they were published
    gbl.pkg1-2026.3@3.zip
  extracted/                           unpacked sources, this is what you edit
    gbl.pkg1-2026.3@3/
      petstore-sample-api.yaml
  fixed/                               archives built from the edited folders
    gbl.pkg1-2026.3@3.zip
  manifest.json                        revisions to replace on upload
```

`manifest.json`:

```json
[
  {
    "packageId": "gbl.pkg1",
    "version": "2026.3@3"
  }
]
```

Both commands process revisions in 10 parallel workers. A failed revision is logged and
skipped, the rest of the list keeps going.

## Development

```shell
go test ./...
go build ./...
```
