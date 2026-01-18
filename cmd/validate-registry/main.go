// Command validate-registry validates MCP registry JSON files against the schema.
//
// Usage:
//
//	validate-registry [options] [path...]
//
// If no paths are provided, validates appdata/registry by default.
//
// Options:
//
//	-strict     Treat warnings as errors
//	-json       Output results as JSON
//	-quiet      Only output errors
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mcp-scooter/scooter/internal/domain/registry"
)

var (
	strict = false
	asJSON = false
	quiet  = false
)

func main() {
	fs := flag.NewFlagSet("validate-registry", flag.ExitOnError)
	fs.BoolVar(&strict, "strict", false, "Treat warnings as errors")
	fs.BoolVar(&asJSON, "json", false, "Output results as JSON")
	fs.BoolVar(&quiet, "quiet", false, "Only output errors")

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	exitCode := run(fs.Args(), strict, asJSON, quiet)
	os.Exit(exitCode)
}

func run(paths []string, strict, asJSON, quiet bool) int {
	if len(paths) == 0 {
		// Default to appdata/registry
		paths = []string{"appdata/registry"}
	}

	exitCode := 0
	allResults := make(map[string]*registry.ValidationResult)

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s: %v\n", path, err)
			exitCode = 1
			continue
		}

		if info.IsDir() {
			results, err := registry.ValidateDirectory(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error validating directory %s: %v\n", path, err)
				exitCode = 1
				continue
			}
			for name, result := range results {
				fullPath := filepath.Join(path, name)
				allResults[fullPath] = result
			}
		} else {
			result, err := registry.ValidateFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error validating file %s: %v\n", path, err)
				exitCode = 1
				continue
			}
			allResults[path] = result
		}
	}

	// Output results
	if asJSON {
		outputJSON(allResults)
	} else {
		outputText(allResults, quiet, strict)
	}

	// Determine exit code
	for _, result := range allResults {
		if !result.Valid {
			exitCode = 1
		}
		if strict && len(result.Warnings) > 0 {
			exitCode = 1
		}
	}

	return exitCode
}

func outputJSON(results map[string]*registry.ValidationResult) {
	output := struct {
		Results map[string]*registry.ValidationResult `json:"results"`
		Summary struct {
			Total   int `json:"total"`
			Valid   int `json:"valid"`
			Invalid int `json:"invalid"`
		} `json:"summary"`
	}{
		Results: results,
	}

	for _, r := range results {
		output.Summary.Total++
		if r.Valid {
			output.Summary.Valid++
		} else {
			output.Summary.Invalid++
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(output)
}

func outputText(results map[string]*registry.ValidationResult, quiet, strict bool) {
	validCount := 0
	invalidCount := 0

	for path, result := range results {
		if result.Valid && len(result.Warnings) == 0 && quiet {
			validCount++
			continue
		}

		if result.Valid {
			validCount++
			if !quiet {
				fmt.Printf("✓ %s\n", path)
			}
		} else {
			invalidCount++
			fmt.Printf("✗ %s\n", path)
		}

		for _, err := range result.Errors {
			fmt.Printf("  ERROR: %s: %s\n", err.Field, err.Message)
		}

		if !quiet || strict {
			for _, warn := range result.Warnings {
				fmt.Printf("  WARN:  %s: %s\n", warn.Field, warn.Message)
			}
		}
	}

	if !quiet {
		fmt.Println()
		fmt.Printf("Summary: %d valid, %d invalid\n", validCount, invalidCount)
	}
}
