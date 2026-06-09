// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// ShellTool is the interactive Shell command: activate it, click the faces to remove
// (the openings), set the wall thickness in the property window, and OK to hollow the
// active part to that thickness.
type ShellTool struct {
	faces     []FaceHandle
	thickness float64
	added     *feature.PartFeature
}

// NewShellTool returns a shell tool with a default 1-unit wall thickness.
func NewShellTool() *ShellTool { return &ShellTool{thickness: 1} }

// Name implements [Tool].
func (t *ShellTool) Name() string { return "Shell" }

// Start sets the selection filter to faces so clicks pick faces to open.
func (t *ShellTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectFace)) }

// Pick appends the clicked face (ignoring one already chosen, so a double-click does not
// duplicate it).
func (t *ShellTool) Pick(_ *Session, sel Selectable) {
	f, ok := sel.(FaceHandle)
	if !ok || t.hasFace(f) {
		return
	}
	t.faces = append(t.faces, f)
}

func (t *ShellTool) hasFace(f FaceHandle) bool {
	for _, h := range t.faces {
		if h == f {
			return true
		}
	}
	return false
}

// SetThickness/Thickness set the shell wall thickness (database units).
func (t *ShellTool) SetThickness(d float64) { t.thickness = d }
func (t *ShellTool) Thickness() float64     { return t.thickness }

// Faces returns the picked removed-faces (for the UI to list/highlight).
func (t *ShellTool) Faces() []FaceHandle { return append([]FaceHandle(nil), t.faces...) }

// CanCommit reports whether at least one face is picked and the thickness is positive.
// (A shell with no removed faces hollows to a closed void, which our planar engine does
// not yet build, so we require at least one opening.)
func (t *ShellTool) CanCommit() bool { return len(t.faces) > 0 && t.thickness > 0 }

// Commit hollows the active part to the wall thickness, opening the picked faces, and
// recomputes; a sick feature keeps the tool open by returning an error.
func (t *ShellTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	keys := make([][]byte, len(t.faces))
	for i, f := range t.faces {
		keys[i] = f.Face.ReferenceKey()
	}
	th := t.thickness
	t.added = feature.NewDressUpFeatures(part.Features()).AddShell(keys, func() float64 { return th })
	part.Recompute()
	s.recordEdit(part, "Shell")
	if !t.added.Health().OK() {
		return errors.New("shell: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *ShellTool) AddedFeature() *feature.PartFeature { return t.added }

// Prompt guides the user through the shell steps.
func (t *ShellTool) Prompt(*Session) string {
	if len(t.faces) == 0 {
		return "Click the faces to remove (openings)"
	}
	return "Set the wall thickness, then click OK"
}

// Cancel restores the default selection filter.
func (t *ShellTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }
