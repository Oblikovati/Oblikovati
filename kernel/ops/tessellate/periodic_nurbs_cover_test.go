// SPDX-License-Identifier: GPL-2.0-only

package tessellate

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
//
// The mouth is placed BOTH away from the seam and straddling it (#3518 review 2). The straddling case
// is the one the covering exists for and it was un-meshed by any test: with the mouth at u = 0.5 the
// replica selection never has to choose between two images of a mouth triangle, so the defect
// keepOneReplicaEach fixed was invisible here. Measured at the wave base fff94140, the straddling
// mouth at PropertyQuality came back with 103 free edges against a boundary of 100 — a three-edge
// crack that the old ceiling of "> 110" admitted.
func TestCoveringPeriodicMeshCoversFullPeriod(t *testing.T) {
	t.Parallel()
	s := closedBSplineCylinder(t, 10, 8)
	rims := []cylLoop{sampleRim(s, 0, 40), sampleRim(s, 1, 40)}
	for _, row := range []struct {
		name  string
		uc    float64
		folds map[string]int
	}{
		{"mouth away from the seam", 0.5, nil},
		// One fold edge at the DISPLAY faceting, and it is PRE-EXISTING: measured in a clean detached
		// worktree at the wave base fff94140, that row comes back byte-identical (822 triangles, 100
		// free edges, 514.7122 mm², one fold). It is pinned rather than asserted away so a rise is
		// caught and a fall says the covering stopped folding there.
		{"mouth straddling the seam", 0, map[string]int{"default": 1}},
	} {
		mouths := []cylLoop{sampleMouth(s, row.uc, 0.5, 0.08, 20)}
		for _, gq := range gateQualities() {
			m := coveringPeriodicMesh(s, gq.q, 0, 1, rims, mouths)
			assertCoveringMeshSpansThePeriod(t, row.name+", "+gq.name, m, 100, row.folds[gq.name])
		}
	}
}

// TestAMouthRemovesAreaFromTheCoveringBand is the covering mesher's only AREA gate, and it is a
// two-sided ratchet rather than an oracle, because one of the two rows it pins is WRONG.
//
// The invariant is arithmetic: cutting a mouth out of a band removes surface, so the band with a mouth
// must mesh to LESS than the same band without one. The mouth away from the seam obeys it — 465.3728
// without, 457.3523 with, at PropertyQuality. The mouth STRADDLING the seam does not: 508.2418, some
// 43 mm² MORE than the band it was cut from. That is the covering over-reporting the seam, and it is
// PRE-EXISTING: measured in a clean detached worktree at the wave base fff94140, the same six numbers
// come back, byte-identical at DefaultQuality (514.7122) and 507.7612 at PropertyQuality — the 0.48 mm²
// difference there is #3518's own three-edge crack closing, not the 43 mm².
//
// So the row pins what is, names what is wrong with it, and fails in BOTH directions: a drift means
// something moved, and the straddling number falling to below the mouthless band means the defect is
// fixed and this row should become the plain invariant for both placements.
func TestAMouthRemovesAreaFromTheCoveringBand(t *testing.T) {
	t.Parallel()
	s := closedBSplineCylinder(t, 10, 8)
	rims := []cylLoop{sampleRim(s, 0, 40), sampleRim(s, 1, 40)}
	bare := coveringPeriodicMesh(s, PropertyQuality(), 0, 1, rims, nil).Area()
	away := coveringPeriodicMesh(s, PropertyQuality(), 0, 1, rims,
		[]cylLoop{sampleMouth(s, 0.5, 0.5, 0.08, 20)}).Area()
	seam := coveringPeriodicMesh(s, PropertyQuality(), 0, 1, rims,
		[]cylLoop{sampleMouth(s, 0, 0.5, 0.08, 20)}).Area()
	if away >= bare {
		t.Errorf("a mouth away from the seam meshes %.4f mm² against a mouthless band of %.4f; cutting a "+
			"hole may not add area", away, bare)
	}
	if stdmath.Abs(seam-508.2418) > 0.01 {
		t.Errorf("a mouth STRADDLING the seam meshes %.4f mm²; the pinned measurement is 508.2418. A rise "+
			"is a regression; a fall below the mouthless %.4f means the covering has stopped over-reporting "+
			"the seam (pre-existing at fff94140, ~43 mm²) — delete this pin and gate both placements on the "+
			"invariant above", seam, bare)
	}
}

// assertCoveringMeshSpansThePeriod checks one covering mesh: non-empty, fold-free, reaching ±R in both
// x and y (the #1510 bug was a half-strip), and open along EXACTLY its own rim/mouth boundaries.
func assertCoveringMeshSpansThePeriod(t *testing.T, quality string, m *Mesh, wantFree, wantFolds int) {
	t.Helper()
	if m == nil || m.TriangleCount() == 0 {
		t.Fatalf("%s: covering mesh is empty", quality)
	}
	if folds := validate.FoldEdgeCount(m); folds != wantFolds {
		t.Errorf("%s: covering mesh has %d fold edges; want %d", quality, folds, wantFolds)
	}
	var xmin, xmax, ymin, ymax float64
	for _, p := range m.Positions {
		xmin, xmax = stdmath.Min(xmin, float64(p.X)), stdmath.Max(xmax, float64(p.X))
		ymin, ymax = stdmath.Min(ymin, float64(p.Y)), stdmath.Max(ymax, float64(p.Y))
	}
	// The whole period must be present: reach near +R and −R in BOTH x and y, not just a strip.
	if xmax < 9 || xmin > -9 || ymax < 9 || ymin > -9 {
		t.Errorf("%s: mesh does not span the full cylinder: x[%.1f,%.1f] y[%.1f,%.1f], want ±~10",
			quality, xmin, xmax, ymin, ymax)
	}
	// EXACTLY the boundary it was given — two 40-point rims and a 20-point mouth — the way the chart
	// mesher's own rim gate is exact. The old ceiling of "> 110" was 10 % above the truth and admitted
	// a three-edge crack on the seam-straddling mouth for as long as it stood (#3518 review 2).
	if free := freeEdgeCount(m); free != wantFree {
		t.Errorf("%s: mesh has %d free edges; want exactly %d (the rim+mouth boundaries, no interior crack)",
			quality, free, wantFree)
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
