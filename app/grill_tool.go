// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// GrillTool is the interactive Grill command: pick one or more boundary vent profiles (whose
// inner loops are the bridging ribs/spars/islands), optionally set a draft angle, then OK to
// cut the vent through the active part. It exposes its parameters through ParameterizedTool, so
// the head renders the generic property dialog.
type GrillTool struct {
	profiles []ProfileHandle
	draft    float64
	added    *feature.PartFeature
}

// NewGrillTool returns a grill tool with no draft.
func NewGrillTool() *GrillTool { return &GrillTool{} }

// Name implements [Tool].
func (t *GrillTool) Name() string { return "Grill" }

// Start filters selection to closed regions so clicks pick a boundary profile.
func (t *GrillTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectProfile)) }

// Pick captures a single clicked region (replacing any previous selection).
func (t *GrillTool) Pick(_ *Session, sel Selectable) {
	if p, ok := sel.(ProfileHandle); ok {
		t.profiles = []ProfileHandle{p}
	}
}

// PickWithMods extends the selection on Ctrl+click (toggling), to vent several boundaries.
func (t *GrillTool) PickWithMods(s *Session, sel Selectable, mods Modifier) {
	p, ok := sel.(ProfileHandle)
	if !ok {
		return
	}
	if !mods.Has(CtrlMod) {
		t.Pick(s, sel)
		return
	}
	if i := indexOfProfile(t.profiles, p); i >= 0 {
		t.profiles = append(t.profiles[:i], t.profiles[i+1:]...)
		return
	}
	t.profiles = append(t.profiles, p)
}

// The option the property window drives: the draft angle on the vent walls.
func (t *GrillTool) SetDraft(d float64) { t.draft = d }
func (t *GrillTool) Draft() float64     { return t.draft }

// Params exposes the draft angle for the generic property dialog.
func (t *GrillTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{{Label: "Draft", Get: t.Draft, Set: t.SetDraft}}}
}

// PickedProfiles returns the picked boundaries for the unified tool highlight.
func (t *GrillTool) PickedProfiles() []ProfileHandle {
	return append([]ProfileHandle(nil), t.profiles...)
}

// CanCommit reports whether at least one boundary is picked.
func (t *GrillTool) CanCommit() bool { return len(t.profiles) > 0 }

// Commit cuts the grill on the active part and recomputes; a sick feature keeps the tool open.
func (t *GrillTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addGrill(part.Features())
	part.Recompute()
	s.recordEdit(part, "Grill")
	if !t.added.Health().OK() {
		return errors.New("grill: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// addGrill builds the grill feature into engine fs — shared by Commit and the preview.
func (t *GrillTool) addGrill(fs *feature.PartFeatures) *feature.PartFeature {
	skt := t.profiles[0].Sketch
	return feature.NewGrillFeatures(fs).Add(&feature.GrillDefinition{
		Sketch: skt, Boundaries: profileIndicesOn(t.profiles, skt), Draft: t.draft,
	})
}

// DraftFeature returns the unattached grill feature the viewport previews before commit.
func (t *GrillTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addGrill(fs), nil
	})
}

// Cancel restores the default selection filter.
func (t *GrillTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *GrillTool) AddedFeature() *feature.PartFeature { return t.added }
