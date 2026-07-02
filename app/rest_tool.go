// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// RestTool is the interactive Rest command (#1076, plastic features): pick one or more closed
// sketch regions, set the depth and whether the rest is raised (a pad) or recessed (a pocket),
// then OK to add it to the active part. It exposes its parameters through ParameterizedTool, so
// the head renders the generic property dialog.
type RestTool struct {
	profiles []ProfileHandle
	depth    float64
	recessed bool
	added    *feature.PartFeature
}

// NewRestTool returns a rest tool defaulting to a 1-unit raised pad.
func NewRestTool() *RestTool { return &RestTool{depth: 1} }

// Name implements [Tool].
func (t *RestTool) Name() string { return "Rest" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *RestTool) Start(*Session) {}

// AcceptedKinds declares rest picks closed sketch regions (profiles).
func (t *RestTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectProfile} }

// Picks reports the picked regions for the unified highlight.
func (t *RestTool) Picks() []Selectable { return selectables(t.profiles) }

// Pick captures a single clicked region (replacing any previous selection).
func (t *RestTool) Pick(_ *Session, sel Selectable) {
	if p, ok := sel.(ProfileHandle); ok {
		t.profiles = []ProfileHandle{p}
	}
}

// PickWithMods extends the selection on Ctrl+click (toggling), to rest several regions at once.
func (t *RestTool) PickWithMods(s *Session, sel Selectable, mods Modifier) {
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

// The options the property window drives: depth (database units) and recessed (pocket vs pad).
func (t *RestTool) SetDepth(d float64) { t.depth = d }
func (t *RestTool) Depth() float64     { return t.depth }
func (t *RestTool) SetRecessed(v bool) { t.recessed = v }
func (t *RestTool) Recessed() bool     { return t.recessed }

// Params exposes the rest depth and recessed flag for the generic property dialog.
func (t *RestTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{{Label: "Depth", Get: t.Depth, Set: t.SetDepth}},
		Bools:  []BoolParam{{Label: "Recessed (pocket)", Get: t.Recessed, Set: t.SetRecessed}},
	}
}

// PickedProfiles returns the picked regions for the unified tool highlight.
func (t *RestTool) PickedProfiles() []ProfileHandle {
	return append([]ProfileHandle(nil), t.profiles...)
}

// CanCommit reports whether a region is picked and the depth is positive.
func (t *RestTool) CanCommit() bool { return len(t.profiles) > 0 && t.depth > 0 }

// Commit adds the rest to the active part and recomputes; a sick feature keeps the tool open.
func (t *RestTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addRest(part.Features())
	part.Recompute()
	s.recordEdit(part, "Rest")
	if !t.added.Health().OK() {
		return errors.New("rest: " + t.added.Health().Reason)
	}
	return nil
}

// addRest builds the rest feature into engine fs — shared by Commit and the preview.
func (t *RestTool) addRest(fs *feature.PartFeatures) *feature.PartFeature {
	skt := t.profiles[0].Sketch
	d, rec := t.depth, t.recessed
	return feature.NewPlasticFeatures(fs).AddRest(skt, profileIndicesOn(t.profiles, skt), func() float64 { return d }, rec, 0)
}

// DraftFeature returns the unattached rest feature the viewport previews before commit.
func (t *RestTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addRest(fs), nil
	})
}

// Cancel restores the default selection filter.
func (t *RestTool) Cancel(*Session) {}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *RestTool) AddedFeature() *feature.PartFeature { return t.added }
