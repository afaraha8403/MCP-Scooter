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
	strict = flag.Bool("strict", false, "Treat warnings as errors")
	asJSON = flag.Bool("json", false, "Output results as JSON")
	quiet  = flag.Bool("quiet", false, "Only output errors")
)

func main() {
	flag.Parse()

	paths := flag.Args()
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
	if *asJSON {
		outputJSON(allResults)
	} else {
		outputText(allResults)
	}

	// Determine exit code
	for _, result := range allResults {
		if !result.Valid {
			exitCode = 1
		}
		if *strict && len(result.Warnings) > 0 {
			exitCode = 1
		}
	}

	os.Exit(exitCode)
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

func outputText(results map[string]*registry.ValidationResult) {
	validCount := 0
	invalidCount := 0

	for path, result := range results {
		if result.Valid && len(result.Warnings) == 0 && *quiet {
			validCount++
			continue
		}

		if result.Valid {
			validCount++
			if !*quiet {
				fmt.Printf("✓ %s\n", path)
			}
		} else {
			invalidCount++
			fmt.Printf("✗ %s\n", path)
		}

		for _, err := range result.Errors {
			fmt.Printf("  ERROR: %s: %s\n", err.Field, err.Message)
		}

		if !*quiet || *strict {
			for _, warn := range result.Warnings {
				fmt.Printf("  WARN:  %s: %s\n", warn.Field, warn.Message)
			}
		}
	}

	if !*quiet {
		fmt.Println()
		fmt.Printf("Summary: %d valid, %d invalid\n", validCount, invalidCount)
	}
}
