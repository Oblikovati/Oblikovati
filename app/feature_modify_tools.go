// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/feature"
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

// MoveFaceTool translates one or more picked faces by a vector, retopologizing the solid.
type MoveFaceTool struct {
	faces      []FaceHandle
	dx, dy, dz float64
	added      *feature.PartFeature
}

func NewMoveFaceTool() *MoveFaceTool { return &MoveFaceTool{} }
func (t *MoveFaceTool) Name() string { return "Move Face" }
func (t *MoveFaceTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectFace))
}
func (t *MoveFaceTool) Pick(_ *Session, sel Selectable) {
	if f, ok := sel.(FaceHandle); ok {
		t.faces = append(t.faces, f)
	}
}

// Faces returns the picked faces for the unified tool highlight.
func (t *MoveFaceTool) Faces() []FaceHandle { return append([]FaceHandle(nil), t.faces...) }
func (t *MoveFaceTool) Cancel(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter())
	t.faces = nil
}
func (t *MoveFaceTool) Prompt(*Session) string {
	if len(t.faces) == 0 {
		return "Click the faces to move."
	}
	return "Set the move vector, then OK."
}
func (t *MoveFaceTool) CanCommit() bool {
	return len(t.faces) > 0 && (t.dx != 0 || t.dy != 0 || t.dz != 0)
}

func (t *MoveFaceTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	keys := make([][]byte, len(t.faces))
	for i, f := range t.faces {
		keys[i] = f.Face.ReferenceKey()
	}
	t.added = feature.NewModifyFeatures(part.Features()).AddMoveFace(keys, math.V3(math.Scalar(t.dx), math.Scalar(t.dy), math.Scalar(t.dz)))
	part.Recompute()
	s.recordEdit(part, "Move Face")
	s.Selection().SetFilter(NewSelectionFilter())
	if !t.added.Health().OK() {
		return errors.New("move face: " + t.added.Health().Reason)
	}
	return nil
}

func (t *MoveFaceTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{
		{"Δ X", func() float64 { return t.dx }, func(v float64) { t.dx = v }},
		{"Δ Y", func() float64 { return t.dy }, func(v float64) { t.dy = v }},
		{"Δ Z", func() float64 { return t.dz }, func(v float64) { t.dz = v }},
	}}
}

// --- Combine --------------------------------------------------------------

// CombineTool booleans two picked bodies (Join/Cut/Intersect): the first pick is the
// target, the second the tool body.
type CombineTool struct {
	bodies []*topo.Body
	op     ops.PartFeatureOperation
	added  *feature.PartFeature
}

func NewCombineTool() *CombineTool  { return &CombineTool{op: ops.Join} }
func (t *CombineTool) Name() string { return "Combine" }
func (t *CombineTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectBody))
}
func (t *CombineTool) Pick(_ *Session, sel Selectable) {
	if b, ok := sel.(BodyHandle); ok {
		t.bodies = append(t.bodies, b.Body)
	}
}
func (t *CombineTool) Cancel(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter())
	t.bodies = nil
}
func (t *CombineTool) Prompt(*Session) string {
	return "Pick the target body, then the tool body; set the operation; OK."
}
func (t *CombineTool) CanCommit() bool { return len(t.bodies) >= 2 }

func (t *CombineTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	ti, tj := bodyIndexOf(part, t.bodies[0]), bodyIndexOf(part, t.bodies[1])
	if ti < 0 || tj < 0 {
		return errors.New("combine: a picked body is not in the active part")
	}
	t.added = feature.NewModifyFeatures(part.Features()).AddCombine(ti, tj, t.op)
	part.Recompute()
	s.recordEdit(part, "Combine")
	s.Selection().SetFilter(NewSelectionFilter())
	if !t.added.Health().OK() {
		return errors.New("combine: " + t.added.Health().Reason)
	}
	return nil
}

// SetOperation chooses Join (0), Cut (1) or Intersect (2).
func (t *CombineTool) SetOperation(op ops.PartFeatureOperation) { t.op = op }

func (t *CombineTool) Params() ToolParams {
	return ToolParams{Ints: []IntParam{
		{"Operation (0=Join 1=Cut 2=Intersect)", func() int { return int(t.op) }, func(n int) { t.op = ops.PartFeatureOperation(n) }},
	}}
}

// --- Move (body) ----------------------------------------------------------

// MoveBodyTool relocates a picked body by a translation, preserving its reference keys.
type MoveBodyTool struct {
	bodies     []*topo.Body
	dx, dy, dz float64
	added      *feature.PartFeature
}

func NewMoveBodyTool() *MoveBodyTool { return &MoveBodyTool{} }
func (t *MoveBodyTool) Name() string { return "Move Bodies" }
func (t *MoveBodyTool) Start(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter(SelectBody))
}
func (t *MoveBodyTool) Pick(_ *Session, sel Selectable) {
	if b, ok := sel.(BodyHandle); ok {
		t.bodies = append(t.bodies, b.Body)
	}
}
func (t *MoveBodyTool) Cancel(s *Session) {
	s.Selection().SetFilter(NewSelectionFilter())
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
	part, err := activePart(s)
	if err != nil {
		return err
	}
	idx := bodyIndexOf(part, t.bodies[0])
	if idx < 0 {
		return errors.New("move: the picked body is not in the active part")
	}
	xf := math.Translation4(math.V3(math.Scalar(t.dx), math.Scalar(t.dy), math.Scalar(t.dz)))
	t.added = feature.NewModifyFeatures(part.Features()).AddMove(idx, xf)
	part.Recompute()
	s.recordEdit(part, "Move Bodies")
	s.Selection().SetFilter(NewSelectionFilter())
	if !t.added.Health().OK() {
		return errors.New("move: " + t.added.Health().Reason)
	}
	return nil
}

func (t *MoveBodyTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{
		{"Δ X", func() float64 { return t.dx }, func(v float64) { t.dx = v }},
		{"Δ Y", func() float64 { return t.dy }, func(v float64) { t.dy = v }},
		{"Δ Z", func() float64 { return t.dz }, func(v float64) { t.dz = v }},
	}}
}
