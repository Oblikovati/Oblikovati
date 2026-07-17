// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// n7CanalSurface builds the N7 corner canal patch from the C1 spine + the two roll hosts — the
// object under test for the whole file.
func n7CanalSurface(t *testing.T) (BSplineSurface, []arcStation, []float64) {
	t.Helper()
	wall, s10 := n7Hosts()
	ends := n7Ends()
	res := n7Resolution(ends)
	spine, err := canalSpine([]Surface{wall, s10}, 5, ends, res)
	if err != nil {
		t.Fatalf("canalSpine: %v", err)
	}
	surf, err := loftCanal(spine, [2]Surface{wall, s10}, 5, res)
	if err != nil {
		t.Fatalf("loftCanal: %v", err)
	}
	verts, _ := SpineVertices(spine)
	cols, err := stationColumns(verts, [2]Surface{wall, s10}, 5, res)
	if err != nil {
		t.Fatalf("stationColumns: %v", err)
	}
	vParams, _ := alphaParams(coords3(verts), 1)
	return surf, cols, vParams
}

// TestLoftCanalN7AreaEmerges is the crux: the numerically-integrated corner area must EMERGE at the
// DRAWEXE oracle 90.194 from the rolling-ball geometry (radius-5 arcs on the offset-SSI spine), with
// NO tuned constant. The gate is model-relative and JUSTIFIED, not res.Weld·r² (unachievable for a
// marched-spine interpolating loft — the spike reconstructed the same geometry to only −0.025% ≈
// 0.023 abs, orders looser than res.Weld·r² ≈ 3e-6). We gate at 0.05% (2.5× the spike's demonstrated
// 0.025%), which the interpolating loft through 163 marched stations comfortably meets.
func TestLoftCanalN7AreaEmerges(t *testing.T) {
	surf, _, _ := n7CanalSurface(t)
	const oracle = 90.194
	area := surfaceArea(surf, 96, 384)
	relErr := stdmath.Abs(area-oracle) / oracle
	const gate = 0.0005 // 0.05% — justified by the spike's −0.025% reconstruction of this geometry
	t.Logf("N7 canal area = %.5f (oracle %.3f, rel err %.4f%%)", area, oracle, relErr*100)
	if relErr > gate {
		t.Errorf("area %.5f is %.4f%% off oracle %.3f (gate %.3f%%)", area, relErr*100, oracle, gate*100)
	}
	// Grid convergence: the area must be an intrinsic property of the surface, not an
	// under-integration coincidence near the oracle. A coarse and a fine grid must agree to a
	// far tighter bound than the gate, so refining the integrator cannot move the number onto 90.194.
	coarse := surfaceArea(surf, 48, 192)
	fine := surfaceArea(surf, 192, 768)
	if converge := stdmath.Abs(fine-coarse) / oracle; converge > 1e-4 {
		t.Errorf("area not grid-converged: coarse %.5f vs fine %.5f differ %.5f%% (want <0.01%%)",
			coarse, fine, converge*100)
	}
}

// TestLoftCanalN7Watertight is the load-bearing correctness gate: the four boundary isoparms ARE the
// four N7 rails, and the two foot-loci lie ON their roll hosts. v=0/v=1 are the end cross-section
// arcs (radius 5 about the end ball-centers); u=0/u=1 pass through every marched foot exactly (on
// host to weld). C3 emits Loops on these edges, so this is what makes the B-rep weld watertight.
func TestLoftCanalN7Watertight(t *testing.T) {
	surf, cols, vParams := n7CanalSurface(t)
	wall, s10 := n7Hosts()
	weld := n7Resolution(n7Ends()).Weld()

	// v=0 and v=1 isoparms: exact radius-5 arcs about the first/last spine stations, endpoints = feet.
	assertEndArc(t, surf, 0, cols[0], weld)
	assertEndArc(t, surf, 1, cols[len(cols)-1], weld)

	// u=0 (fa) and u=1 (fb) foot-loci: at each station param the isoparm hits the exact foot, which
	// lies ON its host. This is the "share the edge, keep the foot-locus on its host" watertight rule.
	maxFa := footLocusResidual(t, surf, 0, wall, cols, vParams, true, weld)
	maxFb := footLocusResidual(t, surf, 1, s10, cols, vParams, false, weld)
	t.Logf("foot-locus on-host residual: fa(wall)=%g fb(s_10)=%g (weld %g)", maxFa, maxFb, weld)
}

// TestLoftCanalN7ShapeMatchesOCCT cross-checks the emitted surface against OCCT's own 3×10 rational
// BSpline (result5-poles.txt): every point OCCT's surface samples must lie within 0.1 of OUR surface
// (parametrization-free nearest-point — OCCT's interior isoparms are diagonal, so we compare SHAPE,
// not same-(u,v); the spike measured 0.005). This proves we reproduced OCCT's geometry, not just its
// area.
func TestLoftCanalN7ShapeMatchesOCCT(t *testing.T) {
	surf, _, _ := n7CanalSurface(t)
	occt := occtResult5Surface(t)
	lo, hi := occt.UDomain()
	vlo, vhi := occt.VDomain()
	maxGap := 0.0
	// Project every OCCT sample onto OUR surface (parametrization-free true nearest point, not a
	// grid search — OCCT's interior isoparms are diagonal so same-(u,v) does not apply).
	for _, q := range sampleGrid(occt, lo, hi, vlo, vhi, 24, 48) {
		_, _, foot := ClosestPointOnSurface(surf, q)
		if g := float64(q.DistanceTo(foot)); g > maxGap {
			maxGap = g
		}
	}
	t.Logf("N7 shape gap (OCCT surface → ours, projected) = %g", maxGap)
	if maxGap > 0.1 {
		t.Errorf("shape gap %g exceeds 0.1 — emitted surface does not match OCCT's geometry", maxGap)
	}
}

// assertEndArc asserts the v-boundary isoparm at vEnd (0 or 1) is the radius-5 arc of station col:
// its center is col's ball-center (implied by the feet), endpoints are col.fa/col.fb, and every
// sampled point is radius 5 from that center.
func assertEndArc(t *testing.T, surf BSplineSurface, vEnd float64, col arcStation, weld float64) {
	t.Helper()
	if d := float64(surf.PointAt(0, vEnd).DistanceTo(col.fa)); d > weld {
		t.Errorf("v=%g isoparm u=0 corner %v != fa %v (dist %g)", vEnd, surf.PointAt(0, vEnd), col.fa, d)
	}
	if d := float64(surf.PointAt(1, vEnd).DistanceTo(col.fb)); d > weld {
		t.Errorf("v=%g isoparm u=1 corner %v != fb %v (dist %g)", vEnd, surf.PointAt(1, vEnd), col.fb, d)
	}
	center := ballCenterFromFeet(col) // the arc center recovered from the two feet
	for i := 0; i <= 32; i++ {
		p := surf.PointAt(float64(i)/32, vEnd)
		if r := stdmath.Abs(float64(p.DistanceTo(center)) - 5); r > weld*5 {
			t.Errorf("v=%g isoparm point %v: |dist(center)-5| = %g > %g", vEnd, p, r, weld*5)
		}
	}
}

// ballCenterFromFeet recovers a station's ball center (= arc center) from the feet + shoulder ALONE,
// without trusting the spine — a stronger radius check. The center lies on the chord's perpendicular
// bisector: the shoulder marks the convex (bulge) side, so the center sits on the OPPOSITE side of
// the chord midpoint, at height sqrt(radius² − (½‖fa−fb‖)²) along the shoulder direction.
func ballCenterFromFeet(col arcStation) math.Point3 {
	mid := col.fa.Midpoint(col.fb)
	toShoulder := mid.VectorTo(col.shoulder)
	half := float64(col.fa.DistanceTo(col.fb)) / 2
	d := stdmath.Sqrt(stdmath.Max(0, 25-half*half))
	n := toShoulder.Scale(1 / float64(toShoulder.Length()))
	return mid.TranslateBy(n.Scale(-d))
}

// footLocusResidual samples the u=uEnd isoparm at each station param and returns the max distance
// of those samples from host — they must be on-host to weld (the foot-locus IS the shared edge).
func footLocusResidual(t *testing.T, surf BSplineSurface, uEnd float64, host Surface,
	cols []arcStation, vParams []float64, first bool, weld float64) float64 {
	t.Helper()
	maxResid := 0.0
	for j, v := range vParams {
		p := surf.PointAt(uEnd, v)
		_, _, foot := ClosestPointOnSurface(host, p)
		if d := float64(foot.DistanceTo(p)); d > maxResid {
			maxResid = d
		}
		want := cols[j].fa
		if !first {
			want = cols[j].fb
		}
		if d := float64(p.DistanceTo(want)); d > weld {
			t.Errorf("u=%g isoparm at station %d = %v != foot %v (dist %g > weld %g)", uEnd, j, p, want, d, weld)
		}
	}
	if maxResid > weld {
		t.Errorf("u=%g foot-locus max on-host residual %g > weld %g", uEnd, maxResid, weld)
	}
	return maxResid
}

// surfaceArea integrates the surface area over its [0,1]² domain by triangulating an nu×nv grid.
func surfaceArea(surf BSplineSurface, nu, nv int) float64 {
	grid := make([][]math.Point3, nu+1)
	for i := 0; i <= nu; i++ {
		grid[i] = make([]math.Point3, nv+1)
		for j := 0; j <= nv; j++ {
			grid[i][j] = surf.PointAt(float64(i)/float64(nu), float64(j)/float64(nv))
		}
	}
	total := 0.0
	for i := 0; i < nu; i++ {
		for j := 0; j < nv; j++ {
			total += triArea(grid[i][j], grid[i+1][j], grid[i+1][j+1])
			total += triArea(grid[i][j], grid[i+1][j+1], grid[i][j+1])
		}
	}
	return total
}

// triArea is the area of triangle abc.
func triArea(a, b, c math.Point3) float64 {
	return 0.5 * float64(a.VectorTo(b).Cross(a.VectorTo(c)).Length())
}

// sampleGrid returns the grid of surface points over [u0,u1]×[v0,v1] at (nu+1)×(nv+1) stations.
func sampleGrid(surf BSplineSurface, u0, u1, v0, v1 float64, nu, nv int) []math.Point3 {
	out := make([]math.Point3, 0, (nu+1)*(nv+1))
	for i := 0; i <= nu; i++ {
		for j := 0; j <= nv; j++ {
			u := u0 + (u1-u0)*float64(i)/float64(nu)
			v := v0 + (v1-v0)*float64(j)/float64(nv)
			out = append(out, surf.PointAt(u, v))
		}
	}
	return out
}
