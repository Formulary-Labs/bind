package mapping_test

import (
	"testing"

	gemara "github.com/gemaraproj/go-gemara"

	"github.com/Formulary-Labs/bind/mapping"
)

func makeDoc(src, tgt string, mappings []gemara.Mapping) *gemara.MappingDocument {
	return &gemara.MappingDocument{
		Title: "test mapping",
		SourceReference: gemara.TypedMapping{ReferenceId: src},
		TargetReference: gemara.TypedMapping{ReferenceId: tgt},
		Mappings:        mappings,
	}
}

func TestResolve_basicMapping(t *testing.T) {
	doc := makeDoc("iso27001", "nist800-53", []gemara.Mapping{
		{Id: "M-001", Source: "A.5.1", Relationship: gemara.RelEquivalent,
			Targets: []gemara.MappingTarget{{EntryId: "AC-1", Strength: 8}}},
		{Id: "M-002", Source: "A.5.2", Relationship: gemara.RelSupports,
			Targets: []gemara.MappingTarget{{EntryId: "AC-2", Strength: 6}}},
	})

	result := mapping.Resolve(doc, nil, nil)
	if result.Source != "iso27001" {
		t.Errorf("Source = %q, want iso27001", result.Source)
	}
	if result.Target != "nist800-53" {
		t.Errorf("Target = %q, want nist800-53", result.Target)
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	if result.Mapped != 2 {
		t.Errorf("Mapped = %d, want 2", result.Mapped)
	}
	if result.Unmapped != 0 {
		t.Errorf("Unmapped = %d, want 0", result.Unmapped)
	}
	if len(result.Entries) != 2 {
		t.Errorf("Entries = %d, want 2", len(result.Entries))
	}
}

func TestResolve_noMatch(t *testing.T) {
	doc := makeDoc("iso27001", "iec62443", []gemara.Mapping{
		{Id: "M-001", Source: "A.5.1", Relationship: gemara.RelNoMatch},
	})

	result := mapping.Resolve(doc, nil, nil)
	if result.Unmapped != 1 {
		t.Errorf("Unmapped = %d, want 1", result.Unmapped)
	}
	if result.Mapped != 0 {
		t.Errorf("Mapped = %d, want 0", result.Mapped)
	}
}

func TestResolve_titleResolutionFromCatalog(t *testing.T) {
	srcCat := &gemara.ControlCatalog{
		Controls: []gemara.Control{
			{Id: "A.5.1", Title: "Policies for information security"},
		},
	}
	tgtCat := &gemara.ControlCatalog{
		Controls: []gemara.Control{
			{Id: "AC-1", Title: "Access Control Policy and Procedures"},
		},
	}

	doc := makeDoc("iso27001", "nist800-53", []gemara.Mapping{
		{Id: "M-001", Source: "A.5.1", Relationship: gemara.RelEquivalent,
			Targets: []gemara.MappingTarget{{EntryId: "AC-1", Strength: 9}}},
	})

	result := mapping.Resolve(doc, srcCat, tgtCat)
	if len(result.Entries) == 0 {
		t.Fatal("expected at least one entry")
	}
	entry := result.Entries[0]
	if entry.SourceTitle != "Policies for information security" {
		t.Errorf("SourceTitle = %q, want %q", entry.SourceTitle, "Policies for information security")
	}
	if len(entry.Targets) == 0 {
		t.Fatal("expected at least one target")
	}
	if entry.Targets[0].TargetTitle != "Access Control Policy and Procedures" {
		t.Errorf("TargetTitle = %q, want %q", entry.Targets[0].TargetTitle, "Access Control Policy and Procedures")
	}
}

func TestResolve_unknownSourceControlFlagged(t *testing.T) {
	srcCat := &gemara.ControlCatalog{
		Controls: []gemara.Control{
			{Id: "A.5.1", Title: "Some control"},
		},
	}

	doc := makeDoc("iso27001", "nist800-53", []gemara.Mapping{
		{Id: "M-001", Source: "X.9.9", Relationship: gemara.RelSupports,
			Targets: []gemara.MappingTarget{{EntryId: "SC-1"}}},
	})

	result := mapping.Resolve(doc, srcCat, nil)
	if len(result.Flags) == 0 {
		t.Error("expected flag for unknown source control X.9.9")
	}
}

func TestResolve_emptyDocument(t *testing.T) {
	doc := makeDoc("src", "tgt", nil)
	result := mapping.Resolve(doc, nil, nil)
	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
	if result.Mapped != 0 {
		t.Errorf("Mapped = %d, want 0 for empty document", result.Mapped)
	}
}
