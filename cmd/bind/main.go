// bind resolves cross-framework control mappings from a gemara MappingDocument.
//
// Usage:
//
//	bind --document <mapping.yaml> [--source-catalog <src.yaml>] [--target-catalog <tgt.yaml>] [flags]
//
// bind reads a gemara MappingDocument and resolves each control mapping to
// a structured output. Providing the source and target ControlCatalogs
// enables title resolution and source-ID validation.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	gemara "github.com/gemaraproj/go-gemara"

	"github.com/Formulary-Labs/bind/mapping"
	"github.com/Formulary-Labs/substrate/exit"
	"github.com/Formulary-Labs/substrate/provenance"
)

const version = "0.1.0"

func main() {
	var (
		documentFlag      = flag.String("document", "", "Path to gemara MappingDocument YAML (required)")
		sourceCatalogFlag = flag.String("source-catalog", "", "Path to source gemara ControlCatalog YAML (enables title resolution)")
		targetCatalogFlag = flag.String("target-catalog", "", "Path to target gemara ControlCatalog YAML (enables title resolution)")
		programFlag       = flag.String("program", "", "Program slug for provenance logging")
		fmtFlag           = flag.String("format", "json", "Output format: json (default), md, csv")
		filterFlag        = flag.String("filter", "", "Filter entries: mapped, unmapped, or empty for all")
		versionFlag       = flag.Bool("version", false, "Print version and exit")
	)
	flag.Usage = usage
	flag.Parse()

	if *versionFlag {
		fmt.Printf("bind version %s\n", version)
		os.Exit(exit.OK)
	}

	if *documentFlag == "" {
		fmt.Fprintln(os.Stderr, `{"error": "--document is required", "code": 2}`)
		flag.Usage()
		os.Exit(exit.ToolError)
	}

	doc, err := mapping.LoadMappingDocument(*documentFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"error": %q, "code": 2}`+"\n", err.Error())
		os.Exit(exit.ToolError)
	}

	// Load optional catalogs for title resolution and validation.
	srcCat := loadOptionalCatalog(*sourceCatalogFlag)
	tgtCat := loadOptionalCatalog(*targetCatalogFlag)

	result := mapping.Resolve(doc, srcCat, tgtCat)
	result = applyFilter(result, *filterFlag)

	switch *fmtFlag {
	case "md":
		printMD(result)
	case "csv":
		printCSV(result)
	default:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result) //nolint:errcheck
	}

	_ = provenance.Write("logs/provenance.jsonl", provenance.Entry{
		Spec:        "functions/control-coverage-spec.md",
		Output:      *documentFlag,
		OutputType:  "other",
		Program:     *programFlag,
		Purpose:     fmt.Sprintf("bind: resolved %d/%d cross-framework mappings (%s → %s)", result.Mapped, result.Total, result.Source, result.Target),
		Reusability: provenance.Instance,
		QualityGate: provenance.Pass,
		Tool:        "bind",
		ToolVersion: version,
	})
}

func loadOptionalCatalog(path string) *gemara.ControlCatalog {
	if path == "" {
		return nil
	}
	cat, err := mapping.LoadControlCatalog(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load catalog %q: %v\n", path, err)
		return nil
	}
	return cat
}

func applyFilter(r mapping.MappingResult, filter string) mapping.MappingResult {
	if filter == "" {
		return r
	}
	var filtered []mapping.MappingEntry
	for _, e := range r.Entries {
		switch strings.ToLower(filter) {
		case "mapped":
			if len(e.Targets) > 0 {
				filtered = append(filtered, e)
			}
		case "unmapped":
			if len(e.Targets) == 0 {
				filtered = append(filtered, e)
			}
		}
	}
	r.Entries = filtered
	return r
}

func printMD(r mapping.MappingResult) {
	fmt.Printf("# Cross-Framework Mapping: %s → %s\n\n", r.Source, r.Target)
	fmt.Printf("**Total:** %d | **Mapped:** %d | **Unmapped:** %d\n\n", r.Total, r.Mapped, r.Unmapped)
	if len(r.Flags) > 0 {
		fmt.Printf("## Flags\n\n")
		for _, f := range r.Flags {
			fmt.Printf("- %s\n", f)
		}
		fmt.Printf("\n")
	}
	fmt.Printf("| Source | Relationship | Target Controls | Strength |\n|---|---|---|---|\n")
	for _, e := range r.Entries {
		var targets []string
		var maxStr int64
		for _, t := range e.Targets {
			label := t.TargetID
			if t.TargetTitle != "" {
				label = fmt.Sprintf("%s — %s", t.TargetID, t.TargetTitle)
			}
			targets = append(targets, label)
			if t.Strength > maxStr {
				maxStr = t.Strength
			}
		}
		srcLabel := e.SourceID
		if e.SourceTitle != "" {
			srcLabel = fmt.Sprintf("%s — %s", e.SourceID, e.SourceTitle)
		}
		tgtStr := strings.Join(targets, "<br>")
		if tgtStr == "" {
			tgtStr = "—"
		}
		strLabel := "—"
		if maxStr > 0 {
			strLabel = fmt.Sprintf("%d/10", maxStr)
		}
		fmt.Printf("| %s | %s | %s | %s |\n", srcLabel, e.Relationship, tgtStr, strLabel)
	}
}

func printCSV(r mapping.MappingResult) {
	fmt.Println("source_id,source_title,relationship,target_id,target_title,strength,rationale")
	for _, e := range r.Entries {
		if len(e.Targets) == 0 {
			fmt.Printf("%s,%s,%s,,,\n",
				csvEsc(e.SourceID), csvEsc(e.SourceTitle), csvEsc(e.Relationship))
			continue
		}
		for _, t := range e.Targets {
			fmt.Printf("%s,%s,%s,%s,%s,%d,%s\n",
				csvEsc(e.SourceID), csvEsc(e.SourceTitle), csvEsc(e.Relationship),
				csvEsc(t.TargetID), csvEsc(t.TargetTitle), t.Strength, csvEsc(t.Rationale))
		}
	}
}

func csvEsc(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

func usage() {
	fmt.Fprintln(os.Stderr, `bind — cross-framework control mapping resolver

Usage:
  bind --document <mapping.yaml> [flags]

Flags:
  --document string         Path to gemara MappingDocument YAML (required)
  --source-catalog string   Path to source gemara ControlCatalog YAML (enables title resolution)
  --target-catalog string   Path to target gemara ControlCatalog YAML (enables title resolution)
  --program string          Program slug for provenance logging
  --format string           Output format: json (default), md, csv
  --filter string           Filter output: mapped, unmapped, or all (default)
  --version                 Print version and exit

bind reads a gemara MappingDocument (Layer 1 artifact) and resolves the
cross-framework control mappings it defines. Providing source and target
ControlCatalogs enables human-readable title resolution and source-ID
validation. Without catalogs, IDs are used directly.

Examples:
  bind --document mappings/iso27001-to-nist800-53.yaml
  bind --document mappings/iso27001-to-nist800-53.yaml \
       --source-catalog catalogs/iso27001.yaml \
       --target-catalog catalogs/nist800-53.yaml \
       --format md
  bind --document mappings/iso42001-to-iso27001.yaml --filter unmapped --format csv`)
}
