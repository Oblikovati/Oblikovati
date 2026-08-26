// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/solve"
)

// representativeSketch builds a sketch exercising a broad spread of geometric and
// dimensional constraint types — the fixture the guard tests run against.
func representativeSketch(t *testing.T) *Sketch {
	t.Helper()
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(2, 0))
	l1 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(2, 0.1))
	l2 := s.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(2, 1.2))
	c1 := s.Circles().AddByCenterRadius(math.P2(1, 1), 0.5)
	g.AddCoincident(a, b)
	g.AddHorizontal(a, b)
	g.AddParallel(l1, l2)
	g.AddTangent(l1, c1)
	g.AddPointOnCircle(a, c1)
	g.AddFix(a)
	if _, err := s.DimensionConstraints().AddDistance(a, b, "2 cm"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	if _, err := s.DimensionConstraints().AddRadius(c1, "0.5 cm"); err != nil {
		t.Fatalf("AddRadius: %v", err)
	}
	return s
}

// TestSketchSolvePathIsFiniteDifferenceFree is acceptance criterion 1 of #1417: every
// constraint a 2D sketch hands the solver supplies its own analytic partials, so the
// solver's Jacobian assembly takes the analytic path and never finite-differences.
func TestSketchSolvePathIsFiniteDifferenceFree(t *testing.T) {
	s := representativeSketch(t)
	for _, c := range s.Constraints() {
		if _, ok := any(c).(solve.Differentiable); !ok {
			t.Errorf("constraint %T is not solve.Differentiable — it would force the finite-difference fallback", c)
		}
	}
}

// TestMobilityAndDOFDoNotPerturbLiveGeometry is acceptance criterion 3 of #1417: the
// hover-time DOF and mobility analyses build the Jacobian from exact analytic partials
// that READ the live variables, so no coordinate is transiently written (as the old
// finite-difference Jacobian did with orig±h).
func TestMobilityAndDOFDoNotPerturbLiveGeometry(t *testing.T) {
	s := representativeSketch(t)
	s.Solve()

	before := snapshotVars(s.variables())
	_ = s.DegreesOfFreedom()
	_ = s.AnalyzeConstraints()
	mc := s.MoveableClassifier()
	for _, e := range []Entity{s.Points().Item(0), s.Lines().Item(0), s.Circles().Item(0)} {
		_ = mc.Of(e)
	}
	after := snapshotVars(s.variables())

	for i := range before {
		if before[i] != after[i] {
			t.Errorf("variable %d changed during DOF/mobility analysis: %v → %v (the analysis path must not perturb live geometry)",
				i, before[i], after[i])
		}
	}
}
