// SPDX-License-Identifier: GPL-2.0-only

package tessellate_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The corpus of the chart-driven curved-face mesher (ADR-0061).
//
// Every row here is a body whose curved face used to be meshed over the surface's WHOLE parametric
// domain — carrying material the face does not have, omitting the face's own boundary, and leaving the
// neighbours nothing to meet. Each asserts the three things that says: the body is watertight, it
// reports no discarded trim, and its volume is a chord deficit rather than a factor off the analytic
// value the boolean's own certified corpus carries.

// chartCorpusRow is one measured body and how its mesh volume is held against the analytic value.
//
// A row the chart mesher OWNS carries maxRel: its error is a chord deficit and nothing more, and any
// improvement to the faceting is welcome. A row it does not own yet carries a PIN — the relative error
// measured today and a narrow window round it — because a one-sided bound on such a row is not a
// ratchet: it would sit green through a drift halfway to the bound AND through the fix that is one
// slice away. Two-sided, both trip it.
type chartCorpusRow struct {
	name      string
	body      func(t *testing.T) *topo.Body
	want      float64
	maxRel    float64 // the chord-deficit bound, for a row the chart mesher owns
	pinnedRel float64 // the relative error measured today, for a row it does not
	pinWindow float64
}

// ringMinus builds `ring − tool` for the corpus rows that bore a torus.
func ringMinus(t *testing.T, tool *topo.Body) *topo.Body {
	t.Helper()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	body, err := ops.Boolean(ops.Cut, ring, tool)
	if err != nil {
		t.Fatalf("ring − tool: %v", err)
	}
	return body
}

// mustCylinder builds a cylindrical tool, failing the test rather than returning an error.
func mustCylinder(t *testing.T, base math.Point3, axis math.Vector3, radius, height float64) *topo.Body {
	t.Helper()
	b, err := brep.SolidCylinder(base, axis, radius, height)
	if err != nil {
		t.Fatalf("cylinder r=%g h=%g: %v", radius, height, err)
	}
	return b
}

// mustSphere builds a spherical tool, failing the test rather than returning an error.
func mustSphere(t *testing.T, centre math.Point3, radius float64) *topo.Body {
	t.Helper()
	b, err := brep.SolidSphere(centre, radius, "ball")
	if err != nil {
		t.Fatalf("sphere r=%g: %v", radius, err)
	}
	return b
}

// mustBlock builds a block tool, failing the test rather than returning an error.
func mustBlock(t *testing.T, lo, hi math.Point3) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(lo, hi, "box")
	if err != nil {
		t.Fatalf("block: %v", err)
	}
	return b
}

// rodAndBall are the folded-window pair: a rod with a ball set into its side.
func rodAndBall(t *testing.T) (*topo.Body, *topo.Body) {
	t.Helper()
	rod := mustCylinder(t, math.P3(0, 0, -2), math.V3(0, 0, 1), 1, 4)
	ball := mustSphere(t, math.P3(1.4, 0, 0), 0.5)
	return rod, ball
}

// rodBall builds one operation of the folded-window pair.
func rodBall(t *testing.T, op ops.PartFeatureOperation) *topo.Body {
	t.Helper()
	rod, ball := rodAndBall(t)
	body, err := ops.Boolean(op, rod, ball)
	if err != nil {
		t.Fatalf("rod/ball: %v", err)
	}
	return body
}

// chartCorpus is the measured roster. The tolerances are per row and each says what sets it.
func chartCorpus() []chartCorpusRow {
	return []chartCorpusRow{
		// The representative face: a coaxial shaft bored through a ring leaves a torus band that wraps
		// the ring's AZIMUTH. Before: the whole torus, 144.78 against 203.59 with 64 free edges.
		{name: "RS− ring − coaxial shaft", want: 203.59, maxRel: 0.05, body: func(t *testing.T) *topo.Body {
			return ringMinus(t, mustCylinder(t, math.P3(0, 0, -4), math.V3(0, 0, 1), 4, 8))
		}},
		// A torus carrying TWO drill windows and no outer loop. Before: the whole torus grid, SILENTLY
		// (the one-window complement mesher declined a second window and fell through without a word),
		// 64 free edges.
		{name: "RD− ring − axial drill", want: 216.26, maxRel: 0.05, body: func(t *testing.T) *topo.Body {
			return ringMinus(t, mustCylinder(t, math.P3(5, 0, -4), math.V3(0, 0, 1), 0.8, 8))
		}},
		// The folded-window family. Both faces are the chart mesher's now: the ball's bulge (a sphere
		// minus a window, once the whole ball with 28 free edges) and the rod's WALL, which the unroll
		// used to claim — it meshed 24.47 mm² of wall with triangles whose planes pass as close as 0.5
		// to the axis, so the wall integrated 7.19 where 8.26 is right, and the two rows sat PINNED at
		// 0.0872 and 0.0902 with that number written down as what would move them.
		//
		// It moved. The unroll now keeps only the bands the general covering cannot serve — an
		// uncharted one, or one whose lens windows nearly pinch (twoRimHoledTrimOf) — and these two
		// measure 0.0144 and 0.0143, exactly the 1.44 %/1.43 % the pin predicted. They are bounded
		// rows again, at 0.02: a chord deficit is all that is left, and pinning one would fail the day
		// the faceting improves.
		{name: "RODB∪ rod ∪ ball", want: 13.077910, maxRel: 0.02,
			body: func(t *testing.T) *topo.Body { return rodBall(t, ops.Join) }},
		{name: "RODB− rod − ball", want: 12.555898, maxRel: 0.02,
			body: func(t *testing.T) *topo.Body { return rodBall(t, ops.Cut) }},
		// RODB∩'s two faces are small single-loop patches chorded flat across a lens 0.1 deep. The rod
		// WALL's carries a chart and no special shape claims it, so the classification names it
		// kindChart and it is meshed from that chart: 0.30200 → 0.27410 (measured, 60 triangles, 0 free
		// edges). The BALL's face is arc-bounded, so it classifies as kindSpherePatch and still chords
		// flat across the lens — that is what the remaining 27% is. (Verified by logging the kind of
		// both faces: sphere → sphere-patch, cylinder → chart.) Pinned so the pair's third operation
		// cannot move, in either direction, without saying so.
		//
		// 0.2741 → 0.2810 (Task 7): re-MEASURED, not widened. Giving a STRAIGHT covering axis the cell
		// size the other axis's chord asks for (balancedCoverGrid) refines the rod wall's lens patch,
		// and a finer faceting of a lens 0.1 mm deep on a body of 0.012 mm³ takes a little more volume
		// out of it. The body it belongs to is unchanged everywhere else.
		{name: "RODB∩ rod ∩ ball", want: 0.012187, pinnedRel: 0.2810, pinWindow: 0.005,
			body: func(t *testing.T) *topo.Body { return rodBall(t, ops.Intersect) }},
	}
}

// TestChartedFacesMeshTheirOwnRegion is the corpus gate: every body whose curved face carries a chart
// meshes watertight, reports no discarded trim, and integrates to its analytic volume.
func TestChartedFacesMeshTheirOwnRegion(t *testing.T) {
	t.Parallel()
	for _, row := range chartCorpus() {
		body := row.body(t)
		mesh, _ := tessellate.TessellateBody(body, ops.DefaultQuality())
		if free := tessellate.FreeEdgeCount(mesh); free != 0 {
			t.Errorf("%s meshed with %d free edges, want a watertight mesh", row.name, free)
		}
		if meshReportsIgnoredTrim(mesh) {
			t.Errorf("%s reported a discarded trim; the chart mesher charts it now", row.name)
		}
		assertCorpusVolume(t, row, tessellate.MeshGeometryProperties(mesh).Volume)
	}
}

// assertCorpusVolume holds one row's mesh volume against its analytic value, the way the row asks to
// be held: a bound for a row the chart mesher owns, a two-sided pin for one it does not. A row that
// carries NEITHER asserts nothing about its volume, which would make it a watertightness test wearing
// a corpus row's clothes, so that is a failure of the row itself.
func assertCorpusVolume(t *testing.T, row chartCorpusRow, got float64) {
	t.Helper()
	rel := stdmath.Abs(got-row.want) / row.want
	if row.maxRel <= 0 && row.pinWindow <= 0 {
		t.Errorf("%s carries neither a bound nor a pin, so its volume of %.5f is unchecked", row.name, got)
	}
	if row.maxRel > 0 && rel > row.maxRel {
		t.Errorf("%s meshes to %.5f against an analytic %.5f (rel %.4f > %.4f); that is not a chord deficit",
			row.name, got, row.want, rel, row.maxRel)
	}
	if row.pinWindow > 0 && stdmath.Abs(rel-row.pinnedRel) > row.pinWindow {
		t.Errorf("%s meshes to %.5f against an analytic %.5f (rel %.4f), off its pin of %.4f ± %.4f — "+
			"a regression, or the wall mesher was retired and this row is now the chart mesher's to bound",
			row.name, got, row.want, rel, row.pinnedRel, row.pinWindow)
	}
}

// TestTheGenusOneComplementIsChartedNotWindowed is the re-pointed row of the deleted
// torusComplementMesh (Oblikovati#1375): a torus cut by an axis-parallel half-space keeps the whole
// tube minus ONE oval cap — a torus-minus-disk, outerless, with the oval as its only boundary. The
// window-and-patch construction that used to mesh it is gone and the chart-driven mesher takes the face.
//
// Its section under x = R is the LEMNISCATE, so the boundary passes through the same 3D point twice and
// this row is what the boundary clearance is swept on (chartBoundaryClearance). Re-measured there:
// 201.258 against an analytic 203.905 at DefaultQuality, 1.30 % — the comment said 1.11 % from before the
// sweep, and the number moved with the constant. At PropertyQuality it is 203.869, 0.017 %.
//
// The torus FACE's own area is pinned two-sided beside the body volume, because the clearance trades
// interior density for the boundary and only a per-face reading shows what it costs.
func TestTheGenusOneComplementIsChartedNotWindowed(t *testing.T) {
	t.Parallel()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	box := mustBlock(t, math.P3(5, -20, -20), math.P3(20, 20, 20))
	body, err := ops.Boolean(ops.Cut, ring, box)
	if err != nil {
		t.Fatalf("ring − half space: %v", err)
	}
	an, ok := query.AnalyticGeometryProperties(body)
	if !ok {
		t.Fatal("the analytic integrator declined the cut ring; the row needs an oracle it does not read from the mesh")
	}
	mesh, _ := tessellate.TessellateBody(body, ops.DefaultQuality())
	if free := tessellate.FreeEdgeCount(mesh); free != 0 {
		t.Errorf("the genus-1 complement meshed with %d free edges, want a watertight mesh", free)
	}
	if hasIgnoredTrim(t, body) {
		t.Error("the genus-1 complement reported a discarded trim")
	}
	got := tessellate.MeshGeometryProperties(mesh).Volume
	// 0.02, not the 0.05 the other rows carry: this row is MEASURED at 1.30% (201.258 against 203.905),
	// and a bound four times looser than the reading would sit green through the window mesher's own
	// 1.39% — the very number this row exists to have beaten.
	if rel := stdmath.Abs(got-an.Volume) / an.Volume; rel > 0.02 {
		t.Errorf("the genus-1 complement meshes to %.5f against the analytic %.5f (rel %.4f > 0.02)", got, an.Volume, rel)
	}
	assertComplementFaceArea(t, body)
}

// complementTorusFaceArea is the torus face's own meshed area at DefaultQuality, pinned TWO-SIDED: the
// boundary clearance buys the lemniscate's watertightness with interior density next to a coarse
// boundary, and this is the only place that price is visible.
//
// The window is 0.05 mm², not the 0.5 a first attempt used, because 0.5 pinned nothing: it admitted BOTH
// values the clearance moved between (263.42317 at k = 1.0 and 263.72994 at k = 0.5), so a row meant to
// catch the constant drifting would have sat green through exactly that. Tessellation is byte-identical
// run to run by ground rule, so the reading is exact; the window absorbs only the five decimals this
// literal is written to (1e-5) and the last-place spread an FMA-contracting toolchain gives an area sum.
// 0.05 is four decades above that and a factor of 2.6 below the nearest reading it has to exclude.
//
// Measured at the shipped k = 0.875. Across the sweep the same face reads 263.72994 (k ≤ 0.55),
// 263.68219 (0.6), 263.60871 (0.7–0.75), 263.55487 (0.875), 263.45103 (0.925), 263.42317 (1.0),
// 263.15360 (1.5), 260.51898 (3.0) — monotone in k, which is what a clearance that only ever REMOVES
// interior nodes must be.
const complementTorusFaceArea, complementTorusFaceWindow = 263.55487, 0.05

// assertComplementFaceArea holds the complement's torus face to its measured area, both ways.
func assertComplementFaceArea(t *testing.T, body *topo.Body) {
	t.Helper()
	for _, f := range body.Faces() {
		if _, isTorus := f.Geometry().(geom.Torus); !isTorus {
			continue
		}
		got := tessellate.MeshGeometryProperties(tessellate.TessellateFace(f, ops.DefaultQuality())).Area
		if stdmath.Abs(got-complementTorusFaceArea) > complementTorusFaceWindow {
			t.Errorf("the complement's torus face meshes %.5f mm², off its pin of %.5f ± %g — the "+
				"boundary clearance moved, or the covering's density next to a coarse rim did",
				got, complementTorusFaceArea, complementTorusFaceWindow)
		}
		return
	}
	t.Fatal("the genus-1 complement has no torus face; the area pin covers nothing")
}

// TestTheAzimuthBandMeshesOneTurnNotTwo is the PER-FACE gate on the representative row, which a body
// volume cannot give: a band that wraps a period is meshed once or it is meshed twice, and both
// answers close.
//
// A shaft of radius 4 bored through the ring keeps the torus where R + r·cos v > 4, so cos v > −2/3 and
// the band is |v| < 2.3005. Its area is 2π·r·∫(R + r·cos v)dv over that range = 237.87 mm², against
// 296.09 for the whole torus and 58.22 for the band the shaft removed.
func TestTheAzimuthBandMeshesOneTurnNotTwo(t *testing.T) {
	t.Parallel()
	bored := ringMinus(t, mustCylinder(t, math.P3(0, 0, -4), math.V3(0, 0, 1), 4, 8))
	faces, meshes := tessellate.TessellateBodyFaces(bored, ops.DefaultQuality())
	found := false
	for i, f := range faces {
		if _, isTorus := f.Geometry().(geom.Torus); !isTorus {
			continue
		}
		found = true
		got := tessellate.MeshGeometryProperties(meshes[i]).Area
		if rel := stdmath.Abs(got-237.87) / 237.87; rel > 0.05 {
			t.Errorf("the azimuth band meshes %.4f mm² against an analytic 237.87 (rel %.4f); "+
				"the whole torus is 296.09 and the removed band 58.22", got, rel)
		}
	}
	if !found {
		t.Fatal("the bored ring has no torus face; the corpus row no longer tests what it says")
	}
}

// chartPoleSliverFloor is the fraction of the mean triangle area the SMALLEST triangle of a
// pole-containing charted face must reach.
//
// A sphere's pole row is a whole row of covering-space nodes at one 3D point. Collapsed, it welds to a
// single vertex and the triangles that used two of its copies vanish, leaving a clean fan; NOT
// collapsed, those triangles survive with two coincident corners and an area of zero. Measured on the
// rod ∪ ball sphere the worst triangle is 0.2986 of the mean, so this floor sits six times below what
// the mesher achieves and far above anything a surviving sliver could reach.
const chartPoleSliverFloor = 0.05

// TestThePoleRowCollapsesToOneVertex is brief item 4's gate: a charted region that CONTAINS a sphere
// pole must carry that pole once, with no zero-area triangle left behind. The rod ∪ ball sphere is that
// region — the ball minus the window the rod cuts in its side, so both poles are interior to it.
func TestThePoleRowCollapsesToOneVertex(t *testing.T) {
	t.Parallel()
	faces, meshes := tessellate.TessellateBodyFaces(rodBall(t, ops.Join), ops.DefaultQuality())
	found := false
	for i, f := range faces {
		sph, isSphere := f.Geometry().(geom.Sphere)
		if !isSphere {
			continue
		}
		found = true
		assertPoleIsOneVertex(t, meshes[i], sph)
		assertNoCollapsedTriangleSurvives(t, meshes[i])
	}
	if !found {
		t.Fatal("rod ∪ ball has no sphere face; the row no longer tests what it says")
	}
}

// assertPoleIsOneVertex checks that each pole of the sphere appears exactly once in the mesh — the
// covering's whole pole row welded down to the single point it is.
func assertPoleIsOneVertex(t *testing.T, m *tessellate.Mesh, sph geom.Sphere) {
	t.Helper()
	for _, pole := range []struct {
		name string
		v    float64
	}{{"north", stdmath.Pi / 2}, {"south", -stdmath.Pi / 2}} {
		at := sph.PointAt(0, pole.v)
		n := 0
		for _, p := range m.Positions {
			if float64(p.DistanceTo(at)) < geom.ResolutionForPoints(m.Positions).Weld() {
				n++
			}
		}
		if n != 1 {
			t.Errorf("the %s pole appears %d times in the sphere mesh, want the whole row welded to one vertex", pole.name, n)
		}
	}
}

// assertNoCollapsedTriangleSurvives checks that no triangle repeats a vertex and that the smallest is a
// real triangle rather than the zero-area remnant of a pole row (see chartPoleSliverFloor).
func assertNoCollapsedTriangleSurvives(t *testing.T, m *tessellate.Mesh) {
	t.Helper()
	minArea, total := stdmath.Inf(1), 0.0
	for ti, n := 0, m.TriangleCount(); ti < n; ti++ {
		a, b, c := tessellate.TriVerts(m, ti)
		if i, j, k := m.Indices[3*ti], m.Indices[3*ti+1], m.Indices[3*ti+2]; i == j || j == k || i == k {
			t.Fatalf("triangle %d repeats a vertex (%d,%d,%d): a collapsed pole triangle survived", ti, i, j, k)
		}
		area := float64(a.VectorTo(b).Cross(a.VectorTo(c)).Length()) / 2
		minArea, total = stdmath.Min(minArea, area), total+area
	}
	mean := total / float64(m.TriangleCount())
	if ratio := minArea / mean; ratio < chartPoleSliverFloor {
		t.Errorf("the smallest triangle is %.4f of the mean area (%.4e against %.4e); a pole row did not collapse",
			ratio, minArea, mean)
	}
}
