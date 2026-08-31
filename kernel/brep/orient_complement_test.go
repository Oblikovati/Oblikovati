// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The #3453 / #3429 regression for the ORIENTATION side of the closed-surface trim family: a Ø10 ball
// joined with a Ø6 rod through its top keeps the 9/10-of-the-sphere cap below the seam, whose only trim
// ring is the small circle around the rod. That face is the COMPLEMENT of its ring in the sphere's
// parameter domain, and reading the ring as the region inverted both readings the winding path takes off
// it — the outward sign (−1 where the outward normal is +1, off a ring of signed area −1.317) and the
// quadrature rectangle (the ring's bounding box covers the small cap instead of the big one). So
// PointInside reported the ball's own centre OUTSIDE the solid, which declined the measurement gating
// ops' analytic result and demoted a geometrically correct boolean to the faceted fallback.

// shoulderRodJoin is the reference body: a ball of radius 5 at the origin joined with a coaxial rod of
// radius 3 running from the centre out to y = 4.5 — past the seam circle at y = 4, so the ball's surface
// survives in two pieces and the big one is its ring's complement.
func shoulderRodJoin(t *testing.T) *topo.Body {
	t.Helper()
	ball, err := SolidSphere(math.P3(0, 0, 0), 5, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	rod, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 1, 0), 3, 4.5)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	join, ok := CoaxialSphereRodJoin(ball, rod)
	if !ok {
		t.Fatal("ball ∪ shoulder rod declined the analytic path")
	}
	return join
}

// TestBigCapOutwardSignSurvivesItsRingComplement pins the per-face outward signs of that body against
// the sign the geometry implies, read independently: step a hair along each face's normal at a point the
// face owns and ask the orientation-free ray parity whether that step left the solid.
func TestBigCapOutwardSignSurvivesItsRingComplement(t *testing.T) {
	join := shoulderRodJoin(t)
	faces := facesOfAny(join)
	q := newFluxQuery(faces)
	if len(q.faces) != 4 {
		t.Fatalf("ball ∪ shoulder rod prepared %d flux faces, want 4 (two caps, the rod wall, its lid)", len(q.faces))
	}
	box := join.RangeBox()
	for i := range q.faces {
		f := &q.faces[i]
		if got, want := f.sign, outwardSignFromParity(t, faces, f, box); got != want {
			t.Errorf("face %d (%T): orientFaceSigns says %+.0f, the geometry says %+.0f", i, f.cf.surface, got, want)
		}
	}
}

// TestBallJoinRodPointInsideClaimsItsInterior is the end-to-end reading of the same defect through the
// nearest-crossing/flux route [PointInside] takes, which the ray-parity route does not exercise.
func TestBallJoinRodPointInsideClaimsItsInterior(t *testing.T) {
	join := shoulderRodJoin(t)
	for _, p := range []math.Point3{math.P3(0, 0, 0), math.P3(0, -3, 0), math.P3(0, 4.2, 0), math.P3(0, 0, 4)} {
		if !PointInside(join, p) {
			t.Errorf("PointInside(ball ∪ shoulder rod, %v) = false, want true", p)
		}
	}
	for _, p := range []math.Point3{math.P3(9, 0, 0), math.P3(0, 6, 0), math.P3(4.5, 4.5, 0)} {
		if PointInside(join, p) {
			t.Errorf("PointInside(ball ∪ shoulder rod, %v) = true, want false", p)
		}
	}
}

// TestComplementIsClaimedOnlyOnAClosedDomain guards the generalization: the complement reading is asked
// for on a closed parameter domain only, so the rod's cylindrical wall and its planar lid — whose rings
// do bound their faces — keep the rings' interior, and exactly the one spherical cap flips.
func TestComplementIsClaimedOnlyOnAClosedDomain(t *testing.T) {
	q := newFluxQuery(facesOfAny(shoulderRodJoin(t)))
	complements := 0
	for i := range q.faces {
		s := q.faces[i].cf.surface
		if !q.faces[i].region.complement {
			continue
		}
		complements++
		uPer, vPer := surfacePeriodic(s)
		if _, open := castAxis(s, uPer, vPer); open {
			t.Errorf("face %d (%T) is read as its ring's complement on an OPEN domain", i, s)
		}
	}
	if complements != 1 {
		t.Errorf("ball ∪ shoulder rod has %d complement face(s), want 1 (the big spherical cap)", complements)
	}
}

// signProbeOffset steps off the surface for the independent sign probe. The body is ~10 units across and
// its thinnest feature is the rod's 0.5-long free stub, so this step lands well inside the material on
// the wrong side and well outside it on the right one, clear of the on-surface band.
const signProbeOffset = 1e-3

// signProbeGrid samples the surface domain per axis when looking for a point the face owns; it only has
// to find ONE.
const signProbeGrid = 40

// outwardSignFromParity is the sign the geometry implies for one prepared face, read WITHOUT the
// orientation machinery under test: offset a point the face owns along S_u×S_v by a hair and ask the
// orientation-free even–odd ray parity whether that offset point is still in the solid.
func outwardSignFromParity(t *testing.T, faces []curvedFace, f *fluxFace, box math.Box) float64 {
	t.Helper()
	u, v, ok := ownedParamOf(f)
	if !ok {
		t.Fatalf("no in-trim sample on face %T over [%g,%g]x[%g,%g]", f.cf.surface, f.u0, f.u1, f.v0, f.v1)
	}
	n := f.cf.surface.NormalAt(u, v)
	step := math.Scalar(signProbeOffset / float64(n.Length()))
	if rayParityInside(faces, f.cf.surface.PointAt(u, v).TranslateBy(n.Scale(step)), box) {
		return -1
	}
	return 1
}

// ownedParamOf scans for a (u, v) the face's trim owns, over the surface's own domain where that is
// finite — NOT the quadrature rectangle, so the scan is independent of the domain choice under test.
func ownedParamOf(f *fluxFace) (float64, float64, bool) {
	u0, u1 := finiteAxis(f.cf.surface.UDomain())
	v0, v1 := finiteAxis(f.cf.surface.VDomain())
	for i := 1; i < signProbeGrid; i++ {
		for j := 1; j < signProbeGrid; j++ {
			u, v := station(u0, u1, i), station(v0, v1, j)
			if pointInTrimUV(f.cf, f.cf.surface.PointAt(u, v)) {
				return u, v, true
			}
		}
	}
	return 0, 0, false
}

// finiteAxis replaces an unbounded parameter axis (a plane's, a cylinder's height) with a window wide
// enough to cover this body, so the scan has somewhere to look.
func finiteAxis(lo, hi float64) (float64, float64) {
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return -shoulderRodExtent, shoulderRodExtent
	}
	return lo, hi
}

// shoulderRodExtent bounds the reference body: everything in it lies within 5 of the origin.
const shoulderRodExtent = 5

// station is the k-th of signProbeGrid stations across [lo, hi].
func station(lo, hi float64, k int) float64 {
	return lo + (hi-lo)*float64(k)/signProbeGrid
}
