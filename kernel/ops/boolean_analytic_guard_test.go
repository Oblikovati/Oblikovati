// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestCurvedVolumeBracketRejectsGrossError pins the curved analytic volume guard (#1601): the
// Requicha two-sided bracket the planar path already uses, now applied to the curved analytic
// path with a tolerance that dominates curved tessellation deficit, accepts a correct-volume
// result and rejects a grossly-wrong one. This is the predicate curvedExactGuarded runs before
// shipping an analytic body — closing the hole where the analytic short-circuit bypassed the
// volume guard, so a recognizer that kept the wrong lobe or removed too much shipped silently.
func TestCurvedVolumeBracketRejectsGrossError(t *testing.T) {
	// V(A)=100, V(B)=30; the curved margin is 10% of the larger operand = 10.
	const va, vb = 100.0, 30.0
	tol := curvedVolumeGuardFraction * va
	cases := []struct {
		name    string
		op      PartFeatureOperation
		bodyVol float64
		bad     bool
	}{
		{"cut, tool half inside", Cut, 85, false},          // ∈ [70,100]
		{"cut, near lower edge (deficit)", Cut, 71, false}, // ~V(A)−V(B), inside the margin
		{"cut removes too much", Cut, 40, true},            // 40 < 70−10: wrong lobe kept
		{"cut fabricates material", Cut, 130, true},        // 130 > 100+10
		{"join correct", Join, 115, false},                 // ∈ [100,130]
		{"join fabricates material", Join, 145, true},      // 145 > 130+10
		{"join loses material", Join, 80, true},            // 80 < 100−10
		{"intersect correct", Intersect, 25, false},        // ≤ 30
		{"intersect too big", Intersect, 55, true},         // 55 > 30+10: wrong lobe kept
	}
	for _, c := range cases {
		if got := volumeOutOfBracket(c.op, va, vb, c.bodyVol, tol); got != c.bad {
			t.Errorf("%s: volumeOutOfBracket(%v, body=%g) = %v, want %v", c.name, c.op, c.bodyVol, got, c.bad)
		}
	}
}

// TestCurvedExactBooleanPassesVolumeGuard is the no-false-rejection regression (#1601): correct
// crossing-cylinder booleans (unequal radii — an exact analytic path) must STAY analytic through
// the new curved volume guard, never pushed to the faceted CSG fallback. Each result keeps its
// cylinder faces and records no analytic-volume-reject defect. These results sit deep inside their
// Requicha brackets; the margin (curvedVolumeGuardFraction) protects the near-boundary analytic
// results tessellation deficit could otherwise trip, and is calibrated from the deficit bound in
// that constant's comment. This test is the regression that the guard did not break the common path.
func TestCurvedExactBooleanPassesVolumeGuard(t *testing.T) {
	fat, err := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if err != nil {
		t.Fatalf("fat cylinder: %v", err)
	}
	thin, err := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	if err != nil {
		t.Fatalf("thin cylinder: %v", err)
	}
	for _, op := range []PartFeatureOperation{Intersect, Cut, Join} {
		rec := &diag.Recorder{}
		res, err := BooleanWithDiagnostics(op, fat, thin, rec)
		if err != nil || res == nil {
			t.Fatalf("%v: err=%v res=%v", op, err, res)
		}
		if rec.Has(CodeBooleanAnalyticVolumeReject) {
			t.Errorf("%v: a correct crossing-cylinder boolean was volume-rejected: %v", op, rec.Records())
		}
		if cylinderFaceCount(res) == 0 {
			t.Errorf("%v: result kept no cylinder face — the analytic path was abandoned (false rejection)", op)
		}
	}
}

// TestCurvedVolumeRejectHandsOffAndDiagnoses drives the reject-and-hand-off path end to end
// (#1601): with the bracket forced so even a CORRECT crossing-cylinder result reads as out of
// bracket, curvedExactGuarded must decline, record CodeBooleanAnalyticVolumeReject, and leave the
// operation to the next path — which must still deliver a valid solid.
//
// It no longer asserts that the next path is the FACETED CSG one. Since the per-face dispatch
// learned to imprint a conic onto a planar face (ADR-0058), crossing cylinders are handled
// analytically by brep.BooleanDiag, so a rejected recognizer result now hands off to an exact path
// rather than to facets. That is the better outcome, and pinning the old one would pin the loss.
func TestCurvedVolumeRejectHandsOffAndDiagnoses(t *testing.T) {
	forced := -1e9 // any bracket, even the true one, now reads as violated
	curvedGuardBracketOverride = &forced
	defer func() { curvedGuardBracketOverride = nil }()

	fat, err := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if err != nil {
		t.Fatalf("fat cylinder: %v", err)
	}
	thin, err := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	if err != nil {
		t.Fatalf("thin cylinder: %v", err)
	}
	rec := &diag.Recorder{}
	res, err := BooleanWithDiagnostics(Intersect, fat, thin, rec)
	if err != nil || res == nil {
		t.Fatalf("intersect: err=%v res=%v", err, res)
	}
	if !rec.Has(CodeBooleanAnalyticVolumeReject) {
		t.Errorf("rejected analytic result recorded no %q; got %v", CodeBooleanAnalyticVolumeReject, rec.Records())
	}
	if r := Validate(res); !r.ValidSolid() {
		t.Errorf("the path taken after the rejection left an invalid solid: %+v", r)
	}
}

// cylinderFaceCount counts the analytic cylinder faces on a body — a nonzero count proves the
// exact curved path ran (the CSG fallback triangulates every curved surface away).
func cylinderFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			n++
		}
	}
	return n
}
