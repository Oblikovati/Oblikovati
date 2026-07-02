// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/sketch"

// Constraint and dimension tools follow Inventor's tool-first flow: the user activates
// the tool, then picks the sketch geometry. Each tool accepts only the entity kinds
// valid for its constraint (so the head can highlight allowed geometry on hover) and
// auto-applies once enough valid entities are picked, then deactivates.

// SketchEntityTool is a sketch tool driven by picking sketch entities (constraints,
// dimensions). While it is active, viewport clicks pick sketch geometry and feed it to
// the tool together with the snap under the cursor; Accepts reports which entities are
// valid picks (the candidate-highlight cue), and Picked is what has been chosen so far.
type SketchEntityTool interface {
	Accepts(e sketch.Entity) bool
	Picked() []sketch.Entity
	PickSnap(ent sketch.Entity, snap SnapResult)
}

// constraintPick is one picked entity plus the snap under the cursor when it was picked
// (so e.g. coincident can tell a midpoint snap from a plain on-line snap).
type constraintPick struct {
	entity sketch.Entity
	snap   SnapResult
}

// ConstraintTool is the generic interactive constraint/dimension tool: it gathers
// accepted picks and applies its constraint when ready.
type ConstraintTool struct {
	dialogTool
	name    string
	prompt  string
	accepts func(sketch.Entity) bool
	ready   func([]sketch.Entity) bool
	apply   func(*Session, []constraintPick) error
	picks   []constraintPick
}

// Name/Start/Cancel implement [Tool].
func (t *ConstraintTool) Name() string    { return t.name }
func (t *ConstraintTool) Cancel(*Session) { t.picks = nil }

// Pick records a clicked entity with no snap context (the generic-tool path).
func (t *ConstraintTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(SketchEntityHandle); ok {
		t.PickSnap(h.Entity, SnapResult{})
	}
}

// PickSnap records a valid, not-already-picked entity together with its snap.
func (t *ConstraintTool) PickSnap(ent sketch.Entity, snap SnapResult) {
	if !t.accepts(ent) || t.contains(ent) {
		return
	}
	t.picks = append(t.picks, constraintPick{entity: ent, snap: snap})
}

// CanCommit is true once enough valid entities have been picked.
func (t *ConstraintTool) CanCommit() bool { return t.ready(entitiesOf(t.picks)) }

// AutoCommitOnPick applies the constraint as soon as it is ready (then deactivates).
func (t *ConstraintTool) AutoCommitOnPick() bool { return true }

// Commit applies the constraint to the picked entities.
func (t *ConstraintTool) Commit(s *Session) error { return t.apply(s, t.picks) }

// Accepts/Picked implement [SketchEntityTool] (hover hints + picked highlight).
func (t *ConstraintTool) Accepts(e sketch.Entity) bool { return t.accepts(e) }
func (t *ConstraintTool) Picked() []sketch.Entity      { return entitiesOf(t.picks) }

// Prompt guides the user (Inventor's status-bar prompt).
func (t *ConstraintTool) Prompt(*Session) string { return t.prompt }

func (t *ConstraintTool) contains(e sketch.Entity) bool {
	for _, p := range t.picks {
		if p.entity == e {
			return true
		}
	}
	return false
}

// entitiesOf extracts the entities from a pick list (for the accepts/ready/entity-apply
// helpers that don't need the snap).
func entitiesOf(picks []constraintPick) []sketch.Entity {
	out := make([]sketch.Entity, len(picks))
	for i, p := range picks {
		out[i] = p.entity
	}
	return out
}

// entityApply adapts an entity-based apply function to the pick-based tool signature.
func entityApply(f func(*Session, []sketch.Entity) error) func(*Session, []constraintPick) error {
	return func(s *Session, picks []constraintPick) error { return f(s, entitiesOf(picks)) }
}

// applyCoincidentSnap is the Coincident tool's apply: a midpoint snap on a line picks
// the midpoint constraint; otherwise it falls back to point-to-point / point-on-curve.
func applyCoincidentSnap(s *Session, picks []constraintPick) error {
	ents := entitiesOf(picks)
	pts, lines := filterPoints(ents), filterLines(ents)
	if len(pts) >= 1 && len(lines) >= 1 && midpointPicked(picks) {
		s.geom().AddMidpoint(pts[0], lines[0])
		return s.afterConstraint()
	}
	return applyCoincident(s, ents)
}

// midpointPicked reports whether a line was picked at its midpoint snap.
func midpointPicked(picks []constraintPick) bool {
	for _, p := range picks {
		if _, ok := p.entity.(*sketch.Line); ok && p.snap.Kind == SnapMidpoint {
			return true
		}
	}
	return false
}

// Entity-kind acceptance predicates.
func acceptPoints(e sketch.Entity) bool  { _, ok := e.(*sketch.Point); return ok }
func acceptLines(e sketch.Entity) bool   { _, ok := e.(*sketch.Line); return ok }
func acceptCircles(e sketch.Entity) bool { _, ok := e.(*sketch.Circle); return ok }

// acceptCircular accepts any circular curve — a circle or an arc.
func acceptCircular(e sketch.Entity) bool {
	_, ok := e.(sketch.CircularCurve)
	return ok
}

func acceptLinesOrPoints(e sketch.Entity) bool {
	return acceptLines(e) || acceptPoints(e)
}

func acceptLineOrCircular(e sketch.Entity) bool {
	return acceptLines(e) || acceptCircular(e)
}

// acceptCoincident accepts the geometry the polymorphic Coincident tool can pick: a
// point, line, circle, or arc.
func acceptCoincident(e sketch.Entity) bool {
	return acceptPoints(e) || acceptLines(e) || acceptCircular(e)
}

func acceptDimensionable(e sketch.Entity) bool {
	return acceptPoints(e) || acceptLines(e) || acceptCircles(e)
}

// readyCoincident is satisfied by two points, or a point plus a line/circle/arc.
func readyCoincident(ents []sketch.Entity) bool {
	pts := len(filterPoints(ents))
	if pts >= 2 {
		return true
	}
	return pts >= 1 && (len(filterLines(ents)) >= 1 || len(circularCurvesFrom(ents)) >= 1)
}

// Readiness predicates over the picked entities.
func ready2Lines(ents []sketch.Entity) bool    { return len(filterLines(ents)) >= 2 }
func ready2Circular(ents []sketch.Entity) bool { return len(circularCurvesFrom(ents)) >= 2 }
func readyLineOr2Points(ents []sketch.Entity) bool {
	return len(filterLines(ents)) >= 1 || len(filterPoints(ents)) >= 2
}

func readyEqual(ents []sketch.Entity) bool {
	return len(filterLines(ents)) >= 2 || len(circularCurvesFrom(ents)) >= 2
}

// readyTangent is satisfied by a line and a circle/arc, or two circles/arcs.
func readyTangent(ents []sketch.Entity) bool {
	curves := len(circularCurvesFrom(ents))
	return (len(filterLines(ents)) >= 1 && curves >= 1) || curves >= 2
}

func readyFix(ents []sketch.Entity) bool {
	return len(filterPoints(ents)) >= 1 || len(filterLines(ents)) >= 1
}

// readySymmetry needs two points to mirror and a line to mirror them about.
func readySymmetry(ents []sketch.Entity) bool {
	return len(filterPoints(ents)) >= 2 && len(filterLines(ents)) >= 1
}

// acceptSmoothable accepts the curves Smooth can join — a line, arc, or spline.
func acceptSmoothable(e sketch.Entity) bool {
	_, ok := e.(sketch.SmoothCurve)
	return ok
}

// readySmooth needs two smoothable curves, at least one of them a spline.
func readySmooth(ents []sketch.Entity) bool {
	curves := smoothCurvesFrom(ents)
	return len(curves) >= 2 && sketch.HasSplineCurve(curves)
}

func readyDimension(ents []sketch.Entity) bool {
	if len(filterCircles(ents)) >= 1 || len(filterLines(ents)) >= 2 {
		return true
	}
	_, _, ok := pointPairFrom(ents)
	return ok
}

// constraintToolDefs is the table of constraint/dimension tools (ribbon order).
var constraintToolDefs = []struct {
	id, name, tooltip, prompt string
	new                       func() *ConstraintTool
}{
	{
		"Sketch.Coincident", "Coincident", "Coincident — pick two points, or a point and a line/circle/arc.", "Select two points, or a point and a line/circle/arc",
		func() *ConstraintTool {
			return &ConstraintTool{name: "Coincident", prompt: "Select two points, or a point and a line/circle/arc", accepts: acceptCoincident, ready: readyCoincident, apply: applyCoincidentSnap}
		},
	},
	{
		"Sketch.Collinear", "Collinear", "Collinear — pick two lines.", promptSelectTwoLines,
		func() *ConstraintTool {
			return &ConstraintTool{name: "Collinear", prompt: promptSelectTwoLines, accepts: acceptLines, ready: ready2Lines, apply: entityApply(applyCollinear)}
		},
	},
	{
		"Sketch.Parallel", "Parallel", "Parallel — pick two lines.", promptSelectTwoLines,
		func() *ConstraintTool {
			return &ConstraintTool{name: "Parallel", prompt: promptSelectTwoLines, accepts: acceptLines, ready: ready2Lines, apply: entityApply(applyParallel)}
		},
	},
	{
		"Sketch.Perpendicular", "Perpendicular", "Perpendicular — pick two lines.", promptSelectTwoLines,
		func() *ConstraintTool {
			return &ConstraintTool{name: "Perpendicular", prompt: promptSelectTwoLines, accepts: acceptLines, ready: ready2Lines, apply: entityApply(applyPerpendicular)}
		},
	},
	{
		"Sketch.Horizontal", "Horizontal", "Horizontal — pick a line or two points.", promptSelectLineOrTwoPoints,
		func() *ConstraintTool {
			return &ConstraintTool{name: "Horizontal", prompt: promptSelectLineOrTwoPoints, accepts: acceptLinesOrPoints, ready: readyLineOr2Points, apply: entityApply(applyHorizontal)}
		},
	},
	{
		"Sketch.Vertical", "Vertical", "Vertical — pick a line or two points.", promptSelectLineOrTwoPoints,
		func() *ConstraintTool {
			return &ConstraintTool{name: "Vertical", prompt: promptSelectLineOrTwoPoints, accepts: acceptLinesOrPoints, ready: readyLineOr2Points, apply: entityApply(applyVertical)}
		},
	},
	{
		"Sketch.Tangent", "Tangent", "Tangent — pick a line and a circle/arc, or two circles/arcs.", "Select a line and a circle/arc, or two circles/arcs",
		func() *ConstraintTool {
			return &ConstraintTool{name: "Tangent", prompt: "Select a line and a circle/arc, or two circles/arcs", accepts: acceptLineOrCircular, ready: readyTangent, apply: entityApply(applyTangent)}
		},
	},
	{
		"Sketch.Concentric", "Concentric", "Concentric — pick two circles or arcs.", "Select two circles or arcs",
		func() *ConstraintTool {
			return &ConstraintTool{name: "Concentric", prompt: "Select two circles or arcs", accepts: acceptCircular, ready: ready2Circular, apply: entityApply(applyConcentric)}
		},
	},
	{
		"Sketch.Equal", "Equal", "Equal — pick two lines, or two circles/arcs.", "Select two lines, or two circles/arcs",
		func() *ConstraintTool {
			return &ConstraintTool{name: "Equal", prompt: "Select two lines, or two circles/arcs", accepts: acceptLineOrCircular, ready: readyEqual, apply: entityApply(applyEqual)}
		},
	},
	{
		"Sketch.Fix", "Fix", "Fix — pick a point or line to pin.", "Select a point or line to fix",
		func() *ConstraintTool {
			return &ConstraintTool{name: "Fix", prompt: "Select a point or line to fix", accepts: acceptLinesOrPoints, ready: readyFix, apply: entityApply(applyFix)}
		},
	},
	{
		"Sketch.Symmetric", "Symmetric", "Symmetric — pick two points and the mirror line.", "Select two points and the symmetry line",
		func() *ConstraintTool {
			return &ConstraintTool{name: "Symmetric", prompt: "Select two points and the symmetry line", accepts: acceptLinesOrPoints, ready: readySymmetry, apply: entityApply(applySymmetry)}
		},
	},
	{
		"Sketch.Smooth", "Smooth", "Smooth (G2) — pick a spline and an adjacent curve.", "Select a spline and an adjacent curve to join smoothly (G2)",
		func() *ConstraintTool {
			return &ConstraintTool{name: "Smooth", prompt: "Select a spline and an adjacent curve to join smoothly (G2)", accepts: acceptSmoothable, ready: readySmooth, apply: entityApply(applySmooth)}
		},
	},
}

// newDimensionTool builds the dimension tool (distance/length, radius, or angle).
func newDimensionTool() *ConstraintTool {
	return &ConstraintTool{
		name: "Dimension", prompt: "Select points, a line, a circle, or two lines",
		accepts: acceptDimensionable, ready: readyDimension, apply: entityApply(applyDimension),
	}
}
