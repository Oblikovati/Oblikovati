// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// The sketch transform/pattern tools (Move, Copy, Rotate, Scale; Rectangular, Circular
// pattern) gather a variable selection set and apply a model edit with the tool's
// parameters on OK — they do not auto-commit. The viewport/dialog feeds the parameters
// through the Set* methods; the behavior lives in model/sketch and is tested headlessly
// here (the head dialog is a thin numeric-entry shell over these setters).

// sketchSelectTool is the shared selection collector: it is fed entity picks (PickSnap)
// and is ready once at least one entity is selected. Concrete tools embed it and add their
// parameters + Commit.
type sketchSelectTool struct {
	picks []sketch.Entity
}

func (t *sketchSelectTool) PickSnap(ent sketch.Entity, _ SnapResult) { t.picks = append(t.picks, ent) }
func (t *sketchSelectTool) Pick(*Session, Selectable)                {}
func (t *sketchSelectTool) Start(*Session)                           {}
func (t *sketchSelectTool) Cancel(*Session)                          { t.picks = nil }
func (t *sketchSelectTool) CanCommit() bool                          { return len(t.picks) > 0 }

// Selected returns the gathered entities (for the head/preview).
func (t *sketchSelectTool) Selected() []sketch.Entity { return t.picks }

// activeSketchOrErr resolves the active sketch or an error labelled with op.
func activeSketchOrErr(s *Session, op string) (*sketch.Sketch, error) {
	sk := s.ActiveSketch()
	if sk == nil {
		return nil, errors.New(op + ": no active sketch")
	}
	return sk, nil
}

// --- Move / Copy ----------------------------------------------------------

// SketchMoveTool translates the selection by a vector (set via SetVector or a base→target
// point pair).
type SketchMoveTool struct {
	sketchSelectTool
	vector math.Vector2
}

func NewSketchMoveTool() *SketchMoveTool { return &SketchMoveTool{} }
func (t *SketchMoveTool) Name() string   { return "Move" }
func (t *SketchMoveTool) Prompt(*Session) string {
	return "Select geometry, set the move vector, then OK."
}

// SetVector sets the displacement; SetByPoints derives it from a base and target point.
func (t *SketchMoveTool) SetVector(dx, dy float64) {
	t.vector = math.V2(math.Scalar(dx), math.Scalar(dy))
}
func (t *SketchMoveTool) SetByPoints(base, target math.Point2) { t.vector = base.VectorTo(target) }

func (t *SketchMoveTool) Commit(s *Session) error {
	sk, err := activeSketchOrErr(s, "move")
	if err != nil {
		return err
	}
	if err := rejectUnmoveablePicks(sk, t.picks, "move"); err != nil {
		return err
	}
	sk.MoveEntities(t.picks, t.vector)
	return nil
}

// rejectUnmoveablePicks refuses dragging geometry the solver classifies as
// not draggable, naming the entity and the precise reason (M06-F11, #626).
func rejectUnmoveablePicks(sk *sketch.Sketch, picks []sketch.Entity, op string) error {
	moveable := sk.MoveableClassifier()
	for _, e := range picks {
		switch moveable.Of(e) {
		case types.MoveableFixed:
			return fmt.Errorf("%s: entity %d is fixed (grounded or reference geometry) and cannot move", op, e.EntityID())
		case types.MoveableByDimensionChange:
			return fmt.Errorf("%s: entity %d is fully dimensioned; relax a driving dimension to move it", op, e.EntityID())
		}
	}
	return nil
}

// SketchCopyTool duplicates the selection, offset by a vector.
type SketchCopyTool struct {
	sketchSelectTool
	vector math.Vector2
}

func NewSketchCopyTool() *SketchCopyTool { return &SketchCopyTool{} }
func (t *SketchCopyTool) Name() string   { return "Copy" }
func (t *SketchCopyTool) Prompt(*Session) string {
	return "Select geometry, set the copy offset, then OK."
}

func (t *SketchCopyTool) SetVector(dx, dy float64) {
	t.vector = math.V2(math.Scalar(dx), math.Scalar(dy))
}
func (t *SketchCopyTool) SetByPoints(base, target math.Point2) { t.vector = base.VectorTo(target) }

func (t *SketchCopyTool) Commit(s *Session) error {
	sk, err := activeSketchOrErr(s, "copy")
	if err != nil {
		return err
	}
	if len(sk.CopyEntities(t.picks, t.vector)) == 0 {
		return errors.New("copy: produced no geometry")
	}
	return nil
}

// SketchStretchTool moves only the picked vertices (points) by a vector, deforming the
// geometry that shares them — Inventor's Stretch, where endpoints inside the window move
// and the rest stay. It collects point picks (a non-point pick is ignored).
type SketchStretchTool struct {
	points []*sketch.Point
	vector math.Vector2
}

func NewSketchStretchTool() *SketchStretchTool         { return &SketchStretchTool{} }
func (t *SketchStretchTool) Name() string              { return "Stretch" }
func (t *SketchStretchTool) Start(*Session)            {}
func (t *SketchStretchTool) Pick(*Session, Selectable) {}
func (t *SketchStretchTool) Cancel(*Session)           { t.points = nil }
func (t *SketchStretchTool) CanCommit() bool           { return len(t.points) > 0 }
func (t *SketchStretchTool) Prompt(*Session) string {
	return "Pick the vertices to stretch, set the vector, then OK."
}

// PickSnap collects point picks (vertices); non-point picks are ignored.
func (t *SketchStretchTool) PickSnap(ent sketch.Entity, _ SnapResult) {
	if p, ok := ent.(*sketch.Point); ok {
		t.points = append(t.points, p)
	}
}

// SetVector sets the displacement; SetByPoints derives it from a base and target point.
func (t *SketchStretchTool) SetVector(dx, dy float64) {
	t.vector = math.V2(math.Scalar(dx), math.Scalar(dy))
}

func (t *SketchStretchTool) SetByPoints(base, target math.Point2) { t.vector = base.VectorTo(target) }

func (t *SketchStretchTool) Commit(s *Session) error {
	sk, err := activeSketchOrErr(s, "stretch")
	if err != nil {
		return err
	}
	sk.MovePoints(t.points, t.vector)
	return nil
}

// --- Rotate / Scale -------------------------------------------------------

// SketchRotateTool rotates the selection about a center by an angle (radians).
type SketchRotateTool struct {
	sketchSelectTool
	center math.Point2
	angle  float64
}

func NewSketchRotateTool() *SketchRotateTool { return &SketchRotateTool{} }
func (t *SketchRotateTool) Name() string     { return "Rotate" }
func (t *SketchRotateTool) Prompt(*Session) string {
	return "Select geometry, set the center and angle, then OK."
}

func (t *SketchRotateTool) SetCenter(x, y float64) {
	t.center = math.P2(math.Scalar(x), math.Scalar(y))
}
func (t *SketchRotateTool) SetAngle(radians float64) { t.angle = radians }

func (t *SketchRotateTool) Commit(s *Session) error {
	sk, err := activeSketchOrErr(s, "rotate")
	if err != nil {
		return err
	}
	sk.RotateEntities(t.picks, t.center, t.angle)
	return nil
}

// SketchScaleTool scales the selection about a center by a factor.
type SketchScaleTool struct {
	sketchSelectTool
	center math.Point2
	factor float64
}

func NewSketchScaleTool() *SketchScaleTool { return &SketchScaleTool{factor: 1} }
func (t *SketchScaleTool) Name() string    { return "Scale" }
func (t *SketchScaleTool) Prompt(*Session) string {
	return "Select geometry, set the center and factor, then OK."
}

func (t *SketchScaleTool) SetCenter(x, y float64) {
	t.center = math.P2(math.Scalar(x), math.Scalar(y))
}
func (t *SketchScaleTool) SetFactor(f float64) { t.factor = f }

func (t *SketchScaleTool) Commit(s *Session) error {
	sk, err := activeSketchOrErr(s, "scale")
	if err != nil {
		return err
	}
	if t.factor <= 0 {
		return errors.New("scale: factor must be positive")
	}
	sk.ScaleEntities(t.picks, t.center, t.factor)
	return nil
}

// --- Rectangular / Circular pattern ---------------------------------------

// SketchRectPatternTool patterns the selection in a grid: count1 copies along step1 and
// count2 along step2 (each step is the full per-copy displacement vector).
type SketchRectPatternTool struct {
	sketchSelectTool
	step1  math.Vector2
	count1 int
	step2  math.Vector2
	count2 int
}

func NewSketchRectPatternTool() *SketchRectPatternTool {
	return &SketchRectPatternTool{count1: 2, count2: 1}
}
func (t *SketchRectPatternTool) Name() string { return "Rectangular Pattern" }
func (t *SketchRectPatternTool) Prompt(*Session) string {
	return "Select geometry, set the two directions and counts, then OK."
}

func (t *SketchRectPatternTool) SetStep1(dx, dy float64) {
	t.step1 = math.V2(math.Scalar(dx), math.Scalar(dy))
}

func (t *SketchRectPatternTool) SetStep2(dx, dy float64) {
	t.step2 = math.V2(math.Scalar(dx), math.Scalar(dy))
}
func (t *SketchRectPatternTool) SetCount1(n int) { t.count1 = n }
func (t *SketchRectPatternTool) SetCount2(n int) { t.count2 = n }

// CanCommit also requires the grid to make more than one instance.
func (t *SketchRectPatternTool) CanCommit() bool {
	return len(t.picks) > 0 && t.count1 >= 1 && t.count2 >= 1 && t.count1*t.count2 > 1
}

func (t *SketchRectPatternTool) Commit(s *Session) error {
	sk, err := activeSketchOrErr(s, "rectangular pattern")
	if err != nil {
		return err
	}
	_, err = sk.RectangularPattern(t.picks, t.step1, t.count1, t.step2, t.count2)
	return err
}

// SketchCircPatternTool patterns the selection around a center: count copies spread over
// totalAngle (radians).
type SketchCircPatternTool struct {
	sketchSelectTool
	center     math.Point2
	count      int
	totalAngle float64
}

func NewSketchCircPatternTool() *SketchCircPatternTool {
	return &SketchCircPatternTool{count: 4, totalAngle: 2 * stdmath.Pi}
}
func (t *SketchCircPatternTool) Name() string { return "Circular Pattern" }
func (t *SketchCircPatternTool) Prompt(*Session) string {
	return "Select geometry, set the center, count and angle, then OK."
}

func (t *SketchCircPatternTool) SetCenter(x, y float64) {
	t.center = math.P2(math.Scalar(x), math.Scalar(y))
}
func (t *SketchCircPatternTool) SetCount(n int)            { t.count = n }
func (t *SketchCircPatternTool) SetTotalAngle(rad float64) { t.totalAngle = rad }

func (t *SketchCircPatternTool) CanCommit() bool { return len(t.picks) > 0 && t.count >= 2 }

func (t *SketchCircPatternTool) Commit(s *Session) error {
	sk, err := activeSketchOrErr(s, "circular pattern")
	if err != nil {
		return err
	}
	_, err = sk.CircularPattern(t.picks, t.center, t.count, t.totalAngle)
	return err
}

// --- Params (generic property dialog) -------------------------------------

func (t *SketchMoveTool) Params() ToolParams {
	return ToolParams{Floats: xyParams("Δ", &t.vector.X, &t.vector.Y)}
}

func (t *SketchCopyTool) Params() ToolParams {
	return ToolParams{Floats: xyParams("Δ", &t.vector.X, &t.vector.Y)}
}

func (t *SketchStretchTool) Params() ToolParams {
	return ToolParams{Floats: xyParams("Δ", &t.vector.X, &t.vector.Y)}
}

func (t *SketchRotateTool) Params() ToolParams {
	return ToolParams{Floats: append(xyParams("Center", &t.center.X, &t.center.Y),
		FloatParam{"Angle (deg)", func() float64 { return degFromRad(t.angle) }, func(d float64) { t.angle = radFromDeg(d) }})}
}

func (t *SketchScaleTool) Params() ToolParams {
	return ToolParams{Floats: append(xyParams("Center", &t.center.X, &t.center.Y),
		FloatParam{"Factor", func() float64 { return t.factor }, func(f float64) { t.factor = f }})}
}

func (t *SketchRectPatternTool) Params() ToolParams {
	floats := append(xyParams("Step 1", &t.step1.X, &t.step1.Y), xyParams("Step 2", &t.step2.X, &t.step2.Y)...)
	return ToolParams{
		Floats: floats,
		Ints: []IntParam{
			{"Count 1", func() int { return t.count1 }, func(n int) { t.count1 = n }},
			{"Count 2", func() int { return t.count2 }, func(n int) { t.count2 = n }},
		},
	}
}

func (t *SketchCircPatternTool) Params() ToolParams {
	return ToolParams{
		Floats: append(xyParams("Center", &t.center.X, &t.center.Y),
			FloatParam{"Angle (deg)", func() float64 { return degFromRad(t.totalAngle) }, func(d float64) { t.totalAngle = radFromDeg(d) }}),
		Ints: []IntParam{{"Count", func() int { return t.count }, func(n int) { t.count = n }}},
	}
}
