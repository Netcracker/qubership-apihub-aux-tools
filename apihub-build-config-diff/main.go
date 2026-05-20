package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	configdiff "apihub-build-config-diff/internal/diff"
)

func main() {
	oldPath := flag.String("old", "", "Path to the baseline build config JSON")
	newPath := flag.String("new", "", "Path to the compared build config JSON")
	format := flag.String("format", "text", "Output format: text or json")
	only := flag.String("only", "", "Comma-separated categories to print: added,removed,changed")
	outputPath := flag.String("output", "", "Optional output file path; defaults to stdout")
	flag.Parse()

	if *oldPath == "" || *newPath == "" {
		flag.Usage()
		os.Exit(2)
	}

	filter, err := configdiff.ParseFilter(*only)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	result, err := loadAndCompare(*oldPath, *newPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	result = configdiff.ApplyFilter(result, filter)

	var out io.Writer = os.Stdout
	var file *os.File
	if *outputPath != "" {
		file, err = os.Create(*outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		out = file
	}

	switch strings.ToLower(strings.TrimSpace(*format)) {
	case "text":
		err = writeText(out, result, filter)
	case "json":
		err = writeJSON(out, result)
	default:
		fmt.Fprintf(os.Stderr, "unsupported --format value %q, expected text or json\n", *format)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadAndCompare(oldPath, newPath string) (configdiff.Result, error) {
	oldCfg, err := loadConfig(oldPath)
	if err != nil {
		return configdiff.Result{}, fmt.Errorf("load old config: %w", err)
	}
	newCfg, err := loadConfig(newPath)
	if err != nil {
		return configdiff.Result{}, fmt.Errorf("load new config: %w", err)
	}
	return configdiff.Compare(oldCfg.Refs, newCfg.Refs)
}

func loadConfig(path string) (configdiff.BuildConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return configdiff.BuildConfig{}, err
	}
	defer file.Close()
	return configdiff.ParseBuildConfig(file)
}

func writeJSON(w io.Writer, result configdiff.Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func writeText(w io.Writer, result configdiff.Result, filter configdiff.Filter) error {
	if _, err := fmt.Fprintf(w, "Refs diff: added %d, removed %d, changed %d\n", result.Summary.Added, result.Summary.Removed, result.Summary.Changed); err != nil {
		return err
	}

	if filter.Added {
		if err := writeRefs(w, "Added refIds", result.Added); err != nil {
			return err
		}
	}
	if filter.Removed {
		if err := writeRefs(w, "Removed refIds", result.Removed); err != nil {
			return err
		}
	}
	if filter.Changed {
		if err := writeChanged(w, result.Changed); err != nil {
			return err
		}
	}
	return nil
}

func writeRefs(w io.Writer, title string, refs []configdiff.Ref) error {
	if _, err := fmt.Fprintf(w, "\n%s:\n", title); err != nil {
		return err
	}
	if len(refs) == 0 {
		_, err := fmt.Fprintln(w, "- none")
		return err
	}
	for _, ref := range refs {
		if _, err := fmt.Fprintf(w, "- %s  version: %s\n", ref.RefID, ref.Version); err != nil {
			return err
		}
	}
	return nil
}

func writeChanged(w io.Writer, changed []configdiff.ChangedRef) error {
	if _, err := fmt.Fprintln(w, "\nChanged:"); err != nil {
		return err
	}
	if len(changed) == 0 {
		_, err := fmt.Fprintln(w, "- none")
		return err
	}

	for _, ref := range changed {
		if _, err := fmt.Fprintf(w, "- %s\n", ref.RefID); err != nil {
			return err
		}
		for _, change := range ref.Changes {
			if _, err := fmt.Fprintf(w, "  %s: %s -> %s\n", change.Field, change.Old, change.New); err != nil {
				return err
			}
		}
	}
	return nil
}
