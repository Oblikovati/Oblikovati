// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"os"

	"oblikovati.org/model/bom"
)

// The Bill of Materials panel (#768): the Assemble ▸ Bill of Materials command opens a panel that
// lists the active assembly's components in one of two standard views — Structured (the hierarchy,
// sub-assemblies nested) or Parts Only (every unique part once with its total quantity) — and
// exports the current view to CSV. The BOM itself (counting, grouping, CSV) is model/bom; this is
// the head/app wiring around it.

// OpenBOM / CloseBOM toggle the Bill of Materials panel; BOMPanelOpen reports its state (the head
// draws the panel only when open).
func (s *Session) OpenBOM()           { s.bomPanelOpen = true }
func (s *Session) CloseBOM()          { s.bomPanelOpen = false }
func (s *Session) BOMPanelOpen() bool { return s.bomPanelOpen }

// BOMViewKind / SetBOMViewKind read and set which standard view the panel shows.
func (s *Session) BOMViewKind() bom.ViewKind        { return s.bomViewKind }
func (s *Session) SetBOMViewKind(kind bom.ViewKind) { s.bomViewKind = kind }

// AssemblyBOM builds the active assembly's Bill of Materials in the currently selected view
// (Structured or Parts Only), erroring when the active document is not an assembly.
//
//	view, err := session.AssemblyBOM()
func (s *Session) AssemblyBOM() (*bom.View, error) {
	asm, err := activeAssembly(s)
	if err != nil {
		return nil, err
	}
	b := bom.New(asm.Occurrences())
	if s.bomViewKind == bom.PartsOnlyView {
		return b.PartsOnly(), nil
	}
	return b.Structured(), nil
}

// ExportBOMCSV writes the active assembly's BOM (current view, standard columns) to path as CSV —
// the Export button on the panel. Returns the build/encode/write error so the head can report it.
func (s *Session) ExportBOMCSV(path string) error {
	view, err := s.AssemblyBOM()
	if err != nil {
		return err
	}
	csv, err := bom.ExportCSV(view, bom.StandardColumns())
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(csv), 0o644)
}
