// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// The M10-F03 freeform-primitive tool shells (#698): Box, Plane and Quad Ball place a
// sub-D control cage evaluated at a Catmull–Clark level (kernel/subd). They are
// parameter-only — set sizes and the level in the generic tool dialog, then OK; cage
// editing on the placed feature is the freeform.* wire surface (#699).

// freeformCommit adds the primitive through add and reports a sick result as an error —
// the shared back half of the three freeform tools.
func freeformCommit(s *Session, label string, add func(*feature.FreeformFeatures) *feature.PartFeature) (*feature.PartFeature, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	added := add(feature.NewFreeformFeatures(part.Features()))
	part.Recompute()
	s.recordEdit(part, label)
	if !added.Health().OK() {
		return added, errors.New(label + ": " + added.Health().Reason)
	}
	return added, nil
}

// freeformDraft builds the primitive through add into a scratch engine — the shared back half
// of the three freeform DraftFeature implementations (#1626), mirroring freeformCommit so the
// commit gate and preview evaluate the exact feature OK would place.
func freeformDraft(add func(*feature.FreeformFeatures) *feature.PartFeature) (feature.Feature, bool) {
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return add(feature.NewFreeformFeatures(fs)), nil
	})
}

// levelParam is the shared subdivision-level descriptor of the freeform tools.
func levelParam(get func() int, set func(int)) IntParam {
	return IntParam{Label: "Level", Get: get, Set: set}
}

// FreeformBoxTool places a sub-D box primitive (sx × sy × sz cage at a subdivision level).
type FreeformBoxTool struct {
	dialogTool
	sx, sy, sz float64
	level      int
	added      *feature.PartFeature
}

// NewFreeformBoxTool returns a box tool defaulting to a 4×4×4 cage at level 1.
func NewFreeformBoxTool() *FreeformBoxTool { return &FreeformBoxTool{sx: 4, sy: 4, sz: 4, level: 1} }

// Name implements [Tool].
func (t *FreeformBoxTool) Name() string { return "Freeform Box" }

// Prompt guides the input.
func (t *FreeformBoxTool) Prompt(*Session) string {
	return "Set the box sizes and subdivision level, then OK."
}

// Start/Pick implement [Tool] (parameter-only, nothing is picked).

// Params exposes the cage sizes and level for the generic property dialog.
func (t *FreeformBoxTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{
			{Label: "Length", Get: func() float64 { return t.sx }, Set: func(v float64) { t.sx = v }},
			{Label: "Width", Get: func() float64 { return t.sy }, Set: func(v float64) { t.sy = v }},
			{Label: "Height", Get: func() float64 { return t.sz }, Set: func(v float64) { t.sz = v }},
		},
		Ints: []IntParam{levelParam(func() int { return t.level }, func(v int) { t.level = v })},
	}
}

// CanCommit reports whether every size is positive (level clamps at 0 in the model).
func (t *FreeformBoxTool) CanCommit() bool { return t.sx > 0 && t.sy > 0 && t.sz > 0 }

// addBoxCage runs the box construction against ff — shared by Commit (the part's engine) and
// DraftFeature (a scratch engine) so the committed and previewed features cannot drift (#1626).
func (t *FreeformBoxTool) addBoxCage(ff *feature.FreeformFeatures) *feature.PartFeature {
	return ff.AddBox(t.sx, t.sy, t.sz, t.level)
}

// Commit places the box cage and recomputes.
func (t *FreeformBoxTool) Commit(s *Session) error {
	added, err := freeformCommit(s, "Freeform Box", t.addBoxCage)
	t.added = added
	return err
}

// DraftFeature implements [PartFeatureTool] (#1626): the box cage it would commit, built into a
// scratch engine so the commit gate and preview can evaluate it without touching the part.
func (t *FreeformBoxTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return freeformDraft(t.addBoxCage)
}

// Cancel implements [Tool] (nothing to restore).

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *FreeformBoxTool) AddedFeature() *feature.PartFeature { return t.added }

// FreeformPlaneTool places a sub-D plane primitive (an open sx × sy cage — a surface body).
type FreeformPlaneTool struct {
	dialogTool
	sx, sy float64
	level  int
	added  *feature.PartFeature
}

// NewFreeformPlaneTool returns a plane tool defaulting to a 4×4 cage at level 1.
func NewFreeformPlaneTool() *FreeformPlaneTool { return &FreeformPlaneTool{sx: 4, sy: 4, level: 1} }

// Name implements [Tool].
func (t *FreeformPlaneTool) Name() string { return "Freeform Plane" }

// Prompt guides the input.
func (t *FreeformPlaneTool) Prompt(*Session) string {
	return "Set the plane sizes and subdivision level, then OK."
}

// Start/Pick implement [Tool] (parameter-only).

// Params exposes the cage sizes and level for the generic property dialog.
func (t *FreeformPlaneTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{
			{Label: "Length", Get: func() float64 { return t.sx }, Set: func(v float64) { t.sx = v }},
			{Label: "Width", Get: func() float64 { return t.sy }, Set: func(v float64) { t.sy = v }},
		},
		Ints: []IntParam{levelParam(func() int { return t.level }, func(v int) { t.level = v })},
	}
}

// CanCommit reports whether both sizes are positive.
func (t *FreeformPlaneTool) CanCommit() bool { return t.sx > 0 && t.sy > 0 }

// addPlaneCage runs the plane construction against ff — shared by Commit and DraftFeature so
// the committed and previewed features cannot drift (#1626).
func (t *FreeformPlaneTool) addPlaneCage(ff *feature.FreeformFeatures) *feature.PartFeature {
	return ff.AddPlane(t.sx, t.sy, t.level)
}

// Commit places the plane cage and recomputes.
func (t *FreeformPlaneTool) Commit(s *Session) error {
	added, err := freeformCommit(s, "Freeform Plane", t.addPlaneCage)
	t.added = added
	return err
}

// DraftFeature implements [PartFeatureTool] (#1626): the plane cage it would commit, built into
// a scratch engine so the commit gate and preview can evaluate it without touching the part.
func (t *FreeformPlaneTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return freeformDraft(t.addPlaneCage)
}

// Cancel implements [Tool] (nothing to restore).

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *FreeformPlaneTool) AddedFeature() *feature.PartFeature { return t.added }

// FreeformQuadBallTool places a sub-D quad-ball primitive (a closed sphere-like cage).
type FreeformQuadBallTool struct {
	dialogTool
	radius float64
	level  int
	added  *feature.PartFeature
}

// NewFreeformQuadBallTool returns a quad-ball tool defaulting to radius 2 at level 1.
func NewFreeformQuadBallTool() *FreeformQuadBallTool {
	return &FreeformQuadBallTool{radius: 2, level: 1}
}

// Name implements [Tool].
func (t *FreeformQuadBallTool) Name() string { return "Freeform Quad Ball" }

// Prompt guides the input.
func (t *FreeformQuadBallTool) Prompt(*Session) string {
	return "Set the radius and subdivision level, then OK."
}

// Start/Pick implement [Tool] (parameter-only).

// Params exposes the radius and level for the generic property dialog.
func (t *FreeformQuadBallTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{{
			Label: "Radius",
			Get:   func() float64 { return t.radius },
			Set:   func(v float64) { t.radius = v },
		}},
		Ints: []IntParam{levelParam(func() int { return t.level }, func(v int) { t.level = v })},
	}
}

// CanCommit reports whether the radius is positive.
func (t *FreeformQuadBallTool) CanCommit() bool { return t.radius > 0 }

// addQuadBallCage runs the quad-ball construction against ff — shared by Commit and
// DraftFeature so the committed and previewed features cannot drift (#1626).
func (t *FreeformQuadBallTool) addQuadBallCage(ff *feature.FreeformFeatures) *feature.PartFeature {
	return ff.AddQuadBall(t.radius, t.level)
}

// Commit places the quad-ball cage and recomputes.
func (t *FreeformQuadBallTool) Commit(s *Session) error {
	added, err := freeformCommit(s, "Freeform Quad Ball", t.addQuadBallCage)
	t.added = added
	return err
}

// DraftFeature implements [PartFeatureTool] (#1626): the quad-ball cage it would commit, built
// into a scratch engine so the commit gate and preview can evaluate it without touching the part.
func (t *FreeformQuadBallTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return freeformDraft(t.addQuadBallCage)
}

// Cancel implements [Tool] (nothing to restore).

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *FreeformQuadBallTool) AddedFeature() *feature.PartFeature { return t.added }
