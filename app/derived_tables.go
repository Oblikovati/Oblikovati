// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
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
	holder, ok := d.Content().(compdef.ParameterHolder)
	if !ok {
		return nil, false
	}
	var out []param.SourceParameterValue
	for _, p := range holder.Parameters().All() {
		if p.Kind() == param.UserParam && p.IsNumeric() {
			out = append(out, param.SourceParameterValue{Name: p.Name(), Value: p.Value()})
		}
	}
	return out, true
}

// paramEditTarget is the active document's content when it both holds parameters and records
// undo — a part or an assembly. The derived-table mutations need both faces: [compdef.ParameterHolder]
// to edit the table and recompute, recipeStore to capture one undo step (M39-F02, #1558).
type paramEditTarget interface {
	compdef.ParameterHolder
	recipeStore
}

// activeParameterHolder resolves the active document's parameter-editing target, erroring when
// there is no active document or it holds no parameters (a drawing, say). Both a part and an
// assembly satisfy paramEditTarget, so derived-table editing is no longer part-only.
func (s *Session) activeParameterHolder() (paramEditTarget, error) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, errors.New("app: no active document")
	}
	target, ok := d.Content().(paramEditTarget)
	if !ok {
		return nil, fmt.Errorf("app: active document %q holds no parameters (not a part or assembly)", d.DisplayName())
	}
	return target, nil
}

// AddDerivedParameterTable links parameters from another open document into
// the active part or assembly, recording the document reference so the link
// survives in the workspace graph. One undo step.
func (s *Session) AddDerivedParameterTable(sourceDocument string, linked []string) (*param.DerivedParameterTable, error) {
	target, err := s.activeParameterHolder()
	if err != nil {
		return nil, err
	}
	if active := s.ActiveDocument(); active != nil && active.FullDocumentName() == sourceDocument {
		return nil, fmt.Errorf("app: a document cannot derive parameters from itself (%q)", sourceDocument)
	}
	source, ok := s.LinkableSourceParameters(sourceDocument)
	if !ok {
		return nil, fmt.Errorf("app: no open document named %q to derive parameters from", sourceDocument)
	}
	t, err := target.Parameters().AddDerivedTable(sourceDocument, linked, source)
	if err != nil {
		return nil, err
	}
	if _, err := s.ActiveDocument().AddReference(sourceDocument); err != nil {
		return nil, err
	}
	s.autoExportSourceParameters(sourceDocument, linked) // Add2 semantics: linking exports the source params
	target.Recompute()
	s.recordEdit(target, "Derive Parameters")
	return t, nil
}

// autoExportSourceParameters marks each linked source parameter as exported on the source
// document — the reference API's Add2 auto-export-on-link (M39-F05, #1561). A parameter need
// not be pre-exported to be linked; linking exports it so the source advertises it. The flag
// lives in the source document's recipe (a benign, reversible marker), so it is not recorded
// as a separate undo step on the non-active source.
func (s *Session) autoExportSourceParameters(sourceDocument string, linked []string) {
	d, ok := s.workspace.ByName(sourceDocument)
	if !ok {
		return
	}
	holder, ok := d.Content().(compdef.ParameterHolder)
	if !ok {
		return
	}
	for _, name := range linked {
		if p, found := holder.Parameters().ByName(name); found {
			p.ExposedAsProperty = true
		}
	}
}

// SetDerivedTableLinked replaces a table's linked subset on the active part or
// assembly. One undo step.
func (s *Session) SetDerivedTableLinked(id int, linked []string) error {
	target, err := s.activeParameterHolder()
	if err != nil {
		return err
	}
	t, ok := target.Parameters().DerivedTableByID(id)
	if !ok {
		return fmt.Errorf("app: no derived table with id %d", id)
	}
	source, ok := s.LinkableSourceParameters(t.SourceDocument())
	if !ok {
		return fmt.Errorf("app: source document %q is not open; cannot relink", t.SourceDocument())
	}
	if err := target.Parameters().SetDerivedTableLinked(id, linked, source); err != nil {
		return err
	}
	s.autoExportSourceParameters(t.SourceDocument(), linked) // Add2 semantics: linking exports the source params
	target.RecomputeAfterParameterEdit()
	s.recordEdit(target, "Edit Derived Parameters")
	return nil
}

// DeleteDerivedParameterTable removes a table and its derived parameters from
// the active part or assembly (component-owned tables refuse). One undo step.
func (s *Session) DeleteDerivedParameterTable(id int) error {
	target, err := s.activeParameterHolder()
	if err != nil {
		return err
	}
	if err := target.Parameters().DeleteDerivedTable(id); err != nil {
		return err
	}
	target.RecomputeAfterParameterEdit()
	s.recordEdit(target, "Delete Derived Parameters")
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
	holder, ok := d.Content().(compdef.ParameterHolder)
	if !ok {
		return false
	}
	synced := false
	for _, t := range holder.Parameters().DerivedTables() {
		if t.SourceDocument() != sourceName {
			continue
		}
		_ = holder.Parameters().SyncDerivedTable(t.ID(), values, sourceOK)
		synced = true
	}
	if synced {
		holder.RecomputeAfterParameterEdit()
	}
	return synced
}
