// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/math"
)

// The 3D Sketch tab's spline-family draw tools (issue #145): the interpolating
// fit spline and the control-point spline (both point chains over Spline3DTool)
// and the parametric equation curve. The fixed spline deliberately has no tool —
// as in Inventor, fixed splines only arise programmatically (imports / derived
// geometry), so it stays an api/wire-only kind.

// Spline3DTool draws a 3D spline from clicked points: fit interpolates through
// them (3D Spline), control shapes the curve by its control polygon (3D Control
// Point Spline). Closed wraps the curve back to its first point.
type Spline3DTool struct {
	dialogTool
	points []math.Point3
	fit    bool
	closed bool
}

// NewSpline3DTool returns the interpolating (fit) spline tool.
func NewSpline3DTool() *Spline3DTool { return &Spline3DTool{fit: true} }

// NewControlPointSpline3DTool returns the control-polygon spline tool.
func NewControlPointSpline3DTool() *Spline3DTool { return &Spline3DTool{} }

// Name implements [Tool]; Start/Pick are no-ops.
func (t *Spline3DTool) Name() string {
	if t.fit {
		return "3D Spline"
	}
	return "3D Control Point Spline"
}

// AddPoint appends the next defining point (a viewport click).
func (t *Spline3DTool) AddPoint(p math.Point3) { t.points = append(t.points, p) }

// CanCommit is true once two points exist (a 2-point spline is its chord).
func (t *Spline3DTool) CanCommit() bool { return len(t.points) >= 2 }

// Commit adds the spline to the active 3D sketch.
func (t *Spline3DTool) Commit(s *Session) error {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return errors.New("3D spline: not editing a 3D sketch")
	}
	if !t.CanCommit() {
		return errors.New("3D spline: need at least two points")
	}
	sk.AddSpline3D(t.points, t.closed, t.fit)
	return nil
}

// Cancel implements [Tool].

// Params exposes the closed flag (points come from viewport clicks).
func (t *Spline3DTool) Params() ToolParams {
	return ToolParams{Bools: []BoolParam{
		{"Closed", func() bool { return t.closed }, func(b bool) { t.closed = b }},
	}}
}

// EquationCurve3DTool draws a parametric x(t)/y(t)/z(t) curve over [t0,t1]; the
// expressions go through the document's expression engine (lengths in cm).
type EquationCurve3DTool struct {
	dialogTool
	xExpr, yExpr, zExpr string
	t0, t1              float64
}

// NewEquationCurve3DTool returns an equation-curve tool over t ∈ [0,1] with the
// expressions left for the dialog.
func NewEquationCurve3DTool() *EquationCurve3DTool { return &EquationCurve3DTool{t1: 1} }

// Name implements [Tool]; Start/Pick are no-ops (all input is the dialog's).
func (t *EquationCurve3DTool) Name() string { return "Equation Curve" }

// CanCommit requires all three expressions and a non-empty parameter range.
func (t *EquationCurve3DTool) CanCommit() bool {
	return t.xExpr != "" && t.yExpr != "" && t.zExpr != "" && t.t1 > t.t0
}

// Commit adds the curve to the active 3D sketch; a bad expression keeps the tool
// open with the kernel's error (which names the offending expression).
func (t *EquationCurve3DTool) Commit(s *Session) error {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return errors.New("equation curve: not editing a 3D sketch")
	}
	if !t.CanCommit() {
		return errors.New("equation curve: need x(t), y(t), z(t) and t end > t start")
	}
	_, err := sk.AddEquationCurve3D(t.xExpr, t.yExpr, t.zExpr, t.t0, t.t1)
	return err
}

// Cancel implements [Tool].

// Params exposes the three coordinate expressions and the parameter range.
func (t *EquationCurve3DTool) Params() ToolParams {
	return ToolParams{
		Texts: []TextParam{
			{"x(t)", func() string { return t.xExpr }, func(v string) { t.xExpr = v }},
			{"y(t)", func() string { return t.yExpr }, func(v string) { t.yExpr = v }},
			{"z(t)", func() string { return t.zExpr }, func(v string) { t.zExpr = v }},
		},
		Floats: []FloatParam{
			{"t start", func() float64 { return t.t0 }, func(v float64) { t.t0 = v }},
			{"t end", func() float64 { return t.t1 }, func(v float64) { t.t1 = v }},
		},
	}
}

// compile-time assertions that the spline tools satisfy Tool and expose the dialog.
var (
	_ Tool              = (*Spline3DTool)(nil)
	_ Tool              = (*EquationCurve3DTool)(nil)
	_ ParameterizedTool = (*Spline3DTool)(nil)
	_ ParameterizedTool = (*EquationCurve3DTool)(nil)
)
