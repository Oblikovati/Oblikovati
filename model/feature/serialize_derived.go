// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

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
