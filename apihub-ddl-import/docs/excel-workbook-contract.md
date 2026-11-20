# Excel workbook contract — developer requirements

## Purpose

Customer comment workbooks will vary over time: **sheet order**, **column order**, and **extra columns** may change. Tools that consume these files (`apihub-ddl-import`, NC BSS alignment scripts, and related helpers) must bind data **only by sheet name and column header name**, not by index, cell address, or fixed header layout.

Only **two** sheets drive DDL merge and publishing; all other sheets in the file are ignored.

---

## Glossary

| Term | Meaning |
|------|---------|
| **Required sheet** | Must exist in the workbook; parsing fails if missing. |
| **Required header** | The column title must appear in row 1 (any position); parsing fails if missing. |
| **Optional header** | Column may be absent; all rows treat the field as empty. |
| **Required row key** | For a non-empty data row, certain cells must be non-empty or the row is skipped (with a warning). |
| **Empty description** | Header present but cell empty → merge **warning**, no `COMMENT ON` for that entity (not fatal). |

---

## Workbook sheets

| Sheet name | Required | Processing |
|------------|----------|------------|
| List of Tables | **Yes** | Parsed |
| Tables Specifications | **Yes** | Parsed |
| Version | No | **Ignore** (do not read or validate content) |
| Thin Tables Specifications | No | **Ignore** |
| Any other sheet | No | **Ignore** |

**Rules**

- Sheet lookup: **case-insensitive**, trimmed exact name match.
- Position of sheets in the file does not matter.
- Missing either required sheet → **fatal** (e.g. `F_XLSX_HEADER`).
- Both required sheets with zero data rows after filtering → **fatal** (`F_XLSX_EMPTY`).
- One empty required sheet → **warning** (`SHEET_EMPTY`); existing behavior may apply.

---

## Header row rules (both data sheets)

- Row **1** is the header row.
- Match header labels **case-insensitively** after trim.
- **Unknown / extra columns** (e.g. `Deleted History`) → ignored for all logic.
- **Missing optional header** → field is empty for every data row.
- **Missing required header** → **fatal**; error message should list all missing header names.
- Column **order** and **physical position** (e.g. `Table Name` in B1 vs C1) must not matter.
- Rows may have variable width; trailing empty cells must not break parsing.

---

## List of Tables — columns

| Column header | Required header | Used by tool | Notes |
|---------------|-----------------|--------------|-------|
| Domain | **Yes** | APIHUB DDL groups, export **Group** column | Empty cell → no domain mapping for that table |
| Table Name | **Yes** | Match to DDL | Required row key (non-empty per processed row) |
| Table Description | **Yes** | `COMMENT ON TABLE` | Empty value → warning, no table comment |
| External Table | No | Dedup only; not written to DDL | Absent header → treat as empty |
| Deleted History | No | None | Ignore |
| (any other) | No | None | Ignore |

---

## Tables Specifications — columns

| Column header | Required header | Used by tool | Notes |
|---------------|-----------------|--------------|-------|
| Domain | **Yes** | Groups, export **Group** column | |
| Table Name | **Yes** | Match to DDL | Required row key |
| Column Name | **Yes** | Match to DDL | Required row key |
| Column Description | **Yes** | `COMMENT ON COLUMN` | Empty value → warning |
| Data Type | No | Duplicate-row detection only | **Recommended** in customer files; **not** written to enriched DDL |
| IsPK | No | PK mark validation vs DDL | Does **not** create or change `PRIMARY KEY` in DDL |
| Constraint | No | FK generation | `FK` / `PFK` (any case) may generate `ALTER TABLE ... FOREIGN KEY` when `*_id` resolves; absent → no FK from workbook |
| Deployment Release | No | Dedup only | |
| RDB | No | Dedup only | |
| ExternalDB | No | Dedup only | |
| (any other) | No | None | Ignore |

---

## Data rows

- Skip rows where **all mapped fields** are empty after trim.
- **List of Tables:** row has some data but **Table Name** is empty → **warning** and skip (default).
- **Tables Specifications:** row has some data but **Table Name** or **Column Name** is empty → **warning** and skip (default).

---

## Behavioral invariants (unchanged)

- **DDL is the source of truth** for table/column existence and **primary keys**; workbook `IsPK` never rewrites the DDL PK (mismatches → `PK_MISMATCH` warnings).
- **`Constraint=FK` / `PFK`** does not copy Excel text into DDL; FKs are generated only by the `<table_stem>_id` naming convention when unambiguous.
- Duplicate rows, conflicting descriptions, empty descriptions, and merge warning codes remain as documented in `apihub-ddl-import` README unless explicitly changed in a later story.

---

## Affected components (implementation scope)

| Component | Change |
|-----------|--------|
| `apihub-ddl-import/internal/xlsxin` | Name-based headers; required vs optional; ignore extra sheets |
| `apihub-ddl-import` README | Point to this contract instead of fixed column order |
| `scripts/align_nc_bss_xlsx.py` | Same header rules; resolve columns by name; row filter on **Table Name**, not column A |
| Tests | Reordered columns, extra columns, four-sheet workbook, missing optional headers |

Merge/render logic (`merge`, `ddl/render`) should stay unchanged if the reader still produces the same `CommentsDoc` shape.

---

## Acceptance criteria

1. Workbook with **four sheets** in any order, required headers **shuffled**, plus extra columns (`Deleted History`, etc.) → **successful parse** and equivalent merge outcome to a fixed-layout file with the same data.
2. Workbook **without** `Version` / `Thin Tables Specifications` → still valid when both required sheets exist.
3. Missing required header (e.g. no `Column Description` on specs) → **fatal** with a clear list of missing headers.
4. Missing optional headers (`Constraint`, `External Table`, `Data Type`, …) → **success**; behavior matches empty optional fields.
5. Existing golden test cases still pass after layout-tolerant parsing (update fixtures only if needed).
