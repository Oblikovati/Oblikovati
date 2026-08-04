// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/math"
)

// N7 ground truth (blend-sweep-spike-report.md): the corner face result_5 is a radius-5
// rolling-ball blend whose ball-center spine is cyl(wall,R=50-5=45) ∩ cyl(s_10,R=5+5=10),
// running from C″=(55, 50-√2000, 5) to C=(45, 50-√2000, 15). yPlane = 50-√2000 ≈ 5.27864.
// The exact closed form the spike used (option a, no OCCT poles):
//
//	m_x(z) = 55 - √(100-(z-15)²),  m_y(x) = 50 - √(2025-(x-50)²).
func n7Hosts() (wall, s10 Surface) {
	wall, _ = NewCylinder(math.P3(50, 50, 0), math.V3(0, 0, 1), 50)
	s10, _ = NewCylinder(math.P3(55, 0, 15), math.V3(0, 1, 0), 5)
	return wall, s10
}

func n7Ends() [2]math.Point3 {
	yPlane := 50 - stdmath.Sqrt(2000)
	return [2]math.Point3{
		math.P3(55, yPlane, 5),  // C″
		math.P3(45, yPlane, 15), // C
	}
}

// n7Resolution is corner-scale (the two ball-center ends), the tightest honest tolerance.
func n7Resolution(ends [2]math.Point3) Resolution {
	return ResolutionForPoints([]math.Point3{ends[0], ends[1]})
}

func TestCanalSpineN7EndpointsMatchFamilyCenters(t *testing.T) {
	wall, s10 := n7Hosts()
	ends := n7Ends()
	res := n7Resolution(ends)
	weld := res.Weld()
	spine, err := canalSpine([]Surface{wall, s10}, 5, ends, res)
	if err != nil {
		t.Fatalf("canalSpine: unexpected error: %v", err)
	}
	if d := float64(spine.PointAt(0).DistanceTo(ends[0])); d > weld {
		t.Errorf("spine start %v != C″ %v (dist %g > weld %g)", spine.PointAt(0), ends[0], d, weld)
	}
	if d := float64(spine.PointAt(1).DistanceTo(ends[1])); d > weld {
		t.Errorf("spine end %v != C %v (dist %g > weld %g)", spine.PointAt(1), ends[1], d, weld)
	}

	// The two checks above hold BY CONSTRUCTION — subRun force-snaps the boundary vertices to
	// ends — so they would still pass even if the raw march stopped well short of the true
	// endpoints (C1 review finding 1: they are vacuous as a regression guard). Reproduce
	// canalSpine's pipeline up to (but not through) the force-snap and assert the RAW,
	// pre-snap nearest traced vertex reaches each end within the same tight guard the
	// production path enforces (assertTraceReachesEnd) — this is the discriminating check.
	offs, err := offsetHostsToward([]Surface{wall, s10}, 5, ends, weld)
	if err != nil {
		t.Fatalf("offsetHostsToward: %v", err)
	}
	window := spineTraceWindow(offs[0], ends)
	curves := TraceSurfaceIntersection(offs[0], offs[1], window).Curves
	rawPoly := polylineThroughEnds(curves, ends)
	if rawPoly == nil {
		t.Fatal("polylineThroughEnds found no raw traced curve through both ends")
	}
	i0, i1 := nearestIndex(rawPoly, ends[0]), nearestIndex(rawPoly, ends[1])
	if err := assertTraceReachesEnd(rawPoly, i0, ends[0], weld); err != nil {
		t.Errorf("raw (pre-snap) trace does not reach C″: %v", err)
	}
	if err := assertTraceReachesEnd(rawPoly, i1, ends[1], weld); err != nil {
		t.Errorf("raw (pre-snap) trace does not reach C: %v", err)
	}
}

// TestTrimTracedToEndsRejectsShortfall proves the endpoint guard added for C1 review finding 1
// actually discriminates: a traced polyline that stops ~1 unit short of a family-center end must
// be REJECTED by trimTracedToEnds, not silently force-snapped by subRun. This is the regression
// TestCanalSpineN7EndpointsMatchFamilyCenters's post-snap PointAt(0)==ends[0] assertion alone could
// never catch — that equality holds after the unconditional overwrite regardless of how far short
// the raw trace actually got. Mutation evidence (manually, by temporarily neutering the guard):
// disabling assertTraceReachesEnd's call site turns this test's "want reject" branch into a FAIL
// ("trimTracedToEnds accepted a trace that stops ~1 unit short..."), confirming it fires; restoring
// the guard turns it back to PASS.
func TestTrimTracedToEndsRejectsShortfall(t *testing.T) {
	ends := n7Ends()
	res := n7Resolution(ends)
	weld := res.Weld()

	const step = 0.05
	healthy := straightSteps(ends[1], ends[0], step) // dense straight trace, ends[1] -> ends[0]
	if _, err := trimTracedToEnds([][]math.Point3{healthy}, ends, weld); err != nil {
		t.Fatalf("trimTracedToEnds rejected a healthy trace reaching both ends: %v", err)
	}

	// Drop the last ~1.0 units of coverage near ends[0] (20 steps of 0.05) — a stand-in for a
	// tracer that stopped short of the true endpoint.
	dropped := int(1.0 / step)
	short := healthy[:len(healthy)-dropped]
	_, err := trimTracedToEnds([][]math.Point3{short}, ends, weld)
	if err == nil {
		t.Fatal("trimTracedToEnds accepted a trace that stops ~1 unit short of ends[0]; want reject")
	}
	if !strings.Contains(err.Error(), "stopped short") {
		t.Errorf("shortfall error should name the condition, got: %v", err)
	}
}

// straightSteps returns points stepped by `step` along the straight segment from a to b, ending
// exactly at b — a synthetic stand-in for a traced polyline, used to test the endpoint guard in
// isolation from the real SSI tracer.
func straightSteps(a, b math.Point3, step float64) []math.Point3 {
	d := float64(a.DistanceTo(b))
	n := int(stdmath.Ceil(d / step))
	dir := a.VectorTo(b)
	pts := make([]math.Point3, 0, n+1)
	for i := 0; i < n; i++ {
		pts = append(pts, a.TranslateBy(dir.Scale(float64(i)*step/d)))
	}
	return append(pts, b)
}

// spineSamples returns ~count evenly-spaced sample points (rolling-ball centers) of the spine.
// It reads the polyline VERTICES via [SpineVertices], not PointAt(t): a polyline's off-vertex
// points ride the chords (sag ~4e-5), so the rolling-ball invariant is a property of the spine's
// own samples — exactly the centers C2 lofts its cross-sections at (C1 review finding 2: this
// used to reach past canalSpine's Curve3 return with a fragile spine.(Polyline) type-assertion).
func spineSamples(t *testing.T, spine Curve3, count int) []math.Point3 {
	t.Helper()
	verts, ok := SpineVertices(spine)
	if !ok {
		t.Fatalf("spine is %T, want a spine curve exposing vertices (SpineVertices)", spine)
	}
	out := make([]math.Point3, 0, count+1)
	for i := 0; i <= count; i++ {
		out = append(out, verts[i*(len(verts)-1)/count])
	}
	return out
}

func TestCanalSpineN7PointsAtRadiusFromBothHosts(t *testing.T) {
	wall, s10 := n7Hosts()
	ends := n7Ends()
	res := n7Resolution(ends)
	spine, err := canalSpine([]Surface{wall, s10}, 5, ends, res)
	if err != nil {
		t.Fatalf("canalSpine: unexpected error: %v", err)
	}
	weld := res.Weld()
	maxResid := 0.0
	for _, p := range spineSamples(t, spine, 10) {
		for name, host := range map[string]Surface{"wall": wall, "s_10": s10} {
			_, _, foot := ClosestPointOnSurface(host, p)
			resid := stdmath.Abs(float64(foot.DistanceTo(p)) - 5)
			if resid > maxResid {
				maxResid = resid
			}
			if resid > weld {
				t.Errorf("spine point %v: |dist(%s)=%g - 5| = %g > weld %g",
					p, name, float64(foot.DistanceTo(p)), resid, weld)
			}
		}
	}
	t.Logf("max |dist-5| on-host residual over sampled spine points = %g (weld %g)", maxResid, weld)
}

// The spine must satisfy the spike's exact closed form m(z) (it IS cyl45 ∩ cyl10), and sit in
// the y≈5.27864 slab. This is the parametrization-free cross-check against the spike ground truth.
func TestCanalSpineN7MatchesSpikeClosedForm(t *testing.T) {
	wall, s10 := n7Hosts()
	ends := n7Ends()
	res := n7Resolution(ends)
	spine, err := canalSpine([]Surface{wall, s10}, 5, ends, res)
	if err != nil {
		t.Fatalf("canalSpine: unexpected error: %v", err)
	}
	yPlane := 50 - stdmath.Sqrt(2000)
	for _, p := range spineSamples(t, spine, 10) {
		x, y, z := float64(p.X), float64(p.Y), float64(p.Z)
		wantX := 55 - stdmath.Sqrt(clampNonNeg(100-(z-15)*(z-15)))
		wantY := 50 - stdmath.Sqrt(clampNonNeg(2025-(x-50)*(x-50)))
		if stdmath.Abs(x-wantX) > 1e-6 {
			t.Errorf("spine x=%g at z=%g != closed form %g", x, z, wantX)
		}
		if stdmath.Abs(y-wantY) > 1e-6 {
			t.Errorf("spine y=%g at x=%g != closed form %g", y, x, wantY)
		}
		// The spine dips to y=5.0 at x=50 (min of m_y); spike max |Δy|=0.29 grounds the 0.3 slab.
		if stdmath.Abs(y-yPlane) > 0.3 {
			t.Errorf("spine y=%g strayed from the y≈%g slab (|Δy| %g > 0.3)", y, yPlane, stdmath.Abs(y-yPlane))
		}
	}
}

func clampNonNeg(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}

// Point-spine: coincident ball centers (the three offsets concur) → honest reject carrying the gap.
func TestCanalSpineRejectsPointSpine(t *testing.T) {
	wall, s10 := n7Hosts()
	ends := n7Ends()
	res := n7Resolution(ends)
	degenerate := [2]math.Point3{ends[1], ends[1]} // C == C
	_, err := canalSpine([]Surface{wall, s10}, 5, degenerate, res)
	if err == nil {
		t.Fatal("canalSpine accepted a point-spine (coincident ends); want reject")
	}
	if !strings.Contains(err.Error(), "point-spine") {
		t.Errorf("point-spine error should name the condition, got: %v", err)
	}
}

// Offset self-intersection: r=10 inner offset of a radius-5 cylinder folds → honest reject.
func TestCanalSpineRejectsOffsetSelfIntersection(t *testing.T) {
	wall, _ := n7Hosts()
	tight, _ := NewCylinder(math.P3(55, 0, 15), math.V3(0, 1, 0), 5)
	// Ends INSIDE the tight cylinder → its cavity side is the inner (-r) offset; r=10 > R=5 folds.
	ends := [2]math.Point3{math.P3(54, 5, 15), math.P3(56, 5, 15)}
	res := n7Resolution(ends)
	_, err := canalSpine([]Surface{wall, tight}, 10, ends, res)
	if err == nil {
		t.Fatal("canalSpine accepted a folding inner offset (r=10 on R=5 cylinder); want reject")
	}
	if !strings.Contains(err.Error(), "self-intersect") {
		t.Errorf("offset-fold error should name self-intersection, got: %v", err)
	}
}

// trimAnalyticToEnds is the closed-form-SSI trimmer (kept for future concrete-primitive spines;
// the offset-SSI operands are OffsetSurface-typed, so IntersectSurfacesAnalytic returns
// handled=false and the marched path carries N7). Exercise it directly on an analytic arc.
func TestTrimAnalyticToEndsSnapsToEnds(t *testing.T) {
	circle, err := NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	ends := [2]math.Point3{math.P3(5, 0, 0), math.P3(0, 5, 0)} // 0 and 90° on the circle
	weld := n7Resolution(ends).Weld()
	pts, err := trimAnalyticToEnds([]Curve3{circle}, ends, weld)
	if err != nil {
		t.Fatalf("trimAnalyticToEnds: %v", err)
	}
	if len(pts) < 2 {
		t.Fatalf("trimAnalyticToEnds returned %d points, want a run", len(pts))
	}
	if d := float64(pts[0].DistanceTo(ends[0])); d != 0 {
		t.Errorf("run start %v not snapped to end %v (dist %g)", pts[0], ends[0], d)
	}
	if d := float64(pts[len(pts)-1].DistanceTo(ends[1])); d != 0 {
		t.Errorf("run end %v not snapped to end %v (dist %g)", pts[len(pts)-1], ends[1], d)
	}
}

func TestCanalSpineRejectsWrongHostCount(t *testing.T) {
	wall, _ := n7Hosts()
	ends := n7Ends()
	_, err := canalSpine([]Surface{wall}, 5, ends, n7Resolution(ends))
	if err == nil {
		t.Fatal("canalSpine accepted a single host; want exactly 2")
	}
}
