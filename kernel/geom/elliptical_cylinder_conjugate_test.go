// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestConjugatePerpendicularRecoversAxes pins the sanity case: two ORTHOGONAL conjugate
// semi-diameters (a swept profile whose extrusion is perpendicular to its plane) must return the
// same principal axes and radii — major along u1 when |u1|>|u2|, unchanged radii a,b.
func TestConjugatePerpendicularRecoversAxes(t *testing.T) {
	m := math.V3(1, 0, 0)
	w := math.V3(0, 1, 0)
	u1 := m.Scale(150) // a=150 along m
	u2 := w.Scale(100) // b=100 along w
	majorR, minorR, majorDir, err := principalAxesFromConjugate(u1, u2)
	if err != nil {
		t.Fatalf("principalAxesFromConjugate: %v", err)
	}
	if !nearlyEq(majorR, 150, 1e-9) || !nearlyEq(minorR, 100, 1e-9) {
		t.Fatalf("radii = (%g, %g), want (150, 100)", majorR, minorR)
	}
	if d, _ := math.UnitVector3FromVector(majorDir); stdmath.Abs(d.AsVector().Dot(m))-1 > 1e-9 {
		t.Errorf("major direction = %v, want ±%v", majorDir, m)
	}
}

// TestConjugateAreaOracle is the load-bearing correctness check (advisor's oracle): for ANY pair of
// conjugate semi-diameters, the principal radii satisfy majorR·minorR = |u1 × u2| (Apollonius'
// invariant). This holds even when u1,u2 are non-orthogonal (the general oblique-ellipse case).
func TestConjugateAreaOracle(t *testing.T) {
	// A deliberately skew (non-orthogonal) conjugate pair — the oblique-ellipse extrusion case.
	u1 := math.V3(150, 40, 0)
	u2 := math.V3(30, 100, 0)
	majorR, minorR, _, err := principalAxesFromConjugate(u1, u2)
	if err != nil {
		t.Fatalf("principalAxesFromConjugate: %v", err)
	}
	wantArea := u1.Cross(u2).Length() // |u1 × u2|
	if got := majorR * minorR; !nearlyEq(got, wantArea, 1e-6) {
		t.Errorf("majorR·minorR = %g, want |u1×u2| = %g", got, wantArea)
	}
}

// TestConjugateObliqueCircleForeshortens pins the corrected physics (advisor): an oblique linear
// extrusion of a CIRCLE of radius r along direction d has a perpendicular cross-section with radii
// r (unchanged, along d×n) and r·|d·n| (FORESHORTENED — an orthogonal projection cannot enlarge).
// This mirrors STEP case U3: CIRCLE(12) swept along an oblique direction.
func TestConjugateObliqueCircleForeshortens(t *testing.T) {
	r := 12.0
	n := math.V3(0, 0, 1)           // circle plane normal
	m := math.V3(1, 0, 0)           // any in-plane axis
	w := n.Cross(m)                 // = (0,1,0)
	phi := 0.6                      // tilt angle of d off the normal
	d := math.V3(stdmath.Sin(phi), 0, stdmath.Cos(phi))
	dn := d.Dot(n) // = cos phi
	// conjugate semi-diameters = r·(m − (m·d)d), r·(w − (w·d)d)
	u1 := project(m, d).Scale(r)
	u2 := project(w, d).Scale(r)
	majorR, minorR, _, err := principalAxesFromConjugate(u1, u2)
	if err != nil {
		t.Fatalf("principalAxesFromConjugate: %v", err)
	}
	if !nearlyEq(majorR, r, 1e-9) {
		t.Errorf("major radius = %g, want unchanged r = %g", majorR, r)
	}
	if !nearlyEq(minorR, r*stdmath.Abs(dn), 1e-9) {
		t.Errorf("minor radius = %g, want foreshortened r·|d·n| = %g", minorR, r*stdmath.Abs(dn))
	}
}

// TestConjugateDegenerateErrors pins the grazing guard: a swept profile nearly parallel to its
// extrusion direction collapses the cross-section to a line — the two conjugate diameters become
// (near-)parallel, and the constructor must reject rather than mint a zero-width cylinder.
func TestConjugateDegenerateErrors(t *testing.T) {
	u1 := math.V3(150, 0, 0)
	u2 := math.V3(150.0000001, 1e-12, 0) // essentially parallel to u1
	if _, _, _, err := principalAxesFromConjugate(u1, u2); err == nil {
		t.Fatal("expected degenerate-section error for near-parallel conjugate diameters, got nil")
	}
}

// TestNewEllipticalCylinderFromConjugate builds the surface end-to-end and checks a boundary point
// of the base ellipse lands on it (ParamAt→PointAt round-trips on-surface). d ⊥ plane so the section
// is the base ellipse itself.
func TestNewEllipticalCylinderFromConjugate(t *testing.T) {
	o := math.P3(5, 0, 0)
	d := math.V3(0, 0, 1) // perpendicular sweep
	u1 := math.V3(150, 0, 0)
	u2 := math.V3(0, 100, 0)
	cyl, err := NewEllipticalCylinderFromConjugate(o, d, u1, u2)
	if err != nil {
		t.Fatalf("NewEllipticalCylinderFromConjugate: %v", err)
	}
	// A point on the base ellipse at t=0: o + 150·m.
	p := math.P3(5+150, 0, 3)
	u, v := cyl.ParamAt(p)
	if got := cyl.PointAt(u, v); got.DistanceTo(p) > 1e-6 {
		t.Errorf("ParamAt→PointAt round-trip: got %v, want %v", got, p)
	}
	if !nearlyEq(cyl.MajorRadius, 150, 1e-9) || !nearlyEq(cyl.MinorRadius, 100, 1e-9) {
		t.Errorf("radii = (%g, %g), want (150, 100)", cyl.MajorRadius, cyl.MinorRadius)
	}
}

// project removes the component of v along unit-ish d (the oblique projection onto the plane ⊥ d).
func project(v, d math.Vector3) math.Vector3 {
	return v.Sub(d.Scale(v.Dot(d)))
}

func nearlyEq(a, b, tol float64) bool { return stdmath.Abs(a-b) <= tol }
