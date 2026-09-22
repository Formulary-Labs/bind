// Package mapping implements cross-framework control mapping using the gemara
// MappingDocument artifact (Layer 1). It reads a MappingDocument YAML and
// two ControlCatalogs (source and target), then resolves the atomic control
// mappings into a structured MappingResult.
package mapping

import (
	"context"
	"fmt"
	"strings"

	gemara "github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/fetcher"
)

// MappingResult is the structured output of a bind run.
type MappingResult struct { //nolint:revive // stutter is intentional
	// Source is the source framework identifier.
	Source string `json:"source"`
	// Target is the target framework identifier.
	Target string `json:"target"`
	// Total is the total number of mappings in the document.
	Total int `json:"total"`
	// Mapped is the count of source controls that have at least one target.
	Mapped int `json:"mapped"`
	// Unmapped is the count of source controls with no-match relationship.
	Unmapped int `json:"unmapped"`
	// Entries is the resolved per-control mapping list.
	Entries []MappingEntry `json:"entries"`
	// Flags captures any data quality issues found during resolution.
	Flags []string `json:"flags,omitempty"`
}

// MappingEntry is a single resolved mapping between a source control and its
// target controls.
type MappingEntry struct { //nolint:revive // stutter is intentional
	SourceID     string         `json:"source_id"`
	SourceTitle  string         `json:"source_title,omitempty"`
	Relationship string         `json:"relationship"`
	Targets      []MappedTarget `json:"targets,omitempty"`
}

// MappedTarget is one target control in a MappingEntry.
type MappedTarget struct {
	TargetID    string `json:"target_id"`
	TargetTitle string `json:"target_title,omitempty"`
	Strength    int64  `json:"strength,omitempty"`
	Rationale   string `json:"rationale,omitempty"`
}

// LoadMappingDocument loads a gemara MappingDocument from a YAML/JSON file.
func LoadMappingDocument(path string) (*gemara.MappingDocument, error) {
	f := &fetcher.File{}
	doc, err := gemara.Load[gemara.MappingDocument](context.Background(), f, path)
	if err != nil {
		return nil, fmt.Errorf("loading mapping document from %q: %w", path, err)
	}
	return doc, nil
}

// LoadControlCatalog loads a gemara ControlCatalog from a YAML/JSON file.
func LoadControlCatalog(path string) (*gemara.ControlCatalog, error) {
	f := &fetcher.File{}
	cat, err := gemara.Load[gemara.ControlCatalog](context.Background(), f, path)
	if err != nil {
		return nil, fmt.Errorf("loading control catalog from %q: %w", path, err)
	}
	return cat, nil
}

// Resolve resolves the mappings in the MappingDocument against the source and
// target catalogs. When a catalog is nil, title resolution is skipped (IDs are
// always used). Catalog paths are optional — pass empty strings to skip loading.
func Resolve(doc *gemara.MappingDocument, srcCat, tgtCat *gemara.ControlCatalog) MappingResult {
	srcTitles := controlTitleIndex(srcCat)
	tgtTitles := controlTitleIndex(tgtCat)

	srcID := doc.SourceReference.ReferenceId
	tgtID := doc.TargetReference.ReferenceId

	result := MappingResult{
		Source: srcID,
		Target: tgtID,
		Total:  len(doc.Mappings),
	}

	for _, m := range doc.Mappings {
		rel := m.Relationship.String()
		entry := MappingEntry{
			SourceID:     m.Source,
			SourceTitle:  srcTitles[m.Source],
			Relationship: rel,
		}

		if strings.EqualFold(rel, "no-match") || len(m.Targets) == 0 {
			result.Unmapped++
		} else {
			result.Mapped++
			for _, t := range m.Targets {
				entry.Targets = append(entry.Targets, MappedTarget{
					TargetID:    t.EntryId,
					TargetTitle: tgtTitles[t.EntryId],
					Strength:    t.Strength,
					Rationale:   t.Rationale,
				})
			}
		}

		// Validate source ID exists in source catalog.
		if srcCat != nil && srcTitles[m.Source] == "" {
			result.Flags = append(result.Flags,
				fmt.Sprintf("[CONFLICT — VERIFY] source control %q not found in source catalog %q", m.Source, srcID))
		}

		result.Entries = append(result.Entries, entry)
	}

	return result
}

// controlTitleIndex builds a control ID → title map from a ControlCatalog.
// Returns an empty map when cat is nil.
func controlTitleIndex(cat *gemara.ControlCatalog) map[string]string {
	index := make(map[string]string)
	if cat == nil {
		return index
	}
	for _, c := range cat.Controls {
		index[c.Id] = c.Title
	}
	return index
}
