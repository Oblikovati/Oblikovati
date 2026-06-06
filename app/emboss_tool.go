// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati/model/feature"
)

// EmbossTool is the interactive Emboss command: pick one or more closed sketch regions, set the
// depth and whether to engrave (cut) or raise (join), then OK to emboss the active part. It
// exposes its parameters through ParameterizedTool, so the head renders the generic dialog.
type EmbossTool struct {
	profiles []ProfileHandle
	depth    float64
	engrave  bool
	added    *feature.PartFeature
}

// NewEmbossTool returns an emboss tool defaulting to a 1-unit raised emboss.
func NewEmbossTool() *EmbossTool { return &EmbossTool{depth: 1} }

// Name implements [Tool].
func (t *EmbossTool) Name() string { return "Emboss" }

// Start filters selection to closed regions so clicks pick a profile.
func (t *EmbossTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectProfile)) }

// Pick captures a single clicked region (replacing any previous selection).
func (t *EmbossTool) Pick(_ *Session, sel Selectable) {
	if p, ok := sel.(ProfileHandle); ok {
		t.profiles = []ProfileHandle{p}
	}
}

// PickWithMods extends the selection on Ctrl+click (toggling), to emboss several regions at once.
func (t *EmbossTool) PickWithMods(s *Session, sel Selectable, mods Modifier) {
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

// The options the property window drives: depth (database units) and engrave (cut vs raise).
func (t *EmbossTool) SetDepth(d float64) { t.depth = d }
func (t *EmbossTool) Depth() float64     { return t.depth }
func (t *EmbossTool) SetEngrave(v bool)  { t.engrave = v }
func (t *EmbossTool) Engrave() bool      { return t.engrave }

// Params exposes the emboss depth and engrave flag for the generic property dialog.
func (t *EmbossTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{{Label: "Depth", Get: t.Depth, Set: t.SetDepth}},
		Bools:  []BoolParam{{Label: "Engrave (cut)", Get: t.Engrave, Set: t.SetEngrave}},
	}
}

// PickedProfiles returns the picked regions for the unified tool highlight.
func (t *EmbossTool) PickedProfiles() []ProfileHandle {
	return append([]ProfileHandle(nil), t.profiles...)
}

// CanCommit reports whether a region is picked and the depth is positive.
func (t *EmbossTool) CanCommit() bool { return len(t.profiles) > 0 && t.depth > 0 }

// Commit embosses the active part and recomputes; a sick feature keeps the tool open.
func (t *EmbossTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	skt := t.profiles[0].Sketch
	d, eng := t.depth, t.engrave
	t.added = feature.NewEmbossFeatures(part.Features()).Add(skt, profileIndicesOn(t.profiles, skt), func() float64 { return d }, eng, 0)
	part.Recompute()
	s.recordEdit(part, "Emboss")
	if !t.added.Health().OK() {
		return errors.New("emboss: " + t.added.Health().Reason)
	}
	s.Selection().SetFilter(NewSelectionFilter())
	return nil
}

// Cancel restores the default selection filter.
func (t *EmbossTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *EmbossTool) AddedFeature() *feature.PartFeature { return t.added }
