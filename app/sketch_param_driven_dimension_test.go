// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// A dimension's value is an EXPRESSION in the document's parameter DAG, not a frozen number, so a
// user parameter can drive it: dimension the sketch with "width", edit width, and the geometry
// follows. That is the whole point of the parametric model, and these pin it end to end — the
// dimension resolving, the solver honouring it, and an edit propagating.

// parameterisedSketch returns a session editing a sketch bound to the part's parameter set, plus
// that set — the wiring a dimension needs to see user parameters at all.
func parameterisedSketch(t *testing.T) (*Session, *sketch.Sketch, *param.Parameters) {
	t.Helper()
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	ps := def.Parameters()
	if sk.Parameters() != ps {
		t.Fatal("the sketch is not bound to the part's parameters — a dimension could not name one")
	}
	return s, sk, ps
}

// TestUserParameterDrivesADimension: naming a parameter as the dimension's expression must resolve
// to that parameter's value, not to zero or a parse failure.
func TestUserParameterDrivesADimension(t *testing.T) {
	t.Parallel()
	_, sk, ps := parameterisedSketch(t)
	if _, err := ps.AddUserParameter("width", "40 mm"); err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))

	d, err := sk.DimensionConstraints().AddDistance(l.A, l.B, "width")
	if err != nil {
		t.Fatalf("AddDistance(\"width\"): %v", err)
	}

	if got := d.Parameter().ModelValue(); got != 4 { // 40 mm = 4 cm, the database unit
		t.Errorf("dimension resolves to %v cm, want 4 (the parameter's 40 mm)", got)
	}
	if r := sk.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: residual %v", r.Residual)
	}
	if got := float64(l.A.Position().DistanceTo(l.B.Position())); got < 3.999 || got > 4.001 {
		t.Errorf("solved line length %v cm, want the parameter's 4", got)
	}
}

// TestEditingTheParameterMovesTheGeometry is the point of driving a dimension with a parameter:
// change the parameter, and every dimension naming it re-solves.
func TestEditingTheParameterMovesTheGeometry(t *testing.T) {
	t.Parallel()
	_, sk, ps := parameterisedSketch(t)
	if _, err := ps.AddUserParameter("width", "40 mm"); err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	if _, err := sk.DimensionConstraints().AddDistance(l.A, l.B, "width"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	if r := sk.Solve(); !r.Converged {
		t.Fatalf("first solve did not converge: residual %v", r.Residual)
	}

	w, ok := ps.ByName("width")
	if !ok {
		t.Fatal("the width parameter vanished")
	}
	if err := ps.SetExpression(w.ID(), "65 mm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	if r := sk.Solve(); !r.Converged {
		t.Fatalf("re-solve did not converge: residual %v", r.Residual)
	}

	if got := float64(l.A.Position().DistanceTo(l.B.Position())); got < 6.499 || got > 6.501 {
		t.Errorf("after width=65 mm the line is %v cm, want 6.5 — the edit did not reach the geometry", got)
	}
}

// TestFormulaOverParametersDrivesADimension: a dimension may name an EXPRESSION over parameters,
// which is how a derived dimension is stated (CLAUDE.md: use formulas for derived dimensions).
func TestFormulaOverParametersDrivesADimension(t *testing.T) {
	t.Parallel()
	_, sk, ps := parameterisedSketch(t)
	if _, err := ps.AddUserParameter("width", "40 mm"); err != nil {
		t.Fatalf("AddUserParameter width: %v", err)
	}
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))

	d, err := sk.DimensionConstraints().AddDistance(l.A, l.B, "width / 2")
	if err != nil {
		t.Fatalf("AddDistance(\"width / 2\"): %v", err)
	}

	if got := d.Parameter().ModelValue(); got != 2 {
		t.Errorf("derived dimension = %v cm, want 2 (half of 40 mm)", got)
	}
}

// TestUnresolvedParameterIsFlaggedAndDoesNotDrive: naming a parameter that does not exist is NOT
// an error — a reference may be authored before the parameter it names and binds when it appears
// (Parameters.onParameterAdded). What must not happen is the unbound expression's ZERO reaching
// the solver: a mistyped "widht" drove a 40 mm line to 4.6e-12 and the solve reported success.
func TestUnresolvedParameterIsFlaggedAndDoesNotDrive(t *testing.T) {
	t.Parallel()
	_, sk, _ := parameterisedSketch(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))

	d, err := sk.DimensionConstraints().AddDistance(l.A, l.B, "widht")
	if err != nil {
		t.Fatalf("a forward reference must be allowed, not rejected: %v", err)
	}

	h := d.Parameter().Health()
	if h.OK() {
		t.Error("an undefined reference left the dimension healthy — nothing would tell the user")
	}
	if !strings.Contains(h.Reason, "widht") {
		t.Errorf("health reason %q should name the offending reference", h.Reason)
	}
	if r := sk.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: residual %v", r.Residual)
	}
	if got := float64(l.A.Position().DistanceTo(l.B.Position())); got < 3.999 {
		t.Errorf("the line collapsed to %v cm — a sick dimension drove the geometry to zero", got)
	}
}

// TestLateDefinedParameterBindsAndDrives is the other half: once the named parameter exists, the
// dimension binds to it and starts driving, without the dimension being re-authored.
func TestLateDefinedParameterBindsAndDrives(t *testing.T) {
	t.Parallel()
	_, sk, ps := parameterisedSketch(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	d, err := sk.DimensionConstraints().AddDistance(l.A, l.B, "late_w")
	if err != nil {
		t.Fatalf("AddDistance: %v", err)
	}

	if _, err := ps.AddUserParameter("late_w", "70 mm"); err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}

	if !d.Parameter().Health().OK() {
		t.Fatalf("the dimension did not bind once its parameter existed: %v", d.Parameter().Health().Reason)
	}
	if r := sk.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: residual %v", r.Residual)
	}
	if got := float64(l.A.Position().DistanceTo(l.B.Position())); got < 6.999 || got > 7.001 {
		t.Errorf("line length %v cm, want the late-defined 7", got)
	}
}

// TestParameterDrivesARectanglesLockedDimension is the interactive path: a rectangle placed with a
// locked value creates a real driving dimension, and that dimension is an expression in the same
// DAG — so it can be RE-STATED as a parameter afterwards and keep driving the shape.
func TestParameterDrivesARectanglesLockedDimension(t *testing.T) {
	t.Parallel()
	s, sk, ps := parameterisedSketch(t)
	if _, err := ps.AddUserParameter("plate_w", "80 mm"); err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	s.StartTool(NewRectangleTool())
	s.Click(40, 40)
	for _, r := range "50" {
		s.PlacementFieldInput(r)
	}
	s.PlacementFieldTab() // lock Width, so the commit creates a driving dimension
	s.Click(140, 120)

	dims := sk.DimensionConstraints().All()
	if len(dims) == 0 {
		t.Fatal("a locked placement value must create a driving dimension")
	}
	if err := ps.SetExpression(dims[0].Parameter().ID(), "plate_w"); err != nil {
		t.Fatalf("re-state the dimension as a parameter: %v", err)
	}
	if r := sk.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: residual %v", r.Residual)
	}
	if got := dims[0].Parameter().ModelValue(); got != 8 { // 80 mm = 8 cm
		t.Errorf("the rectangle's dimension = %v cm, want the parameter's 8", got)
	}
}
