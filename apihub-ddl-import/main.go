// apihub-ddl-import merges a raw PostgreSQL DDL with a comments/PFK Excel
// workbook, publishes the enriched DDL to APIHUB, creates per-domain DDL table
// groups and downloads the enriched xlsx exports.
//
// See README.md for the pipeline, flags, config file and warning codes.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"apihub-ddl-import/internal/apihub"
	"apihub-ddl-import/internal/config"
	"apihub-ddl-import/internal/logx"
	"apihub-ddl-import/internal/pipeline"
)

const defaultConfigFile = "ddl-import.yaml"

func main() {
	os.Exit(run())
}

func run() int {
	fs := flag.NewFlagSet("apihub-ddl-import", flag.ExitOnError)

	configPath := fs.String("config", "", "Path to the YAML config file (default "+defaultConfigFile+" when present)")

	version := fs.String("version", "", "Version to publish (required)")
	previousVersion := fs.String("previous-version", "", "Previous published version, or 'none' for the first publish (required)")
	status := fs.String("status", "draft", "Version status: draft or release")

	apihubURL := fs.String("apihub-url", "", "APIHUB base URL")
	apihubKey := fs.String("apihub-api-key", "", "APIHUB api-key (or env "+config.EnvApihubAPIKey+")")
	packageID := fs.String("package-id", "", "APIHUB package id")

	ddlType := fs.String("ddl-source-type", "", "DDL source type: file or gitlab")
	ddlRepo := fs.String("ddl-repo", "", "DDL GitLab repository URL")
	ddlBranch := fs.String("ddl-branch", "", "DDL GitLab branch")
	ddlPath := fs.String("ddl-path", "", "DDL folder (or single .sql file) path")
	ddlToken := fs.String("ddl-token", "", "DDL GitLab token (or env "+config.EnvDdlGitlabToken+"/"+config.EnvGitlabToken+")")

	cmType := fs.String("comments-source-type", "", "Comments source type: file or gitlab")
	cmRepo := fs.String("comments-repo", "", "Comments GitLab repository URL")
	cmBranch := fs.String("comments-branch", "", "Comments GitLab branch")
	cmPath := fs.String("comments-path", "", "Comments .xlsx file path")
	cmToken := fs.String("comments-token", "", "Comments GitLab token (or env "+config.EnvCommentsGitlabToken+"/"+config.EnvGitlabToken+")")

	outputDir := fs.String("output-dir", "", "Output directory for artifacts (default ./ddl-import-out)")
	versionLabels := fs.String("version-labels", "", "Comma-separated version labels")
	publishTimeout := fs.Duration("publish-timeout", 0, "Publish poll timeout (default 15m)")

	dryRun := fs.Bool("dry-run", false, "Merge and report only — no APIHUB calls")
	strict := fs.Bool("strict", false, "Exit with code 3 before publishing when the merge has warnings")
	skipGroups := fs.Bool("skip-groups", false, "Skip the DDL table groups step")
	skipExports := fs.Bool("skip-exports", false, "Skip the xlsx exports step")
	skipEnrichment := fs.Bool("skip-enrichment", false, "Keep APIHUB's raw exports — do not add the Group/Analytics Severity custom columns")
	insecureTLS := fs.Bool("insecure-skip-tls-verify", false, "Skip TLS certificate verification (APIHUB and GitLab)")
	noColor := fs.Bool("no-color", false, "Disable colored output")
	debug := fs.Bool("debug", false, "Verbose HTTP diagnostics")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "apihub-ddl-import — merge DDL with an Excel comments workbook and publish to APIHUB\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  apihub-ddl-import --version 2026.1 --previous-version none [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Semi-static parameters (sources, tokens, APIHUB coordinates) usually live in %s;\nevery value can be overridden by a flag. Precedence: flag > env > config file.\n\nFlags:\n", defaultConfigFile)
		fs.PrintDefaults()
	}
	_ = fs.Parse(os.Args[1:])

	if *noColor {
		logx.DisableColor()
	}
	logx.SetDebug(*debug)
	if *debug {
		apihub.VerboseHTTP = true
		apihub.HTTPDebug = logx.Debugf
	}

	// Load config file (flag > env > file > default).
	explicit := *configPath != ""
	path := *configPath
	if path == "" {
		path = defaultConfigFile
	}
	cfg, err := config.Load(path, explicit)
	if err != nil {
		logx.Errorf("%v", err)
		return 2
	}
	cfg.ApplyEnv()

	overlay := func(dst *string, flagVal string) {
		if flagVal != "" {
			*dst = flagVal
		}
	}
	overlay(&cfg.Apihub.URL, *apihubURL)
	overlay(&cfg.Apihub.APIKey, *apihubKey)
	overlay(&cfg.Apihub.PackageID, *packageID)
	overlay(&cfg.DDLSource.Type, *ddlType)
	overlay(&cfg.DDLSource.Repo, *ddlRepo)
	overlay(&cfg.DDLSource.Branch, *ddlBranch)
	overlay(&cfg.DDLSource.Path, *ddlPath)
	overlay(&cfg.DDLSource.Token, *ddlToken)
	overlay(&cfg.CommentsSource.Type, *cmType)
	overlay(&cfg.CommentsSource.Repo, *cmRepo)
	overlay(&cfg.CommentsSource.Branch, *cmBranch)
	overlay(&cfg.CommentsSource.Path, *cmPath)
	overlay(&cfg.CommentsSource.Token, *cmToken)
	overlay(&cfg.Output.Dir, *outputDir)
	if *publishTimeout > 0 {
		cfg.Publish.Timeout = publishTimeout.String()
	}
	cfg.ApplyDefaults()

	labels := cfg.Publish.VersionLabels
	if *versionLabels != "" {
		labels = nil
		for _, l := range strings.Split(*versionLabels, ",") {
			if l = strings.TrimSpace(l); l != "" {
				labels = append(labels, l)
			}
		}
	}

	// Validation (usage errors → exit 2).
	usageErr := func(format string, args ...any) int {
		logx.Errorf(format, args...)
		fmt.Fprintln(os.Stderr)
		fs.Usage()
		return 2
	}
	if *version == "" {
		return usageErr("--version is required")
	}
	if *previousVersion == "" {
		return usageErr("--previous-version is required (use 'none' for the very first publish)")
	}
	st := strings.ToLower(strings.TrimSpace(*status))
	if st != "draft" && st != "release" {
		return usageErr("--status must be draft or release, got %q", *status)
	}
	if err := config.ValidateSource("ddlSource", cfg.DDLSource); err != nil {
		return usageErr("%v", err)
	}
	if err := config.ValidateSource("commentsSource", cfg.CommentsSource); err != nil {
		return usageErr("%v", err)
	}
	timeout, err := cfg.PublishTimeout()
	if err != nil {
		return usageErr("%v", err)
	}
	if !*dryRun {
		if cfg.Apihub.URL == "" || cfg.Apihub.PackageID == "" {
			return usageErr("apihub.url and apihub.packageId are required (config or --apihub-url/--package-id) unless --dry-run")
		}
		if cfg.Apihub.APIKey == "" {
			return usageErr("APIHUB api key is required (config apihub.apiKey, --apihub-api-key or env %s) unless --dry-run", config.EnvApihubAPIKey)
		}
	}

	return pipeline.Run(pipeline.Options{
		Cfg:             cfg,
		Version:         *version,
		PreviousVersion: strings.TrimSpace(*previousVersion),
		Status:          st,
		DryRun:          *dryRun,
		Strict:          *strict,
		SkipGroups:      *skipGroups,
		SkipExports:     *skipExports,
		SkipEnrichment:  *skipEnrichment,
		InsecureTLS:     *insecureTLS,
		PublishTimeout:  timeout,
		VersionLabels:   labels,
	})
}
