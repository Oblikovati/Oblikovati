// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"slices"

	"oblikovati.org/model/feature"
)

// ShellTool is the interactive Shell command: activate it, click the faces to remove
// (the openings), set the wall thickness in the property window, and OK to hollow the
// active part to that thickness.
type ShellTool struct {
	featureEditMode // set ⇒ this panel re-edits a committed shell (see editShellTool)
	faces           []FaceHandle
	seededFaceKeys  [][]byte // edit mode: the feature's existing removed-face keys
	thickness       float64
	added           *feature.PartFeature
}

// NewShellTool returns a shell tool with a default 1-unit wall thickness.
func NewShellTool() *ShellTool { return &ShellTool{thickness: 1} }

// Name implements [Tool].
func (t *ShellTool) Name() string { return "Shell" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *ShellTool) Start(*Session) {}

// AcceptedKinds declares shell picks faces (the faces to open).
func (t *ShellTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectFace} }

// Picks reports the picked faces for the unified highlight.
func (t *ShellTool) Picks() []Selectable { return selectables(t.faces) }

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
	return slices.Contains(t.faces, f)
}

// SetThickness/Thickness set the shell wall thickness (database units).
func (t *ShellTool) SetThickness(d float64) { t.thickness = d }
func (t *ShellTool) Thickness() float64     { return t.thickness }

// Faces returns the picked removed-faces (for the UI to list/highlight).
func (t *ShellTool) Faces() []FaceHandle { return append([]FaceHandle(nil), t.faces...) }

// CanCommit reports whether at least one face is picked and the thickness is positive.
// (A shell with no removed faces hollows to a closed void, which our planar engine does
// not yet build, so we require at least one opening.)
func (t *ShellTool) CanCommit() bool { return t.FaceCount() > 0 && t.thickness > 0 }

// FaceCount counts the selection the panel shows: faces picked this session plus, in
// edit mode, the feature's retained faces.
func (t *ShellTool) FaceCount() int { return len(t.seededFaceKeys) + len(t.faces) }

// selectedFaceKeys is the reference-key set a commit writes: the retained keys plus
// this session's picks.
func (t *ShellTool) selectedFaceKeys() [][]byte {
	keys := cloneKeys(t.seededFaceKeys)
	for _, f := range t.faces {
		keys = append(keys, f.Face.ReferenceKey())
	}
	return keys
}

// Commit hollows the active part to the wall thickness, opening the picked faces, and
// recomputes; a sick feature keeps the tool open by returning an error.
func (t *ShellTool) Commit(s *Session) error {
	if t.IsEditing() {
		return t.commitEdit(s)
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addShell(feature.NewDressUpFeatures(part.Features()))
	part.Recompute()
	s.recordEdit(part, "Shell")
	if !t.added.Health().OK() {
		return errors.New("shell: " + t.added.Health().Reason)
	}
	return nil
}

// addShell builds the shell feature into collection dress — the shared constructor used by
// both Commit (the part's engine) and DraftFeature (a scratch engine).
func (t *ShellTool) addShell(dress *feature.DressUpFeatures) *feature.PartFeature {
	th := t.thickness
	return dress.AddShell(t.selectedFaceKeys(), func() float64 { return th })
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *ShellTool) AddedFeature() *feature.PartFeature { return t.added }

// DraftFeature returns the unattached shell feature the viewport previews before commit
// (satisfying DraftPreviewable), built by the same addShell the commit uses. Empty until a
// face is selected.
func (t *ShellTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addShell(feature.NewDressUpFeatures(fs)), nil
	})
}

// Prompt guides the user through the shell steps.
func (t *ShellTool) Prompt(*Session) string {
	if len(t.faces) == 0 {
		return "Click the faces to remove (openings)"
	}
	return "Set the wall thickness, then click OK"
}

// Cancel restores the default selection filter.
func (t *ShellTool) Cancel(s *Session) {
	if t.IsEditing() {
		cancelFeatureEdit(s, t.target, t.restoreDef)
		return
	}
}

// commitEdit writes the panel state back into the committed shell's definition.
func (t *ShellTool) commitEdit(s *Session) error {
	def := t.target.Definition().(*feature.ShellFeature).Definition()
	def.RemovedFaceKeys = t.selectedFaceKeys()
	def.Thickness = konst(t.thickness)
	return commitFeatureEdit(s, t.target)
}

// ClearFaces empties the face selection — the picks and, in edit mode, the feature's
// retained keys — returning the tool to its pick-faces step.
func (t *ShellTool) ClearFaces() {
	t.faces = nil
	t.seededFaceKeys = nil
}
