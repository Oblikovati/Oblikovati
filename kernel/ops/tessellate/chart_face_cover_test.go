// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// rimBoundedWindowedWall builds the shape a boolean leaves on a drilled wall, WITHOUT the artificial
// seam a primitive's own side carries: a cylinder side (radius wallR, z ∈ [0, wallH]) bounded by its two
// rim circles and carrying one rectangular window. Its chart is the parameter rectangle minus that
// window, which is what the boolean would have recorded (ADR-0063).
func rimBoundedWindowedWall(t *testing.T, uLo, uHi, vLo, vHi float64) *topo.Face {
	t.Helper()
	bottom, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), wallR)
	top := geom.Circle{Center: math.P3(0, 0, wallH), Normal: bottom.Normal, RefDir: bottom.RefDir, Radius: wallR}
	side, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), wallR)
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("chart", "body", 0)))
	vb := bld.AddVertex(bottom.PointAt(0), topo.NewLineage(topo.Tok("chart", "vb", 0)))
	vt := bld.AddVertex(top.PointAt(0), topo.NewLineage(topo.Tok("chart", "vt", 0)))
	eb := bld.AddEdge(bottom, vb, vb, topo.NewLineage(topo.Tok("chart", "eb", 0)))
	et := bld.AddEdge(top, vt, vt, topo.NewLineage(topo.Tok("chart", "et", 0)))
	bld.AddFace(side, topo.NewLineage(topo.Tok("chart", "face", 0)),
		topo.OuterLoop(topo.Fwd(eb)), topo.InnerLoop(topo.Rev(et)),
		windowHoleLoop(bld, uLo, uHi, vLo, vHi))
	f := bld.Build().Faces()[0]
	f.SetChart(wallChart(side, uLo, uHi, vLo, vHi))
	return f
}

// wallChart is the wall's parametric trim: the whole parameter rectangle, minus the window's own four
// corners in the cylinder's (u,v) — the window's edges are exact iso-lines, so four vertices are exact.
func wallChart(side geom.Cylinder, uLo, uHi, vLo, vHi float64) [][]math.Point2 {
	rect := []math.Point2{math.P2(0, 0), math.P2(2*stdmath.Pi, 0), math.P2(2*stdmath.Pi, wallH), math.P2(0, wallH)}
	corner := func(u, v float64) math.Point2 {
		cu, cv := side.ParamAt(wallPoint(u, v))
		return math.P2(cu, cv)
	}
	window := []math.Point2{corner(uLo, vLo), corner(uLo, vHi), corner(uHi, vHi), corner(uHi, vLo)}
	return [][]math.Point2{rect, window}
}

// TestAChartedWallIsMeshedOverItsOwnRegion is the mesher's own end-to-end row, off the router: the wall
// comes back watertight apart from its three boundary rings, and its area is the analytic wall minus the
// window — proof that the region came from the chart and the seam closed without being cut.
func TestAChartedWallIsMeshedOverItsOwnRegion(t *testing.T) {
	t.Parallel()
	const uLo, uHi, vLo, vHi = 3.0, 4.0, 3.0, 6.0
	f := rimBoundedWindowedWall(t, uLo, uHi, vLo, vHi)
	m, ok := chartFaceMesh(f, f.Geometry(), DefaultQuality())
	if !ok {
		t.Fatal("chartFaceMesh declined a charted, rim-bounded windowed wall")
	}
	want := 2*stdmath.Pi*wallR*wallH - (uHi-uLo)*wallR*(vHi-vLo)
	if got := m.Area(); stdmath.Abs(got-want)/want > 0.02 {
		t.Errorf("the charted wall meshes %.4f mm², want the analytic %.4f (wall minus window)", got, want)
	}
	// EXACTLY the rim: more means the mesh tore, fewer means it closed over its own boundary, which is
	// what a covering of the whole surface looks like.
	if free, rim := WeldedFreeEdgeCount(m), chainSegmentCount(chartBoundaryChains(f, f.Geometry(), mustRegion(t, f), DefaultQuality())); free != rim {
		t.Errorf("the charted wall has %d unpaired edges against a %d-segment rim", free, rim)
	}
}

// mustRegion reads a face's chart region or fails the test.
func mustRegion(t *testing.T, f *topo.Face) chartRegion {
	t.Helper()
	r, ok := newChartRegion(f, f.Geometry())
	if !ok {
		t.Fatal("newChartRegion declined a charted face")
	}
	return r
}

// TestChartFaceMeshDeclinesAFaceWithoutAChart: the mesher reads a record, it does not guess one. A face
// that carries none is exactly what still reaches the discarded-trim reporter.
func TestChartFaceMeshDeclinesAFaceWithoutAChart(t *testing.T) {
	t.Parallel()
	f := rimBoundedWindowedWall(t, 3.0, 4.0, 3.0, 6.0)
	f.SetChart(nil)
	if _, ok := chartFaceMesh(f, f.Geometry(), DefaultQuality()); ok {
		t.Error("chartFaceMesh took a face with no chart")
	}
}

// TestChartFaceMeshDeclinesAnAperiodicSurface: a plane has one branch and no seam, so ToUVLoops already
// charts its trim; there is nothing here for this mesher to settle.
func TestChartFaceMeshDeclinesAnAperiodicSurface(t *testing.T) {
	t.Parallel()
	f := chartedPlanarTriangle(t)
	if _, ok := chartFaceMesh(f, f.Geometry(), DefaultQuality()); ok {
		t.Error("chartFaceMesh took an aperiodic surface")
	}
}

// chartedPlanarTriangle builds a triangular face on the z=0 plane and gives it a chart, so the decline
// under test is the surface's lack of a seam and not the absence of a record.
func chartedPlanarTriangle(t *testing.T) *topo.Face {
	t.Helper()
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	lin := topo.NewLineage(topo.Tok("chart", "plane", 0))
	bld := topo.NewBuilder(false, lin)
	p := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)}
	v := []*topo.Vertex{bld.AddVertex(p[0], lin), bld.AddVertex(p[1], lin), bld.AddVertex(p[2], lin)}
	uses := make([]topo.Use, 3)
	for i := range p {
		j := (i + 1) % 3
		uses[i] = topo.Fwd(bld.AddEdge(geom.NewLineSegment(p[i], p[j]), v[i], v[j], lin))
	}
	bld.AddFace(pl, lin, topo.OuterLoop(uses...))
	f := bld.Build().Faces()[0]
	f.SetChart([][]math.Point2{{math.P2(0, 0), math.P2(1, 0), math.P2(0, 1)}})
	return f
}

// TestAFaceWithoutAChartStillReportsItsDiscardedTrim keeps the degradation signal meaningful now that
// every charted face is meshed from its chart: the reporter fires for a TRIMMED face and stays silent
// for an untrimmed one, which legitimately IS its whole domain.
func TestAFaceWithoutAChartStillReportsItsDiscardedTrim(t *testing.T) {
	t.Parallel()
	side, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), wallR)
	if got := recordIgnoredTrim(&Mesh{}, side, 2); !hasCode(got, CodeTrimIgnoredFullDomain) {
		t.Error("a trimmed face meshed over the whole domain reported nothing")
	}
	if got := recordIgnoredTrim(&Mesh{}, side, 0); hasCode(got, CodeTrimIgnoredFullDomain) {
		t.Error("an untrimmed face reported a discarded trim; the whole domain IS its region")
	}
}

// hasCode reports whether a mesh carries the given diagnostic code.
func hasCode(m *Mesh, code diag.Code) bool {
	for _, d := range m.Diagnostics {
		if d.Code == code {
			return true
		}
	}
	return false
}

// TestAStraightAxisStillGetsAnInteriorRow: a cylinder's height needs no chord subdivision, so the
// adaptive breakpoints are its two ends and the covering would have no interior row at all — the
// triangulation then reaches across the face for its diagonals and the volume integral collapses
// (measured: a windowed rod wall integrated 1.00 against 8.15).
func TestAStraightAxisStillGetsAnInteriorRow(t *testing.T) {
	t.Parallel()
	got := atLeastMinimumCells([]float64{0, 4}, 0, 4)
	if len(got) != minInteriorCells+1 {
		t.Fatalf("atLeastMinimumCells = %v, want %d stations", got, minInteriorCells+1)
	}
	if got[0] != 0 || got[len(got)-1] != 4 {
		t.Errorf("atLeastMinimumCells = %v, want the axis's own ends kept", got)
	}
	dense := []float64{0, 1, 2, 3, 4}
	if out := atLeastMinimumCells(dense, 0, 4); len(out) != len(dense) {
		t.Errorf("atLeastMinimumCells resampled an already-subdivided axis: %v", out)
	}
}

// TestInwardProbeOnlyMovesABoundedAxisEnd: a station on a wrapping axis has no end to sit on, and an
// interior station is not on the border either — only the first and last of a bounded axis are, and an
// even-odd count there answers by which side the ray came from (the sphere pole that read OUTSIDE).
func TestInwardProbeOnlyMovesABoundedAxisEnd(t *testing.T) {
	t.Parallel()
	stations := []float64{-1.5, -0.5, 0.5, 1.5}
	if got := inwardProbe(stations, 0, false); got != 0.5 {
		t.Errorf("inwardProbe at the first station = %g, want half the gap inward", got)
	}
	if got := inwardProbe(stations, 3, false); got != -0.5 {
		t.Errorf("inwardProbe at the last station = %g, want half the gap inward", got)
	}
	if got := inwardProbe(stations, 1, false); got != 0 {
		t.Errorf("inwardProbe at an interior station = %g, want none", got)
	}
	if got := inwardProbe(stations, 0, true); got != 0 {
		t.Errorf("inwardProbe on a wrapping axis = %g, want none", got)
	}
}

// TestATruncatedChainConstrainsOnlyItsWholeSegments: a replica whose points fall outside the covering's
// pad carries no segment there — its own replica one period along does.
func TestATruncatedChainConstrainsOnlyItsWholeSegments(t *testing.T) {
	t.Parallel()
	if got := chainConstraints([]int{-1, 4, 5, -1, 7}); len(got) != 1 || got[0][0] != 4 || got[0][1] != 5 {
		t.Errorf("chainConstraints = %v, want only the segment with both ends inside", got)
	}
}

// TestBoxIsNearRejectsWhatIsFarEnoughAway — the clearance query's cheap first pass.
func TestBoxIsNearRejectsWhatIsFarEnoughAway(t *testing.T) {
	t.Parallel()
	box := [4]float64{0, 1, 0, 1}
	if !boxIsNear(box, 1.4, 0.5, 0.5) {
		t.Error("boxIsNear rejected a point inside the grown box")
	}
	if boxIsNear(box, 1.6, 0.5, 0.5) {
		t.Error("boxIsNear accepted a point past the grown box")
	}
}

// TestThePadIsMeasuredOnTheCoarsestCell is the premise the canonical selection rests on: within the
// replication pad the covering repeats exactly, so the Delaunay triangulation is the same on both sides
// of the branch window and each seam-spanning triangle has exactly one translate whose centroid the
// window keeps. That holds only while the pad contains a triangle's circumcircle, and a triangle is as
// large as the covering's COARSEST cell — which on a wall with a straight axis is a row gap, not the
// column gap the pad used to be measured in. See chartCoverPadStations for the #1738 measurement.
func TestThePadIsMeasuredOnTheCoarsestCell(t *testing.T) {
	t.Parallel()
	columns, rows := []float64{0, 0.1, 0.2}, []float64{0, 5, 10} // the wall's own shape: fine u, three v rows
	cell := coarsestCoverCell(columns, rows, 3, 1)
	if want := stdmath.Hypot(0.1*3, 5.0); stdmath.Abs(cell-want) > 1e-12 { // tol:numeric
		t.Errorf("coarsestCoverCell = %g, want the ROW gap's diagonal %g, not the column gap's", cell, want)
	}
	if pad := chartPad(cell, 3, 2*stdmath.Pi); pad <= chartCoverPadStations*0.1 {
		t.Errorf("chartPad = %g: still measured in column gaps (%g), which is the #1738 defect", pad, 0.1)
	}
	if pad := chartPad(cell, 3, 0.5); pad != 0.5 {
		t.Errorf("chartPad = %g past a window only 0.5 wide, want it capped at the window", pad)
	}
}

// TestTheCoveringPadHoldsEveryTriangleItKeeps measures the pad against what it has to contain: no
// triangle the canonical window keeps may have a circumcircle wider than the replicated band, or the
// two sides of the window are deciding it against different neighbours.
func TestTheCoveringPadHoldsEveryTriangleItKeeps(t *testing.T) {
	t.Parallel()
	f := rimBoundedWindowedWall(t, 3.0, 4.0, 3.0, 6.0)
	s, q := f.Geometry(), DefaultQuality()
	r := mustRegion(t, f)
	chains := chartBoundaryChains(f, s, r, q)
	b := newChartCover(s, r, q)
	loops := b.addChains(chains)
	b.addInterior(chains)
	kept := b.keepChartTriangles(constrainedTriangulationAll(b.xy, loops))
	if len(kept) == 0 {
		t.Fatal("the charted wall kept no triangle")
	}
	worst := 0.0
	for _, tri := range kept {
		worst = stdmath.Max(worst, coverCircumradius(b, tri))
	}
	if pad := b.padU * b.su; worst > pad {
		t.Errorf("the widest kept triangle's circumcircle is %.4f across against a pad of %.4f — the "+
			"covering is not periodic out to it", worst, pad)
	}
}

// coverCircumradius is a covering triangle's circumradius in the metric-scaled (u,v).
func coverCircumradius(b *chartCover, tri [3]int) float64 {
	a, c, d := b.xy[tri[0]], b.xy[tri[1]], b.xy[tri[2]]
	ab := stdmath.Hypot(c[0]-a[0], c[1]-a[1])
	bc := stdmath.Hypot(d[0]-c[0], d[1]-c[1])
	ca := stdmath.Hypot(a[0]-d[0], a[1]-d[1])
	twiceArea := stdmath.Abs((c[0]-a[0])*(d[1]-a[1]) - (d[0]-a[0])*(c[1]-a[1]))
	if twiceArea == 0 {
		return 0 // a degenerate triangle the weld drops
	}
	return ab * bc * ca / (2 * twiceArea)
}

// TestAStraightAxisTakesItsCellFromTheOther: a floored axis (a cylinder's height, which has no chord to
// resolve) is refined until its cells are no longer than the other axis's; an axis the chord already
// subdivided is left exactly as it is. See balancedCoverGrid for the #1738 measurement.
func TestAStraightAxisTakesItsCellFromTheOther(t *testing.T) {
	t.Parallel()
	columns := []float64{0, 0.1, 0.2, 0.3} // four chord-chosen stations: more than the floor
	rows := []float64{0, 5, 10}            // the floor: a straight axis brings nothing else
	gotU, gotV := balancedCoverGrid(columns, rows, 3, 1)
	if len(gotU) != len(columns) {
		t.Errorf("the chord-subdivided axis was re-balanced: %d stations, want its own %d", len(gotU), len(columns))
	}
	const rowScale, columnScale = 1.0, 3.0 // the wall's own metric: v is the axis, u sweeps radius 3
	if widest := widestStationGap(gotV) * rowScale; widest > widestStationGap(columns)*columnScale {
		t.Errorf("the straight axis's widest cell is %.4f, want no more than the other's %.4f",
			widest, widestStationGap(columns)*columnScale)
	}
	if len(gotV) <= len(rows) {
		t.Errorf("the straight axis kept its %d floor stations; it must take the other's cell size", len(gotV))
	}
}

// TestBalancingNeverExceedsTheCellCap: the refinement is bounded by the package's own maximum cell
// count, so a chord far finer than a face is long cannot explode the covering.
func TestBalancingNeverExceedsTheCellCap(t *testing.T) {
	t.Parallel()
	_, rows := balancedCoverGrid([]float64{0, 1e-6, 2e-6}, []float64{0, 1000}, 1, 1)
	if cells := len(rows) - 1; cells > maxInteriorCells {
		t.Errorf("balancing produced %d cells, past the %d cap", cells, maxInteriorCells)
	}
	if len(rows) < 2 {
		t.Errorf("balancing dropped the axis entirely: %v", rows)
	}
}
