// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/param"
)

// Derived parameter tables (M02-F06, Oblikovati#605): the session is the
// resolution layer — it finds the source document through the workspace,
// feeds its current values to the document-free model in model/param, records
// the cross-document reference in the workspace graph, and re-syncs deriving
// documents whenever a source finalizes an edit (the recordEdit seam).

// LinkableSourceParameters returns the named document's numeric user
// parameters — the candidates a derived table may link. ok=false means the
// document is not open (or holds no part), which sync treats as a broken link.
func (s *Session) LinkableSourceParameters(sourceDocument string) ([]param.SourceParameterValue, bool) {
	d, ok := s.workspace.ByName(sourceDocument)
	if !ok {
		return nil, false
	}
	def, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		return nil, false
	}
	var out []param.SourceParameterValue
	for _, p := range def.Parameters().All() {
		if p.Kind() == param.UserParam && p.IsNumeric() {
			out = append(out, param.SourceParameterValue{Name: p.Name(), Value: p.Value()})
		}
	}
	return out, true
}

// AddDerivedParameterTable links parameters from another open document into
// the active part, recording the document reference so the link survives in
// the workspace graph. One undo step.
func (s *Session) AddDerivedParameterTable(sourceDocument string, linked []string) (*param.DerivedParameterTable, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	if active := s.ActiveDocument(); active != nil && active.FullDocumentName() == sourceDocument {
		return nil, fmt.Errorf("app: a document cannot derive parameters from itself (%q)", sourceDocument)
	}
	source, ok := s.LinkableSourceParameters(sourceDocument)
	if !ok {
		return nil, fmt.Errorf("app: no open part document named %q to derive from", sourceDocument)
	}
	t, err := part.Parameters().AddDerivedTable(sourceDocument, linked, source)
	if err != nil {
		return nil, err
	}
	if _, err := s.ActiveDocument().AddReference(sourceDocument); err != nil {
		return nil, err
	}
	part.Recompute()
	s.recordEdit(part, "Derive Parameters")
	return t, nil
}

// SetDerivedTableLinked replaces a table's linked subset on the active part.
// One undo step.
func (s *Session) SetDerivedTableLinked(id int, linked []string) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t, ok := part.Parameters().DerivedTableByID(id)
	if !ok {
		return fmt.Errorf("app: no derived table with id %d", id)
	}
	source, ok := s.LinkableSourceParameters(t.SourceDocument())
	if !ok {
		return fmt.Errorf("app: source document %q is not open; cannot relink", t.SourceDocument())
	}
	if err := part.Parameters().SetDerivedTableLinked(id, linked, source); err != nil {
		return err
	}
	part.Features().MarkAllDirty()
	part.Recompute()
	s.recordEdit(part, "Edit Derived Parameters")
	return nil
}

// DeleteDerivedParameterTable removes a table and its derived parameters from
// the active part (component-owned tables refuse). One undo step.
func (s *Session) DeleteDerivedParameterTable(id int) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if err := part.Parameters().DeleteDerivedTable(id); err != nil {
		return err
	}
	part.Features().MarkAllDirty()
	part.Recompute()
	s.recordEdit(part, "Delete Derived Parameters")
	return nil
}

// ResyncDerivedFromActiveDocument pushes the active document's current values
// into the documents deriving from it. The wire router calls this after every
// mutating method — legacy wire edits (parameters.set/add) record no undo
// step, so the recordEdit hook alone would miss them.
func (s *Session) ResyncDerivedFromActiveDocument() {
	if d := s.ActiveDocument(); d != nil {
		s.resyncDerivedTables(d, map[string]bool{})
	}
}

// resyncDerivedTables pushes a changed document's parameter values into every
// open document deriving from it, then follows the chain (A→B→C) with a
// visited guard against reference cycles. Syncs are computed consequences of
// the source edit, not edits of their own — they record no undo step.
func (s *Session) resyncDerivedTables(source *doc.Document, visited map[string]bool) {
	name := source.FullDocumentName()
	if visited[name] {
		return
	}
	visited[name] = true
	values, sourceOK := s.LinkableSourceParameters(name)
	for _, d := range source.ReferencingDocuments() {
		if s.resyncDocumentFrom(d, name, values, sourceOK) {
			s.resyncDerivedTables(d, visited)
		}
	}
}

// resyncDocumentFrom refreshes one deriving document's tables that point at
// sourceName, reporting whether anything was synced (and recomputing if so).
func (s *Session) resyncDocumentFrom(d *doc.Document, sourceName string, values []param.SourceParameterValue, sourceOK bool) bool {
	def, isPart := d.Content().(*compdef.PartComponentDefinition)
	if !isPart {
		return false
	}
	synced := false
	for _, t := range def.Parameters().DerivedTables() {
		if t.SourceDocument() != sourceName {
			continue
		}
		_ = def.Parameters().SyncDerivedTable(t.ID(), values, sourceOK)
		synced = true
	}
	if synced {
		def.Features().MarkAllDirty()
		def.Recompute()
	}
	return synced
}
