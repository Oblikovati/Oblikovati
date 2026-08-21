// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// Modify-panel tools whose geometry already existed but had no UI: Move Face (a direct
// edit translating picked faces), Combine (boolean of two bodies) and Move (relocate a
// body). Each gathers its picks + parameters and adds the model feature on OK; the
// geometry is model-tested, so these are interaction shells. Parameters edit through the
// generic property dialog.

// bodyIndexOf returns the index of body b in the part's running body list, or -1.
func bodyIndexOf(part *compdef.PartComponentDefinition, b *topo.Body) int {
	for i, x := range part.SurfaceBodies().All() {
		if x == b {
			return i
		}
	}
	return -1
}

// --- Move Face ------------------------------------------------------------

// MoveFaceTool moves one or more picked faces, retopologizing the solid: by a translation
// vector, or — in rotate mode — about an axis through a point by an angle. AddMoveFaceRotate was
// implemented and routed over the API but the tool only ever called AddMoveFace, so rotating a
// face was API-only (#2050).
type MoveFaceTool struct {
	faces         []FaceHandle
	rotate        bool
	dx, dy, dz    float64 // translate mode: the move vector; rotate mode: the axis point
	axX, axY, axZ float64 // rotate mode: the axis direction (defaults to +Z)
	angle         float64 // rotate mode: the sweep, radians
	added         *feature.PartFeature
}

func NewMoveFaceTool() *MoveFaceTool   { return &MoveFaceTool{axZ: 1} }
func (t *MoveFaceTool) Name() string   { return "Move Face" }
func (t *MoveFaceTool) Start(*Session) {}

// AcceptedKinds declares move-face picks faces (the faces to move).
func (t *MoveFaceTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectFace} }

// Picks reports the picked faces for the unified highlight.
func (t *MoveFaceTool) Picks() []Selectable { return selectables(t.faces) }

func (t *MoveFaceTool) Pick(_ *Session, sel Selectable) {
	if f, ok := sel.(FaceHandle); ok {
		t.faces = append(t.faces, f)
	}
}

// Faces returns the picked faces for the unified tool highlight.
func (t *MoveFaceTool) Faces() []FaceHandle { return append([]FaceHandle(nil), t.faces...) }

func (t *MoveFaceTool) Cancel(s *Session) {
	t.faces = nil
}

func (t *MoveFaceTool) Prompt(*Session) string {
	if len(t.faces) == 0 {
		return "Click the faces to move."
	}
	if t.rotate {
		return "Set the axis and angle, then OK."
	}
	return "Set the move vector, then OK."
}

func (t *MoveFaceTool) CanCommit() bool {
	if len(t.faces) == 0 {
		return false
	}
	if t.rotate { // a zero angle or a degenerate axis would move nothing
		return t.angle != 0 && (t.axX != 0 || t.axY != 0 || t.axZ != 0)
	}
	return t.dx != 0 || t.dy != 0 || t.dz != 0
}

func (t *MoveFaceTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	t.added = t.addMoveFace(feature.NewModifyFeatures(part.Features()))
	part.Recompute()
	s.recordEdit(part, "Move Face")
	if !t.added.Health().OK() {
		return errors.New("move face: " + t.added.Health().Reason)
	}
	return nil
}

// addMoveFace builds the move-face edit into mods — the shared constructor used by both
// Commit (the part's engine) and DraftFeature (a scratch engine), so the two cannot drift.
func (t *MoveFaceTool) addMoveFace(mods *feature.ModifyFeatures) *feature.PartFeature {
	keys := make([][]byte, len(t.faces))
	for i, f := range t.faces {
		keys[i] = f.Face.ReferenceKey()
	}
	if t.rotate {
		a := t.angle
		return mods.AddMoveFaceRotate(keys,
			math.P3(t.dx, t.dy, t.dz), math.V3(math.Scalar(t.axX), math.Scalar(t.axY), math.Scalar(t.axZ)),
			func() float64 { return a })
	}
	return mods.AddMoveFace(keys, math.V3(math.Scalar(t.dx), math.Scalar(t.dy), math.Scalar(t.dz)))
}

// DraftFeature implements [PartFeatureTool] (#1626): the move-face edit it would commit,
// built into a scratch engine so the commit gate and preview can evaluate it without
// touching the part.
func (t *MoveFaceTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addMoveFace(feature.NewModifyFeatures(fs)), nil
	})
}

func (t *MoveFaceTool) Params() ToolParams {
	if t.rotate {
		return t.rotateParams()
	}
	return ToolParams{
		Bools: t.modeParam(),
		Floats: []FloatParam{
			{"Δ X", func() float64 { return t.dx }, func(v float64) { t.dx = v }},
			{"Δ Y", func() float64 { return t.dy }, func(v float64) { t.dy = v }},
			{"Δ Z", func() float64 { return t.dz }, func(v float64) { t.dz = v }},
		},
	}
}

// modeParam is the Rotate toggle both parameter sets share.
func (t *MoveFaceTool) modeParam() []BoolParam {
	return []BoolParam{{"Rotate", func() bool { return t.rotate }, func(v bool) { t.rotate = v }}}
}

// rotateParams are the rotate mode's inputs: the axis point (reusing the vector fields, which
// mean a position here), the axis direction, and the sweep in degrees.
func (t *MoveFaceTool) rotateParams() ToolParams {
	return ToolParams{
		Bools: t.modeParam(),
		Floats: []FloatParam{
			{"Axis X", func() float64 { return t.dx }, func(v float64) { t.dx = v }},
			{"Axis Y", func() float64 { return t.dy }, func(v float64) { t.dy = v }},
			{"Axis Z", func() float64 { return t.dz }, func(v float64) { t.dz = v }},
			{"Dir X", func() float64 { return t.axX }, func(v float64) { t.axX = v }},
			{"Dir Y", func() float64 { return t.axY }, func(v float64) { t.axY = v }},
			{"Dir Z", func() float64 { return t.axZ }, func(v float64) { t.axZ = v }},
			{"Angle°", func() float64 { return degFromRad(t.angle) }, func(v float64) { t.angle = v * radPerDeg }},
		},
	}
}

// --- Combine --------------------------------------------------------------

// CombineTool booleans picked bodies (Join/Cut/Intersect): the first pick is the target, every
// later pick a tool body. Keep-tool-bodies leaves the tools in the part afterwards (#2069).
type CombineTool struct {
	bodies    []*topo.Body
	op        ops.PartFeatureOperation
	keepTools bool // #2069: leave the tool bodies in the part after the boolean (KeepToolBodies)
	added     *feature.PartFeature
}

func NewCombineTool() *CombineTool    { return &CombineTool{op: ops.Join} }
func (t *CombineTool) Name() string   { return "Combine" }
func (t *CombineTool) Start(*Session) {}

// AcceptedKinds declares combine picks solid bodies (target then tool body).
func (t *CombineTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectBody} }

func (t *CombineTool) Pick(_ *Session, sel Selectable) {
	if b, ok := sel.(BodyHandle); ok {
		t.bodies = append(t.bodies, b.Body)
	}
}

func (t *CombineTool) Cancel(s *Session) {
	t.bodies = nil
}

func (t *CombineTool) Prompt(*Session) string {
	return "Pick the target body, then one or more tool bodies; set the operation; OK."
}
func (t *CombineTool) CanCommit() bool { return len(t.bodies) >= 2 }

func (t *CombineTool) Commit(s *Session) error {
	target, tools, err := t.combineOperands(s)
	if err != nil {
		return err
	}
	part, _ := activePart(s) // combineOperands already vetted the part
	t.added = t.addCombine(feature.NewModifyFeatures(part.Features()), target, tools)
	part.Recompute()
	s.recordEdit(part, "Combine")
	if !t.added.Health().OK() {
		return errors.New("combine: " + t.added.Health().Reason)
	}
	return nil
}

// combineOperands resolves the target (first pick) and every tool body (later picks) to indices in
// the active part's running body list — Commit and DraftFeature (#1626) resolve identically, so the
// gate inspects exactly what OK would build.
func (t *CombineTool) combineOperands(s *Session) (int, []int, error) {
	part, err := activePart(s)
	if err != nil {
		return 0, nil, err
	}
	target := bodyIndexOf(part, t.bodies[0])
	if target < 0 {
		return 0, nil, errors.New("combine: the target body is not in the active part")
	}
	tools := make([]int, 0, len(t.bodies)-1)
	for _, b := range t.bodies[1:] {
		j := bodyIndexOf(part, b)
		if j < 0 {
			return 0, nil, errors.New("combine: a picked tool body is not in the active part")
		}
		tools = append(tools, j)
	}
	return target, tools, nil
}

// addCombine builds the boolean into mods — the shared constructor used by both Commit
// (the part's engine) and DraftFeature (a scratch engine), so the two cannot drift. It booleans the
// target against every picked tool at once and honours keep-tool-bodies (#2069/#1894).
func (t *CombineTool) addCombine(mods *feature.ModifyFeatures, target int, tools []int) *feature.PartFeature {
	return mods.AddCombineTools(target, tools, t.op, t.keepTools)
}

// DraftFeature implements [PartFeatureTool] (#1626): the boolean it would commit, built
// into a scratch engine so the commit gate and preview can evaluate it without touching
// the part. The body operands resolve against the session at draft time, exactly as
// Commit does; unresolved operands mean the draft is not ready.
func (t *CombineTool) DraftFeature(s *Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	target, tools, err := t.combineOperands(s)
	if err != nil {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addCombine(feature.NewModifyFeatures(fs), target, tools), nil
	})
}

// SetOperation chooses Join (0), Cut (1) or Intersect (2).
func (t *CombineTool) SetOperation(op ops.PartFeatureOperation) { t.op = op }

// KeepTools / SetKeepTools leave the tool bodies in the part after the boolean instead of consuming
// them (Inventor's KeepToolBodies, #2069/#1894), so a tool can go on to cut something else.
func (t *CombineTool) KeepTools() bool     { return t.keepTools }
func (t *CombineTool) SetKeepTools(v bool) { t.keepTools = v }

// Params exposes the boolean operation as a named dropdown. It was an IntParam whose long
// self-documenting label ("Operation (0=Join 1=Cut 2=Intersect)") overflowed the 95px label
// column and collided with the InputInt steppers, rendering as illegible garble (#1803). A
// ChoiceParam puts the short label in the column and the named options in the control.
func (t *CombineTool) Params() ToolParams {
	return ToolParams{
		Choices: []ChoiceParam{
			{Label: "Operation", Options: []string{"Join", "Cut", "Intersect"},
				Get: func() int { return int(t.op) },
				Set: func(n int) { t.op = ops.PartFeatureOperation(n) }},
		},
		Bools: []BoolParam{{Label: "Keep tool bodies", Get: t.KeepTools, Set: t.SetKeepTools}},
	}
}

// --- Move (body) ----------------------------------------------------------

// MoveBodyTool relocates a picked body by a translation, preserving its reference keys.
type MoveBodyTool struct {
	bodies     []*topo.Body
	dx, dy, dz float64
	added      *feature.PartFeature
}

func NewMoveBodyTool() *MoveBodyTool   { return &MoveBodyTool{} }
func (t *MoveBodyTool) Name() string   { return "Move Bodies" }
func (t *MoveBodyTool) Start(*Session) {}

// AcceptedKinds declares move-bodies picks solid bodies (the bodies to move).
func (t *MoveBodyTool) AcceptedKinds() []SelectionKind { return []SelectionKind{SelectBody} }

func (t *MoveBodyTool) Pick(_ *Session, sel Selectable) {
	if b, ok := sel.(BodyHandle); ok {
		t.bodies = append(t.bodies, b.Body)
	}
}

func (t *MoveBodyTool) Cancel(s *Session) {
	t.bodies = nil
}

func (t *MoveBodyTool) Prompt(*Session) string {
	if len(t.bodies) == 0 {
		return "Pick a body to move."
	}
	return "Set the move vector, then OK."
}

func (t *MoveBodyTool) CanCommit() bool {
	return len(t.bodies) > 0 && (t.dx != 0 || t.dy != 0 || t.dz != 0)
}

func (t *MoveBodyTool) Commit(s *Session) error {
	idx, err := t.movedBodyIndex(s)
	if err != nil {
		return err
	}
	part, _ := activePart(s) // movedBodyIndex already vetted the part
	t.added = t.addMove(feature.NewModifyFeatures(part.Features()), idx)
	part.Recompute()
	s.recordEdit(part, "Move Bodies")
	if !t.added.Health().OK() {
		return errors.New("move: " + t.added.Health().Reason)
	}
	return nil
}

// movedBodyIndex resolves the picked body to its index in the active part's running body
// list — Commit and DraftFeature (#1626) resolve identically, so the gate inspects
// exactly what OK would build.
func (t *MoveBodyTool) movedBodyIndex(s *Session) (int, error) {
	part, err := activePart(s)
	if err != nil {
		return -1, err
	}
	idx := bodyIndexOf(part, t.bodies[0])
	if idx < 0 {
		return -1, errors.New("move: the picked body is not in the active part")
	}
	return idx, nil
}

// addMove builds the body move into mods — the shared constructor used by both Commit
// (the part's engine) and DraftFeature (a scratch engine), so the two cannot drift.
func (t *MoveBodyTool) addMove(mods *feature.ModifyFeatures, bodyIndex int) *feature.PartFeature {
	xf := math.Translation4(math.V3(math.Scalar(t.dx), math.Scalar(t.dy), math.Scalar(t.dz)))
	return mods.AddMove(bodyIndex, xf)
}

// DraftFeature implements [PartFeatureTool] (#1626): the body move it would commit,
// built into a scratch engine so the commit gate and preview can evaluate it without
// touching the part. The body operand resolves against the session at draft time,
// exactly as Commit does; an unresolved operand means the draft is not ready.
func (t *MoveBodyTool) DraftFeature(s *Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	idx, err := t.movedBodyIndex(s)
	if err != nil {
		return nil, false
	}
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addMove(feature.NewModifyFeatures(fs), idx), nil
	})
}

func (t *MoveBodyTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{
		{"Δ X", func() float64 { return t.dx }, func(v float64) { t.dx = v }},
		{"Δ Y", func() float64 { return t.dy }, func(v float64) { t.dy = v }},
		{"Δ Z", func() float64 { return t.dz }, func(v float64) { t.dz = v }},
	}}
}
