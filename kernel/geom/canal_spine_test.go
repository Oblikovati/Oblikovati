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
	spine, err := canalSpine([]Surface{wall, s10}, 5, ends, res)
	if err != nil {
		t.Fatalf("canalSpine: unexpected error: %v", err)
	}
	weld := res.Weld()
	if d := float64(spine.PointAt(0).DistanceTo(ends[0])); d > weld {
		t.Errorf("spine start %v != C″ %v (dist %g > weld %g)", spine.PointAt(0), ends[0], d, weld)
	}
	if d := float64(spine.PointAt(1).DistanceTo(ends[1])); d > weld {
		t.Errorf("spine end %v != C %v (dist %g > weld %g)", spine.PointAt(1), ends[1], d, weld)
	}
}

// spineSamples returns ~count evenly-spaced sample points (rolling-ball centers) of the spine.
// It samples the polyline VERTICES, not PointAt(t): a polyline's off-vertex points ride the
// chords (sag ~4e-5), so the rolling-ball invariant is a property of the spine's own samples —
// exactly the centers C2 lofts its cross-sections at. White-box: the spine is a geom.Polyline.
func spineSamples(t *testing.T, spine Curve3, count int) []math.Point3 {
	t.Helper()
	pl, ok := spine.(Polyline)
	if !ok {
		t.Fatalf("spine is %T, want geom.Polyline", spine)
	}
	verts := pl.Vertices
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
	pts, err := trimAnalyticToEnds([]Curve3{circle}, ends)
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
