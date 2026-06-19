// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// Tool shells for the part-modeling features that were model- and API-complete but had
// no ribbon entry (#702): Boss, Direct Edit, Hull and the Sketch-Driven Pattern. The
// geometry is model-tested; these gather picks + parameters headlessly and add the
// feature on OK.

// --- Boss -------------------------------------------------------------------

// BossTool raises a cylindrical stud on a picked planar face (the join-side mirror of the
// drilled hole, #327): click the face, set diameter and height, OK.
type BossTool struct {
	face     *FaceHandle
	diameter float64
	height   float64
	added    *feature.PartFeature
}

// NewBossTool returns a boss tool with a default Ø1 × 1 stud.
func NewBossTool() *BossTool { return &BossTool{diameter: 1, height: 1} }

// Name implements [Tool].
func (t *BossTool) Name() string { return "Boss" }

// Start filters selection to faces so clicks pick the placement face.
func (t *BossTool) Start(s *Session) { s.Selection().SetFilter(NewSelectionFilter(SelectFace)) }

// Pick captures the planar face the stud grows from.
func (t *BossTool) Pick(_ *Session, sel Selectable) {
	if f, ok := sel.(FaceHandle); ok {
		fc := f
		t.face = &fc
	}
}

// Faces returns the picked face for the unified tool highlight.
func (t *BossTool) Faces() []FaceHandle {
	if t.face == nil {
		return nil
	}
	return []FaceHandle{*t.face}
}

// Prompt guides the pick.
func (t *BossTool) Prompt(*Session) string {
	if t.face == nil {
		return "Click the face to raise the boss from."
	}
	return "Set the diameter and height, then OK."
}

// Params exposes the stud dimensions for the generic property dialog.
func (t *BossTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{
		{Label: "Diameter", Get: func() float64 { return t.diameter }, Set: func(v float64) { t.diameter = v }},
		{Label: "Height", Get: func() float64 { return t.height }, Set: func(v float64) { t.height = v }},
	}}
}

// CanCommit reports whether a face is picked and the stud has positive dimensions.
func (t *BossTool) CanCommit() bool { return t.face != nil && t.diameter > 0 && t.height > 0 }

// Commit raises the stud and recomputes.
func (t *BossTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	d, h := t.diameter, t.height
	t.added = feature.NewBossFeatures(part.Features()).
		Add(t.face.Face.ReferenceKey(), func() float64 { return d }, func() float64 { return h })
	part.Recompute()
	s.recordEdit(part, "Boss")
	s.Selection().SetFilter(NewSelectionFilter())
	if !t.added.Health().OK() {
		return errors.New("boss: " + t.added.Health().Reason)
	}
	return nil
}

// Cancel restores the default selection filter.
func (t *BossTool) Cancel(s *Session) { s.Selection().SetFilter(NewSelectionFilter()) }

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *BossTool) AddedFeature() *feature.PartFeature { return t.added }

// --- Hull -------------------------------------------------------------------

// HullTool wraps the running solids into their single convex hull — no inputs, just OK.
type HullTool struct {
	dialogTool
	added *feature.PartFeature
}

// NewHullTool returns a hull tool.
func NewHullTool() *HullTool { return &HullTool{} }

// Name implements [Tool].
func (t *HullTool) Name() string { return "Hull" }

// Prompt explains the (input-free) action.
func (t *HullTool) Prompt(*Session) string {
	return "OK hulls the part's solids into one convex solid."
}

// Start/Pick implement [Tool] (no inputs).

// CanCommit is always true — the running solids are the input.
func (t *HullTool) CanCommit() bool { return true }

// Commit hulls the running solids and recomputes.
func (t *HullTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewHullFeatures(part.Features()).Add()
	part.Recompute()
	s.recordEdit(part, "Hull")
	if !t.added.Health().OK() {
		return errors.New("hull: " + t.added.Health().Reason)
	}
	return nil
}

// Cancel implements [Tool] (nothing to restore).

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *HullTool) AddedFeature() *feature.PartFeature { return t.added }

// --- Sketch-Driven Pattern ----------------------------------------------------

// FeatureSketchDrivenPatternTool replicates picked source features at a picked sketch's
// points (occurrence per point, relative to the first).
type FeatureSketchDrivenPatternTool struct {
	featureSelectTool
	sketch *sketch.Sketch
}

// NewFeatureSketchDrivenPatternTool returns a sketch-driven pattern tool.
func NewFeatureSketchDrivenPatternTool() *FeatureSketchDrivenPatternTool {
	return &FeatureSketchDrivenPatternTool{}
}

// Name implements [Tool].
func (t *FeatureSketchDrivenPatternTool) Name() string { return "Sketch-Driven Pattern" }

// Prompt guides the two pick stages.
func (t *FeatureSketchDrivenPatternTool) Prompt(*Session) string {
	if len(t.sources) == 0 {
		return "Select the features to pattern."
	}
	if t.sketch == nil {
		return "Select the sketch whose points drive the placements."
	}
	return "OK places one occurrence per sketch point."
}

// Pick collects feature picks and the driving sketch (browser picks).
func (t *FeatureSketchDrivenPatternTool) Pick(s *Session, sel Selectable) {
	if sk, ok := sel.(SketchHandle); ok {
		t.sketch = sk.Sketch
		return
	}
	t.featureSelectTool.Pick(s, sel)
}

// CanCommit reports whether sources and a sketch with points are picked.
func (t *FeatureSketchDrivenPatternTool) CanCommit() bool {
	return len(t.sources) > 0 && t.sketch != nil && t.sketch.Points().Count() > 0
}

// Commit places one occurrence per sketch point and recomputes. The points closure
// re-reads the live sketch, so editing the sketch re-drives the pattern.
func (t *FeatureSketchDrivenPatternTool) Commit(s *Session) error {
	pats, err := t.patternFeatures(s, "sketch-driven pattern")
	if err != nil {
		return err
	}
	if t.sketch == nil {
		return errors.New("sketch-driven pattern: select the driving sketch")
	}
	sk := t.sketch
	pats.AddSketchDriven(t.sources, func() []math.Point3 { return sketchWorldPoints(sk) })
	return t.finishPattern(s, lastPartFeature(s), "Sketch-Driven Pattern")
}

// sketchWorldPoints maps a sketch's points to model space through its plane.
func sketchWorldPoints(sk *sketch.Sketch) []math.Point3 {
	pts := make([]math.Point3, sk.Points().Count())
	for i := range pts {
		pts[i] = sk.Plane().ToModel(sk.Points().Item(i).Position())
	}
	return pts
}

// --- Direct Edit ---------------------------------------------------------------

// directEditOperationNames are the operation choices in commit-dispatch order.
var directEditOperationNames = []string{"Move", "Size", "Rotate", "Delete", "Scale"}

// directEditOperations maps the choice index to the frozen operation id.
var directEditOperations = []types.DirectEditOperationType{
	types.DirectEditMoveOperation, types.DirectEditSizeOperation,
	types.DirectEditRotateOperation, types.DirectEditDeleteOperation,
	types.DirectEditScaleOperation,
}

// DirectEditTool applies one consolidated direct-edit operation (#332) to picked faces
// (or the whole body, for Scale): choose the operation, pick the faces, set the
// operation's inputs, OK. Vector is the translation (Move), push direction (Size) or
// rotation axis (Rotate); Point is the axis point (Rotate) or scale base (Scale).
type DirectEditTool struct {
	faces    []FaceHandle
	op       int
	vec      [3]float64
	point    [3]float64
	distance float64
	angleDeg float64
	scale    float64
	added    *feature.PartFeature
}

// NewDirectEditTool returns a direct-edit tool defaulting to a unit +Z move.
func NewDirectEditTool() *DirectEditTool {
	return &DirectEditTool{vec: [3]float64{0, 0, 1}, distance: 1, angleDeg: 15, scale: 1.5}
}

// Name implements [Tool].
func (t *DirectEditTool) Name() string { return "Direct Edit" }

// Start filters selection to faces.
func (t *DirectEditTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectFace))
}

// Pick collects the faces the operation acts on.
func (t *DirectEditTool) Pick(_ *Session, sel Selectable) {
	if f, ok := sel.(FaceHandle); ok {
		t.faces = append(t.faces, f)
	}
}

// Faces returns the picked faces for the unified tool highlight.
func (t *DirectEditTool) Faces() []FaceHandle { return append([]FaceHandle(nil), t.faces...) }

// Prompt guides the pick.
func (t *DirectEditTool) Prompt(*Session) string {
	if len(t.faces) == 0 && t.operation() != types.DirectEditScaleOperation {
		return "Click the faces to edit, choose the operation, then OK."
	}
	return "Set the operation inputs, then OK."
}

// operation returns the frozen id of the selected operation.
func (t *DirectEditTool) operation() types.DirectEditOperationType {
	return directEditOperations[t.op]
}

// SetOperation selects the operation by choice index (for tests and the dialog).
func (t *DirectEditTool) SetOperation(i int) {
	if i >= 0 && i < len(directEditOperations) {
		t.op = i
	}
}

// Params exposes the operation choice and the per-operation inputs.
func (t *DirectEditTool) Params() ToolParams {
	return ToolParams{
		Floats:  t.floatParams(),
		Choices: []ChoiceParam{{Label: "Operation", Options: directEditOperationNames, Get: func() int { return t.op }, Set: t.SetOperation}},
	}
}

// floatParams are the operation inputs; the labels say which operation reads each.
func (t *DirectEditTool) floatParams() []FloatParam {
	f := func(p *float64, label string) FloatParam {
		return FloatParam{Label: label, Get: func() float64 { return *p }, Set: func(v float64) { *p = v }}
	}
	return []FloatParam{
		f(&t.vec[0], "Vector X"), f(&t.vec[1], "Vector Y"), f(&t.vec[2], "Vector Z"),
		f(&t.point[0], "Point X"), f(&t.point[1], "Point Y"), f(&t.point[2], "Point Z"),
		f(&t.distance, "Distance"), f(&t.angleDeg, "Angle (deg)"), f(&t.scale, "Scale factor"),
	}
}

// CanCommit reports whether the selected operation has its inputs (Scale needs no faces).
func (t *DirectEditTool) CanCommit() bool {
	if t.operation() == types.DirectEditScaleOperation {
		return t.scale > 0
	}
	return len(t.faces) > 0
}

// Commit applies the direct edit and recomputes.
func (t *DirectEditTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = feature.NewModifyFeatures(part.Features()).AddDirectEdit(t.definition())
	part.Recompute()
	s.recordEdit(part, "Direct Edit")
	s.Selection().SetFilter(NewSelectionFilter())
	if !t.added.Health().OK() {
		return errors.New("direct edit: " + t.added.Health().Reason)
	}
	return nil
}

// definition assembles the recipe for the selected operation from the tool inputs.
func (t *DirectEditTool) definition() *feature.DirectEditDefinition {
	keys := make([][]byte, len(t.faces))
	for i, f := range t.faces {
		keys[i] = f.Face.ReferenceKey()
	}
	def := &feature.DirectEditDefinition{Operation: t.operation(), FaceKeys: keys}
	t.fillOperationInputs(def)
	return def
}

// fillOperationInputs copies the vector/point/scalar inputs the operation reads.
func (t *DirectEditTool) fillOperationInputs(def *feature.DirectEditDefinition) {
	vec := math.V3(math.Scalar(t.vec[0]), math.Scalar(t.vec[1]), math.Scalar(t.vec[2]))
	pt := math.P3(math.Scalar(t.point[0]), math.Scalar(t.point[1]), math.Scalar(t.point[2]))
	d, a, k := t.distance, t.angleDeg*stdmath.Pi/180, t.scale
	switch def.Operation {
	case types.DirectEditMoveOperation:
		def.Translation = vec
	case types.DirectEditSizeOperation:
		def.Direction, def.Distance = vec, func() float64 { return d }
	case types.DirectEditRotateOperation:
		def.AxisPoint, def.AxisDir, def.Angle = pt, vec, func() float64 { return a }
	case types.DirectEditScaleOperation:
		def.BasePoint, def.ScaleFactor = pt, func() float64 { return k }
	}
}

// Cancel restores the default selection filter.
func (t *DirectEditTool) Cancel(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter())
	t.faces = nil
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *DirectEditTool) AddedFeature() *feature.PartFeature { return t.added }
