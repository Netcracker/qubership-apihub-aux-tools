// Command gen-comments generates the e2e-demo comments workbook
// (comments-and-pfk.xlsx) for a DDL dump. It reuses the tool's own DDL parser
// so table/column names match exactly, covers every table and column (so the
// merge report stays readable), and plants a small curated set of FK marks
// that exercise each rule of the merge engine once:
//
//	migrated_version_changes.build_id    FK  -> resolved to build(build_id)
//	operation_comparison.operation_id    FK  -> resolved via same-domain tie-break, composite-PK name match
//	operation_group_history.group_id     FK  -> resolved via same-domain tie-break, composite-PK first column
//	ephemeral_file.user_id               FK  -> FK_UNRESOLVED (no table matches stem "user")
//	ai_chat_message.chat_id              FK  -> FK_EXISTS (already declared in the DDL)
//	fts_operation_search_text.operation_id PFK -> FK_AMBIGUOUS (operation vs grouped_operation)
//
// Three columns carry COMMENT ON in the source DDL (published_version.status,
// published_data.media_type, published_version_revision_content.data_type), so
// their workbook descriptions produce the three expected COMMENT_EXISTS
// warnings.
//
// After writing the workbook the program re-reads it with the tool's xlsxin
// reader, merges it against the parsed DDL and fails unless the outcome is
// exactly the expected warning set — so a green run proves the workbook is
// clean by construction.
//
// Usage: go run ./e2e-demo/gen-comments -ddl <file-or-dir> -out <xlsx>
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"

	"apihub-ddl-import/internal/ddl"
	"apihub-ddl-import/internal/merge"
	"apihub-ddl-import/internal/model"
	"apihub-ddl-import/internal/xlsxin"
)

const (
	deployRelease = "2026.3"
	rdbName       = "PostgreSQL"
)

// domainByTable assigns every table to a domain; one APIHUB DDL group is
// created per domain by the import tool.
var domainByTable = map[string]string{
	"schema_migrations":        "Migrations",
	"stored_schema_migration":  "Migrations",
	"migration_run":            "Migrations",
	"migration_changes":        "Migrations",
	"migrated_version_changes": "Migrations",

	"package_group":         "Packages",
	"package_member_role":   "Packages",
	"package_service":       "Packages",
	"package_transition":    "Packages",
	"package_export_config": "Packages",
	"favorite_packages":     "Packages",

	"published_version":                  "Publishing",
	"published_data":                     "Publishing",
	"published_sources":                  "Publishing",
	"published_sources_archives":         "Publishing",
	"published_version_reference":        "Publishing",
	"published_version_revision_content": "Publishing",
	"published_version_validation":       "Publishing",
	"sources_update_tracking":            "Publishing",
	"shared_url_info":                    "Publishing",
	"export_result":                      "Publishing",
	"csv_dashboard_publication":          "Publishing",
	"version_internal_document":          "Publishing",
	"version_internal_document_data":     "Publishing",

	"operation":                   "Operations",
	"operation_data":              "Operations",
	"operation_group":             "Operations",
	"grouped_operation":           "Operations",
	"operation_group_history":     "Operations",
	"operation_group_publication": "Operations",
	"operation_group_template":    "Operations",
	"transformed_content_data":    "Operations",

	"version_comparison":                "Comparisons",
	"operation_comparison":              "Comparisons",
	"comparison_internal_document":      "Comparisons",
	"comparison_internal_document_data": "Comparisons",

	"build":                 "Builds",
	"build_depends":         "Builds",
	"build_result":          "Builds",
	"build_src":             "Builds",
	"builder_notifications": "Builds",

	"user_data":              "Identity & Access",
	"external_identity":      "Identity & Access",
	"user_avatar_data":       "Identity & Access",
	"apihub_api_keys":        "Identity & Access",
	"personal_access_tokens": "Identity & Access",
	"role":                   "Identity & Access",
	"system_role":            "Identity & Access",

	"activity_tracking":            "Analytics",
	"activity_tracking_transition": "Analytics",
	"business_metric":              "Analytics",
	"endpoint_calls":               "Analytics",
	"operation_open_count":         "Analytics",
	"published_version_open_count": "Analytics",
	"published_document_open_count": "Analytics",

	"fts_operation_search_text": "Search",
	"fts_ddl_search_text":       "Search",
	"fts_mcp_search_text":       "Search",

	"ai_chat":         "AI Assistant",
	"ai_chat_message": "AI Assistant",
	"ephemeral_file":  "AI Assistant",

	"ddl_tables":     "DDL Registry",
	"ddl_table_data": "DDL Registry",
	"ddl_comparison": "DDL Registry",

	"mcp_entities":    "MCP Registry",
	"mcp_entity_data": "MCP Registry",

	"build_cleanup_run":             "Housekeeping",
	"versions_cleanup_run":          "Housekeeping",
	"comparisons_cleanup_run":       "Housekeeping",
	"soft_deleted_data_cleanup_run": "Housekeeping",
	"unreferenced_data_cleanup_run": "Housekeeping",
	"locks":                         "Housekeeping",
}

var tableDesc = map[string]string{
	"schema_migrations":        "Applied schema migrations of the backend database and their dirty state.",
	"stored_schema_migration":  "Stored up/down SQL of applied schema migrations for integrity verification.",
	"migration_run":            "Data migration runs with stage, status, scope and retry bookkeeping.",
	"migration_changes":        "Aggregated change summary of one data migration.",
	"migrated_version_changes": "Changelog data rebuilt for package versions during a data migration.",

	"package_group":         "Package hierarchy tree: workspaces, groups, packages and dashboards.",
	"package_member_role":   "Package membership: per-user role assignments.",
	"package_service":       "Service-name bindings of packages within a workspace.",
	"package_transition":    "Package id renames: mapping from the old package id to the new one.",
	"package_export_config": "Per-package export configuration: allowed OpenAPI extensions.",
	"favorite_packages":     "Per-user favorite package bookmarks.",

	"published_version":                  "Published package versions and their revisions, with status and lineage.",
	"published_data":                     "Deduplicated published document payloads, addressed by checksum per package.",
	"published_sources":                  "Source archive metadata and build configuration of published versions.",
	"published_sources_archives":         "Deduplicated source archives, addressed by checksum.",
	"published_version_reference":        "Dashboard composition: referenced package versions with their parent chain.",
	"published_version_revision_content": "Documents (files) belonging to a published version revision.",
	"published_version_validation":       "Validation results (changelog, spectral, BWC) of published versions.",
	"sources_update_tracking":            "Audit of source archive replacements on published versions.",
	"shared_url_info":                    "Public share links to published documents.",
	"export_result":                      "Stored results of export jobs: generated archives with their configuration.",
	"csv_dashboard_publication":          "CSV dashboard publication jobs with status and report payload.",
	"version_internal_document":          "Internal documents attached to published version revisions.",
	"version_internal_document_data":     "Deduplicated payloads of version internal documents, addressed by hash.",

	"operation":                   "API operations extracted from the documents of published versions.",
	"operation_data":              "Deduplicated operation payloads, addressed by content hash.",
	"operation_group":             "Manual and autogenerated operation groups of published versions.",
	"grouped_operation":           "Membership of operations in operation groups.",
	"operation_group_history":     "Audit trail of operation group modifications.",
	"operation_group_publication": "Status of operation group publication jobs.",
	"operation_group_template":    "Export templates of operation groups, addressed by checksum.",
	"transformed_content_data":    "Cached transformed documents generated for operation groups.",

	"version_comparison":                "Version-to-version comparison headers with per-API-type summaries.",
	"operation_comparison":              "Operation-level diffs belonging to a version comparison.",
	"comparison_internal_document":      "Internal documents attached to version comparisons.",
	"comparison_internal_document_data": "Deduplicated payloads of comparison internal documents, addressed by hash.",

	"build":                 "Builder job queue: publish and export builds with status and priority.",
	"build_depends":         "Dependencies between builds: a build waits for the builds it depends on.",
	"build_result":          "Binary result archives of finished builds.",
	"build_src":             "Source archives and configuration of queued builds.",
	"builder_notifications": "Warnings and errors emitted by the builder, per build.",

	"user_data":              "User accounts: profile, credentials and the private package binding.",
	"external_identity":      "External identity provider bindings of user accounts.",
	"user_avatar_data":       "User avatar images with checksums.",
	"apihub_api_keys":        "Package-scoped API keys with role bindings.",
	"personal_access_tokens": "Personal access tokens of users, stored as hashes.",
	"role":                   "Role definitions: permission sets and role ranks.",
	"system_role":            "System-level role assignments of users.",

	"activity_tracking":             "Audit log of user and system events, per package.",
	"activity_tracking_transition":  "Progress tracking of package move and rename transitions.",
	"business_metric":               "Daily business metrics aggregated per user.",
	"endpoint_calls":                "Aggregated REST endpoint call counters.",
	"operation_open_count":          "Open counters of operations (popularity analytics).",
	"published_version_open_count":  "Open counters of published versions.",
	"published_document_open_count": "Open counters of published documents.",

	"fts_operation_search_text": "Full-text search vectors of operations.",
	"fts_ddl_search_text":       "Full-text search vectors of DDL entities.",
	"fts_mcp_search_text":       "Full-text search vectors of MCP entities.",

	"ai_chat":         "AI assistant chat sessions, per user.",
	"ai_chat_message": "Messages of AI assistant chat sessions.",
	"ephemeral_file":  "Short-lived uploaded files with an expiry deadline.",

	"ddl_tables":     "DDL entities (tables and views) of published versions.",
	"ddl_table_data": "Deduplicated DDL entity payloads, addressed by hash.",
	"ddl_comparison": "DDL entity diffs between compared versions.",

	"mcp_entities":    "MCP entities (tools, prompts and resources) of published versions.",
	"mcp_entity_data": "Deduplicated MCP entity payloads, addressed by hash.",

	"build_cleanup_run":             "History of build data cleanup runs.",
	"versions_cleanup_run":          "History of published version cleanup runs.",
	"comparisons_cleanup_run":       "History of version comparison cleanup runs.",
	"soft_deleted_data_cleanup_run": "History of soft-deleted data cleanup runs.",
	"unreferenced_data_cleanup_run": "History of unreferenced data cleanup runs.",
	"locks":                         "Distributed locks held by service instances, with expiry.",
}

// colOverride carries hand-written descriptions for columns whose meaning a
// name heuristic cannot capture. Key: "table.column".
var colOverride = map[string]string{
	"schema_migrations.version": "Migration number applied to the database.",
	"schema_migrations.dirty":   "True when the migration started but did not finish cleanly.",

	"stored_schema_migration.num":       "Migration number.",
	"stored_schema_migration.up_hash":   "Hash of the up migration SQL.",
	"stored_schema_migration.sql_up":    "Full SQL text of the up migration.",
	"stored_schema_migration.down_hash": "Hash of the down migration SQL.",
	"stored_schema_migration.sql_down":  "Full SQL text of the down migration.",

	"package_group.id":                       "Package id: dot-separated path in the hierarchy (workspace.group.package).",
	"package_group.kind":                     "Node kind: workspace, group, package or dashboard.",
	"package_group.alias":                    "Short alias of the package, unique within the parent.",
	"package_group.parent_id":                "Package id of the parent node.",
	"package_group.default_role":             "Default role granted to users on this package.",
	"package_group.default_released_version": "Version shown by default (latest release when empty).",
	"package_group.service_name":             "Service name bound to the package.",
	"package_group.release_version_pattern":  "Regular expression that release version names must match.",
	"package_group.exclude_from_search":      "True when the package is excluded from global search.",
	"package_group.rest_grouping_prefix":     "Path prefix used to auto-group REST operations.",

	"published_version.status":                      "Lifecycle status of the published version: draft, release or archived.",
	"published_version.previous_version":            "Version this one is compared against by default.",
	"published_version.previous_version_package_id": "Package of the previous version when it lives in another package.",
	"published_version.labels":                      "Free-form version labels.",

	"published_data.media_type": "HTTP media type of the stored document payload.",

	"published_version_revision_content.data_type": "Document type of the content item: OpenAPI, Swagger, Markdown and others.",
	"published_version_revision_content.slug":      "URL slug of the document within the version.",
	"published_version_revision_content.index":     "Ordering index of the document within the version.",
	"published_version_revision_content.file_id":   "Source file id of the document.",
	"published_version_revision_content.shareability_status": "Whether the document may be shared publicly.",
	"published_version_revision_content.api_kind":  "API kind declared for the document (bwc/no-bwc/experimental).",

	"operation.operation_id":     "Stable operation identifier within the version (method-path slug).",
	"operation.deprecated":       "True when the operation is marked deprecated.",
	"operation.kind":             "Operation API kind (bwc/no-bwc/experimental).",
	"operation.type":             "API type of the operation: rest or graphql.",
	"operation.deprecated_info":  "Human-readable deprecation note.",
	"operation.deprecated_items": "Deprecated schema items inside the operation (JSON).",
	"operation.previous_release_versions": "Release versions that contained this operation.",
	"operation.models":           "Named schema models referenced by the operation (JSON).",
	"operation.custom_tags":      "Custom x-tags of the operation (JSON).",
	"operation.api_audience":     "Audience of the API: external or internal.",

	"build.build_id":      "Unique identifier of the build job.",
	"build.status":        "Build status: none, running, complete, error.",
	"build.restart_count": "Number of times the build was restarted after failures.",
	"build.client_build":  "True when the build runs in the browser (client-side builder).",
	"build.builder_id":    "Identifier of the builder instance that took the job.",
	"build.priority":      "Queue priority; higher values are taken first.",

	"build_src.source": "Zipped build sources.",
	"build_src.config": "Build configuration (JSON).",

	"builder_notifications.severity": "Notification severity: error or warning.",
	"builder_notifications.file_id":  "Source file the notification refers to.",

	"business_metric.year":   "Calendar year of the metric value.",
	"business_metric.month":  "Calendar month of the metric value.",
	"business_metric.day":    "Calendar day of the metric value.",
	"business_metric.metric": "Metric name.",

	"endpoint_calls.path":    "Normalized endpoint path template.",
	"endpoint_calls.hash":    "Hash of the endpoint call options.",
	"endpoint_calls.options": "Captured call options (JSON).",
	"endpoint_calls.count":   "Number of recorded calls.",

	"user_data.private_package_id": "Id of the user's private package.",

	"external_identity.provider":    "Identity provider type.",
	"external_identity.external_id": "User id on the provider side.",
	"external_identity.internal_id": "APIHUB user id the identity maps to.",
	"external_identity.provider_id": "Identifier of the concrete provider instance.",

	"version_comparison.comparison_id": "Stable identifier of the comparison pair.",
	"version_comparison.operation_types": "Per-API-type change summaries (JSON).",
	"version_comparison.refs":          "Comparison ids of referenced package comparisons (dashboards).",
	"version_comparison.open_count":    "Number of times the comparison was opened.",
	"version_comparison.no_content":    "True when the comparison produced no differences payload.",
	"version_comparison.contract_types": "Per-contract-type change summaries (JSON).",

	"operation_comparison.data_hash":          "Hash of the current operation payload.",
	"operation_comparison.previous_data_hash": "Hash of the previous operation payload.",
	"operation_comparison.changes_summary":    "Aggregated change counters by severity (JSON).",
	"operation_comparison.changes":            "Detailed list of operation changes (JSON).",

	"operation_group.api_type":          "API type the group belongs to: rest or graphql.",
	"operation_group.group_name":        "Group name, unique per version and API type.",
	"operation_group.autogenerated":     "True for groups produced by the grouping prefix.",
	"operation_group.template_checksum": "Checksum of the export template attached to the group.",
	"operation_group.template_filename": "File name of the export template.",

	"operation_group_history.action":    "Performed action: create, update or delete.",
	"operation_group_history.automatic": "True when the change was made by the system.",

	"activity_tracking.e_type": "Event type of the audit record.",

	"activity_tracking_transition.tr_type":                 "Transition type: move or rename.",
	"activity_tracking_transition.from_id":                 "Package id before the transition.",
	"activity_tracking_transition.to_id":                   "Package id after the transition.",
	"activity_tracking_transition.progress_percent":        "Transition progress in percent.",
	"activity_tracking_transition.affected_objects":        "Number of objects affected by the transition.",
	"activity_tracking_transition.completed_serial_number": "Serial number assigned on completion.",

	"apihub_api_keys.api_key":     "Hashed API key value.",
	"apihub_api_keys.roles":       "Roles granted to the key.",
	"apihub_api_keys.created_for": "User the key was issued for.",

	"personal_access_tokens.token_hash": "Hash of the token value.",

	"role.role":        "Role name.",
	"role.rank":        "Role rank used to compare role strength.",
	"role.permissions": "Permissions granted by the role.",
	"role.read_only":   "True for built-in roles that cannot be edited.",

	"system_role.role": "System role name, e.g. System administrator.",

	"migration_run.package_ids":               "Packages the migration is limited to.",
	"migration_run.versions":                  "Versions the migration is limited to.",
	"migration_run.is_rebuild":                "True when builds are fully rebuilt.",
	"migration_run.is_rebuild_changelog_only": "True when only changelogs are rebuilt.",
	"migration_run.current_builder_version":   "Builder version used by the migration.",
	"migration_run.skip_validation":           "True when result validation is skipped.",
	"migration_run.sequence_number":           "Monotonic sequence number of the run.",
	"migration_run.post_check_result":         "Result of the post-migration consistency check (JSON).",
	"migration_run.stages_execution":          "Execution timings of migration stages (JSON).",

	"migrated_version_changes.build_id":       "Build that produced the rebuilt changelog.",
	"migrated_version_changes.migration_id":   "Data migration the rebuild belongs to.",
	"migrated_version_changes.changes":        "Rebuilt changelog payload (JSON).",
	"migrated_version_changes.unique_changes": "Distinct change fingerprints of the rebuilt changelog.",

	"locks.name":        "Lock name.",
	"locks.instance_id": "Service instance currently holding the lock.",
	"locks.version":     "Fencing counter incremented on every acquisition.",

	"ddl_tables.ddl_entity_id": "Stable identifier of the DDL entity within the version.",
	"ddl_tables.kind":          "Entity kind: table or view.",
	"ddl_tables.schema_name":   "Database schema the entity belongs to.",

	"ddl_comparison.ddl_entity_id":          "DDL entity id in the current version.",
	"ddl_comparison.previous_ddl_entity_id": "DDL entity id in the previous version.",

	"mcp_entities.mcp_entity_id": "Stable identifier of the MCP entity within the version.",
	"mcp_entities.kind":          "Entity kind: init, tool, prompt or resource.",
	"mcp_entities.mcp_endpoint":  "MCP endpoint the entity is served on.",

	"ai_chat.pinned":                     "True when the chat is pinned by the user.",
	"ai_chat.messages_count":             "Number of messages in the chat.",
	"ai_chat.compacted_up_to_created_at": "Messages up to this timestamp are folded into the compaction summary.",
	"ai_chat.compaction_summary":         "Summary replacing compacted messages.",
	"ai_chat.last_turn_tokens":           "Token count of the last model turn.",

	"ai_chat_message.role":              "Author role: user or assistant.",
	"ai_chat_message.client_message_id": "Client-generated id used for idempotent retries.",
	"ai_chat_message.tool_invocations":  "Tool calls made while producing the message (JSON).",

	"ephemeral_file.storage_path": "Path of the payload in the object storage.",

	"shared_url_info.shared_id": "Public identifier used in the share link.",

	"transformed_content_data.documents_info": "Per-document metadata of the transformed payload (JSON).",
	"transformed_content_data.build_type":     "Transformation type: documentGroup or reducedSourceSpecifications.",
	"transformed_content_data.format":         "Output format of the transformed payload.",

	"published_sources.archive_checksum": "Checksum referencing the deduplicated source archive.",

	"sources_update_tracking.old_checksum": "Checksum of the replaced source archive.",
	"sources_update_tracking.new_checksum": "Checksum of the new source archive.",

	"package_service.workspace_id": "Workspace the service name is registered in.",

	"package_transition.old_package_id": "Package id before the move or rename.",
	"package_transition.new_package_id": "Package id after the move or rename.",

	"package_export_config.allowed_oas_extensions": "OpenAPI x-extensions preserved in exports.",

	"export_result.export_id": "Identifier of the export job.",
	"export_result.config":    "Export configuration (JSON).",
	"export_result.filename":  "File name offered for download.",

	"csv_dashboard_publication.publish_id": "Identifier of the publication job.",
	"csv_dashboard_publication.csv_report": "Generated CSV report payload.",

	"operation_group_publication.publish_id": "Identifier of the publication job.",

	"version_internal_document.document_id": "Identifier of the internal document within the revision.",
	"comparison_internal_document.document_id": "Identifier of the internal document within the comparison.",

	"fts_operation_search_text.search_data_hash": "Hash of the source text the vector was built from.",
	"fts_operation_search_text.data_vector":      "tsvector with the searchable operation text.",
	"fts_ddl_search_text.search_data_hash":       "Hash of the source text the vector was built from.",
	"fts_ddl_search_text.data_vector":            "tsvector with the searchable DDL entity text.",
	"fts_mcp_search_text.search_data_hash":       "Hash of the source text the vector was built from.",
	"fts_mcp_search_text.data_vector":            "tsvector with the searchable MCP entity text.",

	"grouped_operation.group_id": "Operation group the operation belongs to.",

	"build_depends.depend_id": "Build this build depends on.",

	"versions_cleanup_run.delete_before":    "Versions deleted before this timestamp.",
	"comparisons_cleanup_run.delete_before": "Comparisons deleted before this timestamp.",
	"soft_deleted_data_cleanup_run.delete_before": "Soft-deleted records purged before this timestamp.",
}

// fkMark places the curated Constraint cell values, keyed "table.column".
var fkMark = map[string]string{
	"migrated_version_changes.build_id":      "FK",
	"ephemeral_file.user_id":                 "FK",
	"operation_comparison.operation_id":      "FK",
	"operation_group_history.group_id":       "FK",
	"ai_chat_message.chat_id":                "FK",
	"fts_operation_search_text.operation_id": "PFK",
}

// expected outcome of the self-verification merge.
var expectedWarnings = map[string]int{
	model.WCommentExists: 3,
	model.WFkExists:      1,
	model.WFkUnresolved:  1,
	model.WFkAmbiguous:   1,
}

func main() {
	ddlPath := flag.String("ddl", "", "DDL source: a .sql file or a directory of .sql files")
	out := flag.String("out", "comments-and-pfk.xlsx", "output workbook path")
	flag.Parse()
	if *ddlPath == "" {
		fail("missing -ddl")
	}

	files, err := loadFiles(*ddlPath)
	if err != nil {
		fail("load DDL: %v", err)
	}
	m, err := ddl.ParseFiles(files)
	if err != nil {
		fail("parse DDL: %v", err)
	}

	tables := canonicalTables(m)
	checkCoverage(tables)

	if err := writeWorkbook(*out, tables); err != nil {
		fail("write workbook: %v", err)
	}
	fmt.Printf("workbook written: %s (%d tables)\n\n", *out, len(tables))

	verify(*out, m, tables)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gen-comments: "+format+"\n", args...)
	os.Exit(1)
}

func loadFiles(p string) ([]model.File, error) {
	st, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	var paths []string
	if st.IsDir() {
		entries, err := os.ReadDir(p)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".sql") {
				paths = append(paths, filepath.Join(p, e.Name()))
			}
		}
		sort.Strings(paths)
	} else {
		paths = []string{p}
	}
	var files []model.File
	for _, fp := range paths {
		data, err := os.ReadFile(fp)
		if err != nil {
			return nil, err
		}
		files = append(files, model.File{RelPath: filepath.Base(fp), Data: data})
	}
	return files, nil
}

// canonicalTables drops duplicate definitions the same way the merge does
// (first definition wins).
func canonicalTables(m *model.DDLModel) []*model.DDLTable {
	seen := map[string]bool{}
	var out []*model.DDLTable
	for _, t := range m.Tables {
		k := model.NormKey(t.Name)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, t)
	}
	return out
}

// checkCoverage refuses to run with an incomplete description/domain map, so
// the workbook always covers the whole DDL.
func checkCoverage(tables []*model.DDLTable) {
	var missing []string
	for _, t := range tables {
		if tableDesc[t.Name] == "" {
			missing = append(missing, t.Name+" (description)")
		}
		if domainByTable[t.Name] == "" {
			missing = append(missing, t.Name+" (domain)")
		}
	}
	if len(missing) > 0 {
		fail("tables not covered by the generator maps:\n  %s", strings.Join(missing, "\n  "))
	}
	for name := range tableDesc {
		if !hasTable(tables, name) {
			fail("tableDesc entry %q matches no DDL table", name)
		}
	}
	for key := range colOverride {
		tbl, col, ok := strings.Cut(key, ".")
		if !ok || !hasColumn(tables, tbl, col) {
			fail("colOverride entry %q matches no DDL column", key)
		}
	}
	for key := range fkMark {
		tbl, col, ok := strings.Cut(key, ".")
		if !ok || !hasColumn(tables, tbl, col) {
			fail("fkMark entry %q matches no DDL column", key)
		}
	}
}

func hasTable(tables []*model.DDLTable, name string) bool {
	for _, t := range tables {
		if t.Name == name {
			return true
		}
	}
	return false
}

func hasColumn(tables []*model.DDLTable, table, col string) bool {
	for _, t := range tables {
		if t.Name == table {
			return t.Column(col) != nil
		}
	}
	return false
}

func columnDesc(t *model.DDLTable, c *model.DDLColumn) string {
	if d := colOverride[t.Name+"."+c.Name]; d != "" {
		return d
	}
	entity := strings.ReplaceAll(t.Name, "_", " ")
	switch c.Name {
	case "id":
		return "Primary identifier of the " + entity + " record."
	case "package_id":
		return "Identifier of the package."
	case "previous_package_id":
		return "Package of the previous (compared) version."
	case "version":
		return "Package version label."
	case "previous_version":
		return "Previous (compared) version label."
	case "revision":
		return "Revision number of the version."
	case "previous_revision":
		return "Revision number of the previous version."
	case "user_id":
		return "Identifier of the user."
	case "created_at":
		return "Timestamp when the record was created."
	case "created_by":
		return "User who created the record."
	case "updated_at":
		return "Timestamp of the last update."
	case "updated_by":
		return "User who performed the last update."
	case "deleted_at":
		return "Timestamp of soft deletion; empty while the record is active."
	case "deleted_by":
		return "User who deleted the record."
	case "started_at":
		return "Timestamp when the run started."
	case "started_by":
		return "User who started the run."
	case "finished_at":
		return "Timestamp when the run finished."
	case "published_at":
		return "Timestamp when the version was published."
	case "scheduled_at":
		return "Timestamp when the run was scheduled."
	case "acquired_at":
		return "Timestamp when the lock was acquired."
	case "expires_at":
		return "Expiry deadline."
	case "last_active":
		return "Timestamp of the last activity."
	case "last_message_at":
		return "Timestamp of the last message."
	case "date":
		return "Timestamp of the event."
	case "status":
		return "Current status of the " + entity + " record."
	case "details":
		return "Human-readable details."
	case "error_details":
		return "Error details of a failed run."
	case "metadata":
		return "Auxiliary metadata (JSON)."
	case "data":
		if c.Type == "jsonb" || c.Type == "json" {
			return "Payload (JSON)."
		}
		return "Binary payload."
	case "checksum":
		return "Content checksum."
	case "data_hash":
		return "Hash of the payload, used for deduplication."
	case "hash":
		return "Hash of the payload, used for deduplication."
	case "name":
		return "Display name."
	case "title":
		return "Display title."
	case "description":
		return "Human-readable description."
	case "filename":
		return "File name."
	case "file_id":
		return "Source file identifier."
	case "slug":
		return "URL slug."
	case "open_count":
		return "Number of times the item was opened."
	case "instance_id":
		return "Identifier of the service instance that owns the record."
	case "run_id":
		return "Identifier of the cleanup run."
	case "migration_id":
		return "Identifier of the data migration."
	case "comparison_id":
		return "Identifier of the version comparison."
	case "build_id":
		return "Identifier of the build job."
	case "document_id":
		return "Identifier of the document."
	case "operation_id":
		return "Identifier of the operation."
	case "group_id":
		return "Identifier of the operation group."
	case "chat_id":
		return "Identifier of the chat session."
	case "version_internal_document_id":
		return "Internal document the entity payload is stored in."
	case "comparison_internal_document_id":
		return "Internal document the comparison payload is stored in."
	case "avatar_url":
		return "URL of the user avatar."
	case "email":
		return "E-mail address of the user."
	case "password":
		return "Password hash of local accounts."
	case "avatar":
		return "Avatar image payload."
	case "size_bytes":
		return "Payload size in bytes."
	case "mime_type":
		return "MIME type of the payload."
	case "deleted_rows", "deleted_items":
		return "Number of items deleted by the run."
	case "delete_before":
		return "Items older than this timestamp were deleted."
	case "retry_count":
		return "Number of retries performed."
	case "stage":
		return "Current stage of the run."
	}
	if s, ok := strings.CutSuffix(c.Name, "_count"); ok {
		return "Number of " + strings.ReplaceAll(s, "_", " ") + " items."
	}
	if s, ok := strings.CutSuffix(c.Name, "_id"); ok {
		return "Identifier of the referenced " + strings.ReplaceAll(s, "_", " ") + "."
	}
	// Fallback: humanized column name.
	h := strings.ReplaceAll(c.Name, "_", " ")
	return strings.ToUpper(h[:1]) + h[1:] + "."
}

func isPKCol(t *model.DDLTable, col string) bool {
	for _, c := range t.PKCols {
		if c == col {
			return true
		}
	}
	return false
}

func writeWorkbook(path string, tables []*model.DDLTable) error {
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetDefaultFont("Arial"); err != nil {
		return err
	}
	if err := f.SetSheetName("Sheet1", xlsxin.SheetTables); err != nil {
		return err
	}
	if _, err := f.NewSheet(xlsxin.SheetSpecs); err != nil {
		return err
	}

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"DDEBF7"}, Pattern: 1},
	})
	if err != nil {
		return err
	}

	// ---- List of Tables
	if err := f.SetSheetRow(xlsxin.SheetTables, "A1",
		&[]any{"Domain", "Table Name", "Table Description", "External Table"}); err != nil {
		return err
	}
	row := 2
	for _, t := range tables {
		err := f.SetSheetRow(xlsxin.SheetTables, fmt.Sprintf("A%d", row),
			&[]any{domainByTable[t.Name], t.Name, tableDesc[t.Name], "N"})
		if err != nil {
			return err
		}
		row++
	}
	if err := f.SetCellStyle(xlsxin.SheetTables, "A1", "D1", headerStyle); err != nil {
		return err
	}
	for col, w := range map[string]float64{"A": 18, "B": 36, "C": 84, "D": 14} {
		if err := f.SetColWidth(xlsxin.SheetTables, col, col, w); err != nil {
			return err
		}
	}

	// ---- Tables Specifications
	if err := f.SetSheetRow(xlsxin.SheetSpecs, "A1",
		&[]any{"Domain", "Table Name", "Column Name", "Data Type", "IsPK",
			"Constraint", "Column Description", "Deployment Release", "RDB", "ExternalDB"}); err != nil {
		return err
	}
	row = 2
	for _, t := range tables {
		for i := range t.Columns {
			c := &t.Columns[i]
			isPK := ""
			cons := ""
			if isPKCol(t, c.Name) {
				isPK = "Y"
				cons = "PK"
			}
			if mark := fkMark[t.Name+"."+c.Name]; mark != "" {
				cons = mark
			}
			err := f.SetSheetRow(xlsxin.SheetSpecs, fmt.Sprintf("A%d", row),
				&[]any{domainByTable[t.Name], t.Name, c.Name, c.Type, isPK, cons,
					columnDesc(t, c), deployRelease, rdbName, ""})
			if err != nil {
				return err
			}
			row++
		}
	}
	if err := f.SetCellStyle(xlsxin.SheetSpecs, "A1", "J1", headerStyle); err != nil {
		return err
	}
	for col, w := range map[string]float64{"A": 18, "B": 36, "C": 34, "D": 24, "E": 7,
		"F": 11, "G": 84, "H": 19, "I": 12, "J": 12} {
		if err := f.SetColWidth(xlsxin.SheetSpecs, col, col, w); err != nil {
			return err
		}
	}

	for _, sheet := range []string{xlsxin.SheetTables, xlsxin.SheetSpecs} {
		err := f.SetPanes(sheet, &excelize.Panes{
			Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft",
		})
		if err != nil {
			return err
		}
	}
	return f.SaveAs(path)
}

// verify re-reads the workbook with the tool's own reader and merges it against
// the DDL, then asserts the exact expected outcome.
func verify(path string, m *model.DDLModel, tables []*model.DDLTable) {
	data, err := os.ReadFile(path)
	if err != nil {
		fail("re-read workbook: %v", err)
	}
	doc, err := xlsxin.Read(data)
	if err != nil {
		fail("workbook does not pass the tool's reader: %v", err)
	}
	res := merge.Merge(m, doc)
	st := res.Stats

	totalCols := 0
	for _, t := range tables {
		totalCols += len(t.Columns)
	}
	fmt.Printf("merge self-check: %d/%d tables matched, %d/%d columns matched\n",
		st.TablesMatched, len(tables), st.ColumnsMatched, totalCols)
	fmt.Printf("comments: %d table + %d column; FKs: %d generated, %d unresolved, %d ambiguous\n",
		st.TableCommentsGenerated, st.ColumnCommentsGenerated,
		st.FKsGenerated, st.FKsUnresolved, st.FKsAmbiguous)
	fmt.Printf("domains: %d (%s)\n\n", len(res.Domains), strings.Join(res.Domains, ", "))

	for _, fk := range res.FKs {
		note := ""
		if fk.Note != "" {
			note = " — " + fk.Note
		}
		fmt.Printf("FK %s.%s -> %s(%s)%s\n", fk.Table, fk.Column, fk.TargetTable, fk.TargetColumn, note)
	}
	fmt.Println()
	byCode := map[string]int{}
	for _, w := range res.Warnings {
		byCode[w.Code]++
		fmt.Printf("warning %-16s %s\n", w.Code, w.Msg)
	}

	ok := true
	if st.TablesMatched != len(tables) || st.ColumnsMatched != totalCols {
		ok = false
		fmt.Printf("FAIL: incomplete coverage\n")
	}
	for code, want := range expectedWarnings {
		if byCode[code] != want {
			ok = false
			fmt.Printf("FAIL: warning %s: got %d, want %d\n", code, byCode[code], want)
		}
	}
	for code, got := range byCode {
		if expectedWarnings[code] == 0 {
			ok = false
			fmt.Printf("FAIL: unexpected warning %s ×%d\n", code, got)
		}
	}
	if st.FKsGenerated != 3 {
		ok = false
		fmt.Printf("FAIL: FKs generated: got %d, want 3\n", st.FKsGenerated)
	}
	if !ok {
		os.Exit(1)
	}
	fmt.Println("\nself-check PASSED: workbook is clean against the DDL")
}
