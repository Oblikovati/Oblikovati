// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// W1 canal-weld dispatch + tagged boundary-isocurve accessor + canalWeldFaces skeleton. Every assertion
// drives the REAL N7 fixture (n7CornerFill → extractCurvedCorner → resolveBlend), never a fabricated
// patch, and the DRAWEXE-oracle geometry (end-arc centres C=(45,y,15)/C″=(55,y,5); feet on wall R=50 /
// s_10 R=5). The skeleton floors, so N7 stays declined (corpus 55) until the arm faces land in W2-W4.

// TestCanalBoundaryRoles proves canalBoundaryRoles tags the four canal boundaries by role on the real N7
// patch: the two END ARCS are radius-r cross-section arcs whose fitted centres are the two reflected ball
// centres C/C″, and the two FOOT-LOCI lie on their roll hosts (feet[0] on the wall R=50, feet[1] on s_10
// R=5). The centre/on-host checks are the discriminator: a mis-tag (a foot-locus placed in endArcs, or an
// end arc placed in feet) has neither a ball-centre nor lies on the host, so it fails here.
func TestCanalBoundaryRoles(t *testing.T) {
	t.Parallel()
	patch := n7CanalPatch(t)
	roles, err := canalBoundaryRoles(patch)
	if err != nil {
		t.Fatalf("canalBoundaryRoles declined the real N7 canal patch: %v", err)
	}
	res := geom.ResolutionForSize(150)
	tol := res.Weld() * 50 // tangentCornerScale = wall radius R=50 (model-relative, ADR-0042)

	y := 50 - stdmath.Sqrt(2000) // 5.278640… — s_4/s_5 spine y where x-offset meets R−r
	wantCentres := []math.Point3{math.P3(45, y, 15), math.P3(55, y, 5)}
	assertEndArcCentres(t, roles.endArcs, wantCentres, tol)

	assertFootOnHost(t, roles.feet[0], math.P3(50, 50, 0), math.V3(0, 0, 1), 50, tol, "wall (feet[0]/u0)")
	assertFootOnHost(t, roles.feet[1], math.P3(55, 0, 15), math.V3(0, 1, 0), 5, tol, "s_10 (feet[1]/u1)")

	// Discrimination, made explicit: an end arc must NOT lie on the wall roll host (it is a cross-section
	// arc about a ball centre, off the wall) — so a swapped tag (foot in the endArcs slot) is caught.
	if onHost, off := footIsOnHost(roles.endArcs[0], math.P3(50, 50, 0), math.V3(0, 0, 1), 50, tol); onHost {
		t.Fatalf("endArcs[0] lies on the wall (off %.3e) — roles look swapped (an end arc is NOT a foot-locus)", off)
	}
}

// TestCanalWeldFacesAssemblesCornerAndArms pins the F3 behaviour on a FACE-LESS fixture body: canalWeldFaces
// builds the corner patch face + the three per-arm-centre arm faces and returns an EMPTY reason (F3 wired
// the final assembly, replacing the W3 "not yet assembled" floor). With no host faces on the fixture body,
// it returns exactly the corner patch + the three arm faces (the host retrims run only when the body
// carries the wall/plane faces — the whole-body watertight weld is gated by TestOCCTBlendSimple/N7).
func TestCanalWeldFacesAssemblesCornerAndArms(t *testing.T) {
	t.Parallel()
	w, arms, res := n7CornerFill(t)
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok || loop.Canal == nil {
		t.Fatalf("N7 must extract a Canal-marked loop (the dispatch signal); ok=%v canal=%v", ok, loop.Canal != nil)
	}
	faces, reason := canalWeldFaces(emptyBody(t), arms, w, loop, res)
	if reason != "" {
		t.Fatalf("F3 canalWeldFaces must assemble (empty reason), got decline %q", reason)
	}
	if len(faces) != 1+len(arms) {
		t.Fatalf("F3 must build the corner patch + %d arm faces = %d faces, got %d", len(arms), 1+len(arms), len(faces))
	}
	if _, isBSpline := faces[0].surface.(geom.BSplineSurface); !isBSpline {
		t.Fatalf("corner face surface is %T, want the canal geom.BSplineSurface", faces[0].surface)
	}
}

// TestCanalArmBodyRoutesN7 proves the dispatch ROUTES the N7 corner (loop.Canal != nil) into the canal
// weld AND that F3 now assembles it: canalArmBody, fed the blend sphere at the corner ball centre C, takes
// the corner (took=true) and returns a non-nil assembled body with no decline reason (the sibling
// assembler produced a body). The whole-body watertightness is gated on the real STEP body by
// TestOCCTBlendSimple/N7; here the fixture is face-less, so the returned body carries only the corner +
// arm faces (not a closed solid) — the observable under test is that the canal path assembles, not floors.
func TestCanalArmBodyRoutesN7(t *testing.T) {
	t.Parallel()
	w, arms, res := n7CornerFill(t)
	sphere, err := geom.NewSphere(w.center, w.radius)
	if err != nil {
		t.Fatalf("build N7 corner sphere: %v", err)
	}
	blends := map[uint64]*cornerBlend{7: {sphere: sphere}}
	body, reason, took := canalArmBody(emptyBody(t), arms, blends, 7, res)
	if !took {
		t.Fatal("canalArmBody must TAKE the tangent-degenerate N7 corner (loop.Canal != nil)")
	}
	if reason != "" {
		t.Fatalf("F3 canalArmBody must assemble (empty reason), got decline %q", reason)
	}
	if body == nil {
		t.Fatal("F3 canalArmBody must return the assembled body, got nil")
	}
}

// TestCanalArmBodyPassesOctant proves the dispatch does NOT touch a concurrent-spine corner: the certified
// B3 octant (Canal==nil) returns took=false, so assembleCurvedArmBody falls through to the untouched
// single-ball path — the invariant that keeps B3 byte-identical.
func TestCanalArmBodyPassesOctant(t *testing.T) {
	t.Parallel()
	_, arms, sphere, res := b3CornerWeld(t)
	blends := map[uint64]*cornerBlend{7: {sphere: sphere}}
	if body, reason, took := canalArmBody(nil, arms, blends, 7, res); took {
		t.Fatalf("canalArmBody must PASS the concurrent B3 octant to the single-ball path (Canal==nil); took=true (body=%v reason=%q)", body, reason)
	}
}

// n7CanalPatch resolves the real N7 corner into its canal patch (BlendKindCanal) for the role tests.
func n7CanalPatch(t *testing.T) CornerBlendPatch {
	t.Helper()
	w, arms, res := n7CornerFill(t)
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok || loop.Canal == nil {
		t.Fatalf("N7 must extract a Canal-marked loop; ok=%v canal=%v", ok, loop.Canal != nil)
	}
	patch, ok := resolveBlend(loop, res)
	if !ok || patch.Kind != BlendKindCanal {
		t.Fatalf("N7 must resolve to the canal tier; ok=%v kind=%q", ok, patch.Kind)
	}
	return patch
}

// emptyBody builds a face-less topo.Body — canalWeldFaces reads only len(body.Faces()) (a capacity hint),
// so an empty body is enough to exercise the skeleton without the full feature build path.
func emptyBody(t *testing.T) *topo.Body {
	t.Helper()
	return topo.NewBuilder(true, topo.Lineage{}).Build()
}

// assertEndArcCentres fits a circle through three points of each end-arc isocurve and asserts the two
// fitted centres are the two wanted reflected ball centres (each within tol). A foot-locus mis-tagged as
// an end arc has no such circle centre at C/C″, so this fails on a swapped role.
func assertEndArcCentres(t *testing.T, arcs [2]geom.Curve3, want []math.Point3, tol float64) {
	t.Helper()
	got := [2]math.Point3{arcCurveCentre(t, arcs[0]), arcCurveCentre(t, arcs[1])}
	for _, wc := range want {
		if !anyPointWithin(got[:], wc, tol) {
			t.Fatalf("no end-arc centre within %.3e of reflected ball centre %v; fitted centres = %v", tol, wc, got)
		}
	}
}

// arcCurveCentre returns the centre of a circular isocurve by fitting a circle through its two ends and
// midpoint (the canal v-boundaries are exact radius-r cross-section arcs, so the fit is well posed).
func arcCurveCentre(t *testing.T, c geom.Curve3) math.Point3 {
	t.Helper()
	lo, hi := c.Domain()
	arc, err := geom.Arc3dByThreePoints(c.PointAt(lo), c.PointAt((lo+hi)/2), c.PointAt(hi))
	if err != nil {
		t.Fatalf("fit end-arc circle: %v", err)
	}
	return arc.Center
}

// assertFootOnHost asserts every sampled point of a foot-locus lies within tol of the host cylinder's
// radius (perpendicular distance to its axis), failing name'd on the max off-host deviation.
func assertFootOnHost(t *testing.T, c geom.Curve3, origin math.Point3, axis math.Vector3, radius, tol float64, name string) {
	t.Helper()
	if onHost, off := footIsOnHost(c, origin, axis, radius, tol); !onHost {
		t.Fatalf("%s foot-locus is off its host radius %g by %.3e (tol %.3e)", name, radius, off, tol)
	}
}

// footIsOnHost reports whether every sampled point of c lies within tol of the cylinder (origin, unit
// axis, radius), and the max off-host deviation. Used both to assert the feet ARE on-host and to prove
// the end arcs are NOT (the role discriminator).
func footIsOnHost(c geom.Curve3, origin math.Point3, axis math.Vector3, radius, tol float64) (bool, float64) {
	lo, hi := c.Domain()
	maxOff := 0.0
	for i := 0; i <= 64; i++ {
		p := c.PointAt(lo + float64(i)/64*(hi-lo))
		radial := origin.VectorTo(p)
		d := float64(radial.Sub(axis.Scale(radial.Dot(axis))).Length())
		maxOff = stdmath.Max(maxOff, stdmath.Abs(d-radius))
	}
	return maxOff <= tol, maxOff
}

// anyPointWithin reports whether any point in pts is within tol of q.
func anyPointWithin(pts []math.Point3, q math.Point3, tol float64) bool {
	for _, p := range pts {
		if float64(p.DistanceTo(q)) <= tol {
			return true
		}
	}
	return false
}
