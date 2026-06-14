// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// DerivedAssemblyData is the serialized form of a derived-assembly feature (#715): the
// source document's identity link (so the derive can re-resolve its source and detect a
// stale one on reopen), whether the link is live, and the non-default per-occurrence
// styles keyed by occurrence path. The source geometry is NOT embedded — it is pulled
// from the resolved source document on open (ADR-0020).
type DerivedAssemblyData struct {
	SourceDocument           string             `yaml:"sourceDocument"`
	SourceInternalName       string             `yaml:"sourceInternalName,omitempty"`
	SourceDatabaseRevisionID string             `yaml:"sourceRevision,omitempty"`
	Linked                   bool               `yaml:"linked"`
	Styles                   []deriveStyleEntry `yaml:"styles,omitempty"`
}

// deriveStyleEntry persists one non-default derive style by the occurrence path it
// applies to (the pointer-free key, valid across save/load).
type deriveStyleEntry struct {
	Path  []string `yaml:"path"`
	Style string   `yaml:"style"`
}

// serializeDerivedAssembly projects a derived-assembly feature to its persisted form.
func serializeDerivedAssembly(d *DerivedAssemblyComponent) *DerivedAssemblyData {
	link := d.SourceLink()
	data := &DerivedAssemblyData{
		SourceDocument:           link.Document,
		SourceInternalName:       link.InternalName,
		SourceDatabaseRevisionID: link.DatabaseRevisionID,
		Linked:                   d.linked,
	}
	for _, s := range d.StylesByPath() {
		data.Styles = append(data.Styles, deriveStyleEntry{Path: []string(s.Path), Style: s.Style.String()})
	}
	return data
}

// restoreDerivedAssembly rebuilds an UNBOUND derived-assembly feature from its payload and
// adds it to the engine. The source is rebound — and staleness computed — later, when the
// part's reference graph resolves the source document (see compdef part ResolveReferences).
func restoreDerivedAssembly(fs *PartFeatures, data *DerivedAssemblyData) (*PartFeature, error) {
	if data == nil {
		return nil, fmt.Errorf("derivedAssembly feature is missing its payload")
	}
	link := DeriveSourceLink{
		Document:           data.SourceDocument,
		InternalName:       data.SourceInternalName,
		DatabaseRevisionID: data.SourceDatabaseRevisionID,
	}
	styles := make([]DeriveStyleAtPath, 0, len(data.Styles))
	for _, e := range data.Styles {
		style, ok := DeriveStyleFromName(e.Style)
		if !ok {
			return nil, fmt.Errorf("derivedAssembly: unknown derive style %q (want include/exclude/subtract)", e.Style)
		}
		styles = append(styles, DeriveStyleAtPath{Path: occurrence.OccurrencePath(e.Path), Style: style})
	}
	return fs.Add(RestoreDerivedAssembly(link, data.Linked, styles)), nil
}

// DerivedPartData is the serialized form of a derived-part feature (#715/#717): the
// source document's identity link, the geometry transform applied to the pulled bodies
// (16 row-major cells — a reflection makes an opposite-hand part), and the linked flag.
// The source geometry is pulled from the resolved source on open, never embedded.
type DerivedPartData struct {
	SourceDocument           string    `yaml:"sourceDocument"`
	SourceInternalName       string    `yaml:"sourceInternalName,omitempty"`
	SourceDatabaseRevisionID string    `yaml:"sourceRevision,omitempty"`
	Transform                []float64 `yaml:"transform,omitempty"`
	Linked                   bool      `yaml:"linked"`
}

// serializeDerivedPart projects a derived-part feature to its persisted form.
func serializeDerivedPart(d *DerivedPartComponent) *DerivedPartData {
	link := d.SourceLink()
	cells := d.Transform().Cells()
	return &DerivedPartData{
		SourceDocument:           link.Document,
		SourceInternalName:       link.InternalName,
		SourceDatabaseRevisionID: link.DatabaseRevisionID,
		Transform:                cells[:],
		Linked:                   d.Linked(),
	}
}

// restoreDerivedPart rebuilds an UNBOUND derived-part feature from its payload and adds it
// to the engine. The source is rebound — and staleness computed — later, when the part's
// reference graph resolves the source document.
func restoreDerivedPart(fs *PartFeatures, data *DerivedPartData) (*PartFeature, error) {
	if data == nil {
		return nil, fmt.Errorf("derived feature is missing its payload")
	}
	link := DeriveSourceLink{
		Document:           data.SourceDocument,
		InternalName:       data.SourceInternalName,
		DatabaseRevisionID: data.SourceDatabaseRevisionID,
	}
	transform, err := matrixFromCells(data.Transform)
	if err != nil {
		return nil, fmt.Errorf("derived feature: %w", err)
	}
	return fs.Add(RestoreDerivedPart(link, transform, data.Linked)), nil
}

// matrixFromCells rebuilds a transform from its persisted 16 row-major cells; an empty
// slice restores the identity (the pull-as-is derive).
func matrixFromCells(cells []float64) (math.Matrix4, error) {
	if len(cells) == 0 {
		return math.Identity4(), nil
	}
	if len(cells) != 16 {
		return math.Matrix4{}, fmt.Errorf("transform has %d cells, want 16", len(cells))
	}
	var c [16]float64
	copy(c[:], cells)
	return math.Matrix4FromCells(c), nil
}
