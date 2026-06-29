// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
)

// Presentation rows for the Manage ▸ Parameters dialog's derived-parameter-table section
// (M39-F04, #1560): the head asks the session for ready-to-render rows and the link
// candidates, mirroring how ParameterRows keeps units/formatting on the session side. The
// edit verbs are the existing ones in derived_tables.go (AddDerivedParameterTable, …).

// DerivedTableRow is one derived parameter table rendered for the dialog: its stable id, the
// source document (full name for the edit verbs, display label for the UI), the linked
// subset, the source parameters available to link, and the table's health ("" when every
// link resolves, otherwise the reason).
type DerivedTableRow struct {
	ID             int
	SourceDocument string
	SourceDisplay  string
	Linked         []string
	Available      []string
	Health         string
}

// SourceDocumentRow is one candidate source for the link picker: an open parameter-holding
// document other than the active one, with the numeric user parameters it offers to link.
type SourceDocumentRow struct {
	FullName   string
	Display    string
	Parameters []string
}

// DerivedTableRows returns the active parameter holder's derived parameter tables as
// presentation rows (empty when no parameter-holding document is active). Candidates are read
// live from each source; an unreachable source yields an empty Available and a health reason.
func (s *Session) DerivedTableRows() []DerivedTableRow {
	holder, err := s.activeParameterHolder()
	if err != nil {
		return nil
	}
	var out []DerivedTableRow
	for _, t := range holder.Parameters().DerivedTables() {
		row := DerivedTableRow{
			ID: t.ID(), SourceDocument: t.SourceDocument(),
			SourceDisplay: s.documentDisplayName(t.SourceDocument()),
			Linked:        t.Linked(), Health: t.Health().Reason,
		}
		if src, ok := s.LinkableSourceParameters(t.SourceDocument()); ok {
			row.Available = sourceParameterNames(src)
		} else if row.Health == "" {
			row.Health = "source document " + t.SourceDocument() + " is unavailable"
		}
		out = append(out, row)
	}
	return out
}

// LinkableSourceDocuments returns the OTHER open parameter-holding documents (parts or
// assemblies) the active document can derive parameters from, each with the numeric user
// parameters it offers. The active document is excluded (a document cannot derive from
// itself), as are documents that offer no linkable parameters.
func (s *Session) LinkableSourceDocuments() []SourceDocumentRow {
	active := s.ActiveDocument()
	if active == nil {
		return nil
	}
	var out []SourceDocumentRow
	for _, d := range s.workspace.Documents() {
		if d == active {
			continue
		}
		if _, ok := d.Content().(compdef.ParameterHolder); !ok {
			continue
		}
		src, ok := s.LinkableSourceParameters(d.FullDocumentName())
		if !ok || len(src) == 0 {
			continue
		}
		out = append(out, SourceDocumentRow{
			FullName: d.FullDocumentName(), Display: d.DisplayName(),
			Parameters: sourceParameterNames(src),
		})
	}
	return out
}

// documentDisplayName resolves a full document name to its display label, falling back to the
// full name when the document is not open.
func (s *Session) documentDisplayName(fullName string) string {
	if d, ok := s.workspace.ByName(fullName); ok {
		return d.DisplayName()
	}
	return fullName
}

// sourceParameterNames projects the linkable source parameters to their names.
func sourceParameterNames(src []param.SourceParameterValue) []string {
	names := make([]string, len(src))
	for i, sv := range src {
		names[i] = sv.Name
	}
	return names
}
