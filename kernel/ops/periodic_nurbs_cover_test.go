// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/math"
)

// closedBSplineCylinder builds a B-spline surface that is closed (periodic) in u and open in v — the
// minimal stand-in for an imported barrel: a ring of control columns whose last column equals the first
// (coincident endpoints, so PointAt(0,v)=PointAt(1,v)) over two v-rows (bottom z=0, top z=height).
func closedBSplineCylinder(t *testing.T, radius, height float64) geom.BSplineSurface {
	t.Helper()
	const cols = 9 // last column repeats the first to close the loop
	ctrl := make([][]math.Point3, cols)
	weights := make([][]float64, cols)
	for c := range cols {
		ang := 2 * stdmath.Pi * float64(c) / float64(cols-1)
		x, y := radius*stdmath.Cos(ang), radius*stdmath.Sin(ang)
		ctrl[c] = []math.Point3{math.P3(x, y, 0), math.P3(x, y, height)}
		weights[c] = []float64{1, 1}
	}
	s, err := geom.NewBSplineSurface(2, 1, ctrl, weights, clampedUniformKnots(cols, 2), clampedUniformKnots(2, 1))
	if err != nil {
		t.Fatalf("build closed B-spline cylinder: %v", err)
	}
	return s
}

// clampedUniformKnots returns a clamped uniform knot vector for count control points of the given degree.
func clampedUniformKnots(count, deg int) []float64 {
	n := count + deg + 1
	interior := count - deg - 1
	k := make([]float64, n)
	for i := range n {
		switch {
		case i <= deg:
			k[i] = 0
		case i >= n-1-deg:
			k[i] = 1
		default:
			k[i] = float64(i-deg) / float64(interior+1)
		}
	}
	return k
}

// sampleRim samples a constant-v circle on the surface as a rim cylLoop (u advances 0→~1, wrapping).
func sampleRim(s geom.BSplineSurface, v float64, n int) cylLoop {
	l := cylLoop{}
	for i := range n {
		u := float64(i) / float64(n)
		l.p3 = append(l.p3, s.PointAt(u, v))
		l.u = append(l.u, u)
		l.v = append(l.v, v)
	}
	return l
}

// sampleMouth samples a small (u,v) circle on the surface as a mouth cylLoop.
func sampleMouth(s geom.BSplineSurface, cu, cv, r float64, n int) cylLoop {
	l := cylLoop{}
	for i := range n {
		a := 2 * stdmath.Pi * float64(i) / float64(n)
		u, v := cu+r*stdmath.Cos(a), cv+r*stdmath.Sin(a)
		l.p3 = append(l.p3, s.PointAt(u, v))
		l.u = append(l.u, u)
		l.v = append(l.v, v)
	}
	return l
}

// TestSurfaceClosedInUNotV pins the closure detector: the synthetic cylinder is closed in u (the seam
// joins) and open in v (bottom≠top), so only the u-periodic covering path applies.
func TestSurfaceClosedInUNotV(t *testing.T) {
	t.Parallel()
	s := closedBSplineCylinder(t, 10, 8)
	if !surfaceClosedInU(s) {
		t.Error("cylinder must be detected closed in u")
	}
	if surfaceClosedInV(s) {
		t.Error("cylinder must be detected open in v")
	}
}

// TestCoveringPeriodicMeshCoversFullPeriod drives the covering CDT end to end on the synthetic cylinder
// with two rims and one interior mouth — exercising replicateBoundary, the interior grid, canonical
// selection and the seam weld. It asserts the mesh is fold-free and spans the FULL angular period (the
// #1510 bug was a half-strip), and that its only open edges are the band's own rim/mouth boundaries (no
// interior crack) — an isolated tube band is legitimately open top and bottom.
func TestCoveringPeriodicMeshCoversFullPeriod(t *testing.T) {
	t.Parallel()
	s := closedBSplineCylinder(t, 10, 8)
	rims := []cylLoop{sampleRim(s, 0, 40), sampleRim(s, 1, 40)}
	mouths := []cylLoop{sampleMouth(s, 0.5, 0.5, 0.08, 20)}
	for _, gq := range gateQualities() {
		assertCoveringMeshSpansThePeriod(t, gq.name, coveringPeriodicMesh(s, gq.q, 0, 1, rims, mouths))
	}
}

// assertCoveringMeshSpansThePeriod checks one covering mesh: non-empty, fold-free, reaching ±R in both
// x and y (the #1510 bug was a half-strip), and open only along its own rim/mouth boundaries.
func assertCoveringMeshSpansThePeriod(t *testing.T, quality string, m *Mesh) {
	t.Helper()
	if m == nil || m.TriangleCount() == 0 {
		t.Fatalf("%s quality: covering mesh is empty", quality)
	}
	if folds := validate.FoldEdgeCount(m); folds != 0 {
		t.Errorf("%s quality: covering mesh has %d fold edges; want 0", quality, folds)
	}
	var xmin, xmax, ymin, ymax float64
	for _, p := range m.Positions {
		xmin, xmax = stdmath.Min(xmin, float64(p.X)), stdmath.Max(xmax, float64(p.X))
		ymin, ymax = stdmath.Min(ymin, float64(p.Y)), stdmath.Max(ymax, float64(p.Y))
	}
	// The whole period must be present: reach near +R and −R in BOTH x and y, not just a strip.
	if xmax < 9 || xmin > -9 || ymax < 9 || ymin > -9 {
		t.Errorf("%s quality: mesh does not span the full cylinder: x[%.1f,%.1f] y[%.1f,%.1f], want ±~10",
			quality, xmin, xmax, ymin, ymax)
	}
	if free := freeEdgeCount(m); free > 110 {
		t.Errorf("%s quality: mesh has %d free edges; want ≈100 (the rim+mouth boundaries only, no interior crack)",
			quality, free)
	}
}

// TestInterpRimInterpolatesV pins the band-boundary interpolation v(u) used by the material test —
// including ACROSS THE SEAM. rimSamples folds a rim's u to canonical and sorts it, so the segment from
// the last sample (0.9 here) back to the first (0.1) is not in the list; the interpolant must close it
// over the period rather than clamp flat at both ends. The seam midpoint u = 0 ≡ u = 1 lies exactly
// halfway between v = 3 and v = 1, so its exact value is 2 — a closed-form target, not a captured one.
func TestInterpRimInterpolatesV(t *testing.T) {
	t.Parallel()
	f := interpRim([][2]float64{{0.1, 1}, {0.5, 2}, {0.9, 3}}, 1)
	for _, c := range []struct{ u, want float64 }{{0.3, 1.5}, {0.7, 2.5}, {0.0, 2}, {1.0, 2}, {0.05, 1.5}, {0.95, 2.5}} {
		if got := f(c.u); stdmath.Abs(got-c.want) > 1e-9 {
			t.Errorf("interpRim(%.2f) = %.4f, want %.4f", c.u, got, c.want)
		}
	}
	if lo, hi := f(0.0), f(1.0); stdmath.Abs(lo-hi) > 1e-12 {
		t.Errorf("the interpolant is not periodic: f(0) = %.12f but f(1) = %.12f", lo, hi)
	}
}

// TestInterpRimIsFlatOnAConstantVRim pins the no-op case that the shipped population actually is: both of
// cand_radial's rims are constant-v (measured vspan 1.9e-17 and 0), so closing the seam segment must
// leave the interpolant identically flat — the receipt that this change moves no shipped mesh.
func TestInterpRimIsFlatOnAConstantVRim(t *testing.T) {
	t.Parallel()
	f := interpRim([][2]float64{{0.1, 7}, {0.4, 7}, {0.8, 7}}, 1)
	for _, u := range []float64{0, 0.05, 0.25, 0.6, 0.9, 1} {
		if got := f(u); got != 7 {
			t.Errorf("interpRim(%.2f) = %.15f on a constant-v rim; want exactly 7", u, got)
		}
	}
}

// TestMaterialPointInsideBandOutsideMouth pins the region test: inside the band and clear of the mouth is
// material; below the bottom rim, above the top rim, or inside the mouth is not.
func TestMaterialPointInsideBandOutsideMouth(t *testing.T) {
	t.Parallel()
	vBot := func(float64) float64 { return 0 }
	vTop := func(float64) float64 { return 1 }
	mouths := []cylLoop{{u: []float64{0.4, 0.6, 0.6, 0.4}, v: []float64{0.4, 0.4, 0.6, 0.6}}}
	if !materialPoint(0.2, 0.5, 1, vBot, vTop, mouths, 0) {
		t.Error("a band point clear of the mouth must be material")
	}
	if materialPoint(0.5, 0.5, 1, vBot, vTop, mouths, 0) {
		t.Error("a point inside the mouth must not be material")
	}
	if materialPoint(0.2, -0.1, 1, vBot, vTop, mouths, 0) {
		t.Error("a point below the bottom rim must not be material")
	}
}

// TestClassifyCylinderLoops pins that a full-period wrap is a rim and a localized loop is a mouth.
func TestClassifyCylinderLoops(t *testing.T) {
	t.Parallel()
	rim := cylLoop{u: []float64{0, 0.25, 0.5, 0.75, 0.97}, v: []float64{0, 0, 0, 0, 0}}
	mouth := cylLoop{u: []float64{0.40, 0.46, 0.51, 0.46}, v: []float64{0.3, 0.3, 0.7, 0.7}}
	rims, mouths := classifyCylinderLoops([]cylLoop{rim, mouth}, 1)
	if len(rims) != 1 || len(mouths) != 1 {
		t.Fatalf("classify: rims=%d mouths=%d, want 1 and 1", len(rims), len(mouths))
	}
}

// TestWeldCoverTrianglesMergesCoincident pins that coincident 3D positions (period-shifted seam copies)
// collapse to one vertex, so the compacted triangle references the shared index.
func TestWeldCoverTrianglesMergesCoincident(t *testing.T) {
	t.Parallel()
	pos := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(0, 0, 0)}
	nrm := make([]math.Vector3, 4)
	outPos, _, tris := weldCoverTriangles(pos, nrm, [][3]int{{0, 1, 2}, {3, 1, 2}})
	if len(outPos) != 3 {
		t.Errorf("welded vertex count = %d, want 3 (index 3 ≡ index 0)", len(outPos))
	}
	if len(tris) != 2 {
		t.Errorf("kept %d triangles, want 2", len(tris))
	}
}

// TestConstrainedTriangulationAllCoversHull pins that the flood-free triangulation returns the full set of
// non-super triangles of a square (two triangles), the entry the covering selector consumes.
func TestConstrainedTriangulationAllCoversHull(t *testing.T) {
	t.Parallel()
	pts := [][2]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
	tris := constrainedTriangulationAll(pts, [][]int{{0, 1, 2, 3}})
	if len(tris) != 2 {
		t.Errorf("triangulated square into %d triangles, want 2", len(tris))
	}
}

// TestDistToSeg2D pins the point-to-segment distance used in the mouth-clearance margin test.
func TestDistToSeg2D(t *testing.T) {
	t.Parallel()
	if d := distToSeg2D(0.5, 1, 0, 0, 1, 0); stdmath.Abs(d-1) > 1e-9 {
		t.Errorf("distToSeg2D perpendicular = %.4f, want 1", d)
	}
	if d := distToSeg2D(2, 0, 0, 0, 1, 0); stdmath.Abs(d-1) > 1e-9 {
		t.Errorf("distToSeg2D past-endpoint = %.4f, want 1", d)
	}
}
