// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A trimmed region of a SPHERE must classify correctly however LARGE it is (Oblikovati#3453, #3429).
// The retired classifier projected the loops orthographically onto the tangent plane at the query point,
// which folds the far hemisphere onto the near one, so any region whose rim lies more than a quarter turn
// from the query point wound to ~0 and read as OUTSIDE — at its own centre included. These sweeps pin both
// a region far LARGER than a hemisphere (9/10 of the ball) and one far smaller, so a classifier that
// merely inverted the old answer fails them too.

// capRegionFixture builds the curvedFace for the part of a radius-r sphere (centred at the origin) on one
// side of the plane y = level. Its single loop is the rim circle there, walked about +Y so that n × T —
// the surface normal's left side, the kept region — is the LOW side (y < level) when low is true and the
// HIGH side otherwise. Walking a circle CW about its axis (t 1→0) names the region below it.
func capRegionFixture(t *testing.T, r, level float64, low bool) curvedFace {
	t.Helper()
	sphere, err := geom.NewSphere(math.P3(0, 0, 0), r)
	if err != nil {
		t.Fatalf("sphere r=%g: %v", r, err)
	}
	rim, err := geom.NewCircle(math.P3(0, math.Scalar(level), 0), math.V3(0, 1, 0), stdmath.Sqrt(r*r-level*level))
	if err != nil {
		t.Fatalf("rim at y=%g on r=%g: %v", level, r, err)
	}
	t0, t1 := 1.0, 0.0
	if !low {
		t0, t1 = 0.0, 1.0
	}
	return curvedFace{surface: sphere, loops: []curvedLoop{{edges: []loopEdge{{curve: rim, t0: t0, t1: t1}}}}}
}

// meridianPoint returns the point at station y on the sphere of radius r, on the meridian at azimuth
// (the great circle through both poles of the +Y axis), so a sweep over y walks a full half turn.
func meridianPoint(r, y, azimuth float64) math.Point3 {
	ring := stdmath.Sqrt(stdmath.Max(0, r*r-y*y))
	return math.P3(math.Scalar(ring*stdmath.Cos(azimuth)), math.Scalar(y), math.Scalar(ring*stdmath.Sin(azimuth)))
}

// sweepStations returns count stations spanning (-r, r), plus the two that historically broke a
// classifier: y = -level, where the query point is exactly ANTIPODAL to a rim point (the fan/solid-angle
// formulations are singular there), and y = 0, the equator, squarely inside the big cap.
func sweepStations(r, level float64, count int) []float64 {
	out := make([]float64, 0, count+2)
	for i := 1; i < count; i++ {
		out = append(out, -r+2*r*float64(i)/float64(count))
	}
	return append(out, -level, 0)
}

// assertCapSweep walks every station of every listed azimuth and requires the trim verdict to agree with
// the defining half-space, skipping only a band around the rim itself where the verdict is genuinely
// ambiguous.
func assertCapSweep(t *testing.T, f curvedFace, r, level float64, low bool) {
	t.Helper()
	const rimBand = 0.02 // stations this close to the rim plane straddle the boundary; their verdict is undefined
	for _, azimuth := range []float64{0, 1.9, 4.4} {
		for _, y := range sweepStations(r, level, 400) {
			if stdmath.Abs(y-level) < rimBand {
				continue
			}
			p := meridianPoint(r, y, azimuth)
			want := y < level
			if !low {
				want = y > level
			}
			if got := pointInCurvedFace(f, p); got != want {
				t.Fatalf("cap(low=%v) at y=%g azimuth=%g (%v): inside=%v, want %v", low, y, azimuth, p, got, want)
			}
		}
	}
}

// TestPointInCurvedFaceBigCapSweep: the region below y=4 on a radius-5 sphere is 9/10 of the surface, so
// its rim sits more than a quarter turn from most of it. Every station below the rim is inside it.
func TestPointInCurvedFaceBigCapSweep(t *testing.T) {
	assertCapSweep(t, capRegionFixture(t, 5, 4, true), 5, 4, true)
}

// TestPointInCurvedFaceSmallCapSweep is the complement of the big cap on the same rim — the proof that
// the fix classifies the region, and did not merely invert the verdict.
func TestPointInCurvedFaceSmallCapSweep(t *testing.T) {
	assertCapSweep(t, capRegionFixture(t, 5, 4, false), 5, 4, false)
}

// TestPointInCurvedFaceBeltSweep: a belt bounded by TWO rims (y=-1 walked to keep what is above it,
// y=4 to keep what is below) claims the stations between them and none beyond either — the multi-loop
// case, where the region is the intersection of both rims' kept sides.
func TestPointInCurvedFaceBeltSweep(t *testing.T) {
	const r = 5
	belt := capRegionFixture(t, r, 4, true)
	belt.loops = append(belt.loops, capRegionFixture(t, r, -1, false).loops...)
	for _, y := range sweepStations(r, 4, 400) {
		if stdmath.Abs(y-4) < 0.02 || stdmath.Abs(y+1) < 0.02 {
			continue
		}
		p := meridianPoint(r, y, 0.6)
		if got, want := pointInCurvedFace(belt, p), y > -1 && y < 4; got != want {
			t.Fatalf("belt at y=%g (%v): inside=%v, want %v", y, p, got, want)
		}
	}
}

// slitHemisphereFace builds the upper (y>0) hemisphere of a radius-r sphere the way an imported STEP
// face carries it: the equator loop PLUS a meridian slit run out to the pole and straight back along the
// same curve, so the loop closes as a polygon in the parameter plane. The slit borders nothing — the
// hemisphere lies on both sides of it.
func slitHemisphereFace(t *testing.T, r float64) curvedFace {
	t.Helper()
	f := capRegionFixture(t, r, 0, false) // the equator, walked to keep the y>0 side
	meridian, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	if err != nil {
		t.Fatalf("meridian circle: %v", err)
	}
	ts, _ := geom.CurveParamAtPoint3(meridian, math.P3(math.Scalar(r), 0, 0)) // a station on the equator
	te, _ := geom.CurveParamAtPoint3(meridian, math.P3(0, math.Scalar(r), 0)) // the +Y pole
	if te-ts > 0.5 {
		te-- // walk the QUARTER arc through the kept side, not the three quarters the other way round
	}
	f.loops[0].edges = append(f.loops[0].edges, loopEdge{curve: meridian, t0: ts, t1: te},
		loopEdge{curve: meridian, t0: te, t1: ts})
	return f
}

// TestPointInCurvedFaceIgnoresASlit: the slit must not be read as a border. Reading a side from it cut
// the hemisphere down to the wedge on one side of the slit, which cost the sphere corpus its convex
// edges — S6/S7 fillets refused with "edge is not convex" (#3453 follow-up).
func TestPointInCurvedFaceIgnoresASlit(t *testing.T) {
	const r = 13
	f := slitHemisphereFace(t, r)
	sphere := f.surface.(geom.Sphere)
	for i := range 40 {
		for j := range 40 {
			u, v := 2*stdmath.Pi*float64(i)/40, -stdmath.Pi/2+stdmath.Pi*(float64(j)+0.5)/40
			p := sphere.PointAt(u, v)
			if got := pointInCurvedFace(f, p); got != (p.Y > 0) {
				t.Fatalf("slit hemisphere claims %v (u=%g v=%g) = %v, want %v", p, u, v, got, p.Y > 0)
			}
		}
	}
}

// TestBallJoinRodContainsBallCentre is the end-to-end reading of the same defect (#3453): a ball joined
// with a rod through its top keeps the ball's BIG spherical cap below the seam at y=4, and the ball's own
// centre must read as inside the solid. Every interior station read OUTSIDE while the trim classifier
// disowned the big cap, because the ray parity behind [ClassifyPoint] counts only in-trim crossings.
//
// It reads containment through ClassifyPoint (ray parity), which is the route this classifier gates. The
// SAME body read through PointInside — the nearest-crossing/flux route, which derives each face's outward
// sign from the handedness of its trim ring in (u, v) and integrates over that ring — carried a second,
// independent inversion for exactly this face: it is the COMPLEMENT of its ring in the parameter domain.
// That one lives in orient_consistent.go/winding_flux.go/trim_region.go and is pinned separately by
// orient_complement_test.go.
func TestBallJoinRodContainsBallCentre(t *testing.T) {
	ball, err := SolidSphere(math.P3(0, 0, 0), 5, "ball")
	if err != nil {
		t.Fatalf("ball: %v", err)
	}
	rod, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 1, 0), 3, 4.5)
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	join, ok := CoaxialSphereRodJoin(ball, rod)
	if !ok {
		t.Fatal("ball ∪ shoulder rod declined")
	}
	assertBigCapClaimsItsInterior(t, join)
	for _, p := range []math.Point3{math.P3(0, 0, 0), math.P3(0, -3, 0), math.P3(0, 0, 4)} {
		if got := ClassifyPoint(join, p); got != Inside {
			t.Errorf("%v reads containment %v in ball ∪ rod, want Inside", p, got)
		}
	}
	if got := ClassifyPoint(join, math.P3(9, 0, 0)); got != Outside {
		t.Errorf("a point clear of the solid reads containment %v, want Outside", got)
	}
}

// assertBigCapClaimsItsInterior checks the trim verdict on the joined body's spherical face directly, so
// a failure names the classifier rather than the ray caster that consumes it.
func assertBigCapClaimsItsInterior(t *testing.T, join *topo.Body) {
	t.Helper()
	f := soleSphereFace(t, join)
	for _, y := range []float64{-4.9, -4, -2, 0, 2, 3.9} {
		if p := meridianPoint(5, y, 0); !PointInFaceTrim(f, p) {
			t.Errorf("the joined ball's cap disowns %v (y=%g)", p, y)
		}
	}
	if p := meridianPoint(5, 4.5, 0); PointInFaceTrim(f, p) {
		t.Errorf("the joined ball's cap claims %v, which the rod replaced", p)
	}
}
