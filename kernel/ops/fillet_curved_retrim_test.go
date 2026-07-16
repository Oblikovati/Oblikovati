// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// b3HostFaces builds the four original B3 host faces the retrim consumes, each with its full wedge
// loop (arc rims where the boundary is a circle) so retrimCurvedHost reads real edges: the cylinder
// wall R=50, the top cap z=100, the radial plane x=0, and the bottom cap z=0.
func b3HostFaces(t *testing.T) (wall, topCap, radial, botCap *topo.Face) {
	t.Helper()
	bld := topo.NewBuilder(true, topo.Lineage{})
	wall = buildWallFace(t, bld)
	topCap = buildCapFace(t, bld, 100)
	radial = buildRadialFace(t, bld)
	botCap = buildCapFace(t, bld, 0)
	return wall, topCap, radial, botCap
}

// buildWallFace builds the quarter-cylinder wall: bottom rim arc, W∧N vertical edge, top rim arc,
// y=0 vertical edge.
func buildWallFace(t *testing.T, bld *topo.Builder) *topo.Face {
	t.Helper()
	lin := topo.Lineage{}
	v := func(p math.Point3) *topo.Vertex { return bld.AddVertex(p, lin) }
	a0, a1, a2, a3 := v(math.P3(50, 0, 0)), v(math.P3(0, -50, 0)), v(math.P3(0, -50, 100)), v(math.P3(50, 0, 100))
	bot := bld.AddEdge(mustArc(t, math.P3(0, 0, 0), 50, 0, -stdmath.Pi/2), a0, a1, lin)
	wn := bld.AddEdge(geom.NewLineSegment(a1.Point(), a2.Point()), a1, a2, lin)
	top := bld.AddEdge(mustArc(t, math.P3(0, 0, 100), 50, -stdmath.Pi/2, stdmath.Pi/2), a2, a3, lin)
	y0 := bld.AddEdge(geom.NewLineSegment(a3.Point(), a0.Point()), a3, a0, lin)
	cyl := mustCylinder(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 50)
	return bld.AddFace(cyl, lin, topo.OuterLoop(topo.Fwd(bot), topo.Fwd(wn), topo.Fwd(top), topo.Fwd(y0)))
}

// buildCapFace builds a quarter-disk cap at height z (top z=100 or bottom z=0): y=0 edge, outer rim
// arc R=50, x=0 edge.
func buildCapFace(t *testing.T, bld *topo.Builder, z float64) *topo.Face {
	t.Helper()
	lin := topo.Lineage{}
	v := func(p math.Point3) *topo.Vertex { return bld.AddVertex(p, lin) }
	o, xr, yr := v(math.P3(0, 0, z)), v(math.P3(50, 0, z)), v(math.P3(0, -50, z))
	e1 := bld.AddEdge(geom.NewLineSegment(o.Point(), xr.Point()), o, xr, lin)
	e2 := bld.AddEdge(mustArc(t, math.P3(0, 0, z), 50, 0, -stdmath.Pi/2), xr, yr, lin)
	e3 := bld.AddEdge(geom.NewLineSegment(yr.Point(), o.Point()), yr, o, lin)
	pl := mustPlane(t, math.P3(0, 0, z), math.V3(0, 0, 1))
	return bld.AddFace(pl, lin, topo.OuterLoop(topo.Fwd(e1), topo.Fwd(e2), topo.Fwd(e3)))
}

// buildRadialFace builds the x=0 radial rectangle y∈[−50,0]×z∈[0,100] (all straight edges).
func buildRadialFace(t *testing.T, bld *topo.Builder) *topo.Face {
	t.Helper()
	lin := topo.Lineage{}
	v := func(p math.Point3) *topo.Vertex { return bld.AddVertex(p, lin) }
	c0, c1, c2, c3 := v(math.P3(0, 0, 0)), v(math.P3(0, -50, 0)), v(math.P3(0, -50, 100)), v(math.P3(0, 0, 100))
	seg := func(a, b *topo.Vertex) *topo.Edge {
		return bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, lin)
	}
	e0, e1, e2, e3 := seg(c0, c1), seg(c1, c2), seg(c2, c3), seg(c3, c0)
	pl := mustPlane(t, math.P3(0, 0, 0), math.V3(1, 0, 0))
	return bld.AddFace(pl, lin, topo.OuterLoop(topo.Fwd(e0), topo.Fwd(e1), topo.Fwd(e2), topo.Fwd(e3)))
}

// mustArc builds a z-constant Arc3d centred on the axis (normal ẑ, ref x̂) or fails the test.
func mustArc(t *testing.T, center math.Point3, radius, start, sweep float64) geom.Arc3d {
	t.Helper()
	arc, err := geom.NewArc3d(center, math.V3(0, 0, 1), math.V3(1, 0, 0), radius, start, sweep)
	if err != nil {
		t.Fatalf("build arc: %v", err)
	}
	return arc
}

// buildRadialFaceWithHole builds the same x=0 radial rectangle as buildRadialFace, but with a small
// square INNER (hole) loop added FIRST — at Loops()[0] — and the outer boundary loop added second.
// This ordering is deliberate: a retrim that (bug) reads Loops()[0] unconditionally would try to bite
// the hole instead of the rectangle's boundary, so this fixture makes that regression fail loudly
// rather than silently (Important review finding, T5.3). Returns the hole's loop points in traversal
// order, for the caller to check they survive the retrim unchanged.
func buildRadialFaceWithHole(t *testing.T) (*topo.Face, []math.Point3) {
	t.Helper()
	bld := topo.NewBuilder(true, topo.Lineage{})
	lin := topo.Lineage{}
	v := func(p math.Point3) *topo.Vertex { return bld.AddVertex(p, lin) }
	seg := func(a, b *topo.Vertex) *topo.Edge {
		return bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, lin)
	}
	// outer boundary: identical rectangle to buildRadialFace (y∈[−50,0]×z∈[0,100]).
	c0, c1, c2, c3 := v(math.P3(0, 0, 0)), v(math.P3(0, -50, 0)), v(math.P3(0, -50, 100)), v(math.P3(0, 0, 100))
	oe0, oe1, oe2, oe3 := seg(c0, c1), seg(c1, c2), seg(c2, c3), seg(c3, c0)
	// inner hole: a small 2×2 square well clear of the bitten corner near (0,−50,100) and of the
	// straight-ruling exits, so a CORRECT retrim (reading only the outer loop) is unaffected by it.
	h0, h1, h2, h3 := v(math.P3(0, -24, 49)), v(math.P3(0, -26, 49)), v(math.P3(0, -26, 51)), v(math.P3(0, -24, 51))
	he0, he1, he2, he3 := seg(h0, h1), seg(h1, h2), seg(h2, h3), seg(h3, h0)
	pl := mustPlane(t, math.P3(0, 0, 0), math.V3(1, 0, 0))
	face := bld.AddFace(pl, lin,
		topo.InnerLoop(topo.Fwd(he0), topo.Fwd(he1), topo.Fwd(he2), topo.Fwd(he3)),
		topo.OuterLoop(topo.Fwd(oe0), topo.Fwd(oe1), topo.Fwd(oe2), topo.Fwd(oe3)),
	)
	return face, []math.Point3{h0.Point(), h1.Point(), h2.Point(), h3.Point()}
}

// TestRetrimCurvedHost_InnerLoopSurvives is the regression for the Important review finding: the
// retrim must select the host's OUTER loop by topo.Loop.IsOuter(), not Loops()[0] — a face's inner
// (hole) loops can sit at any index — and must carry any inner loop through into the result UNCHANGED.
// buildRadialFaceWithHole plants the inner loop at index 0 precisely so a Loops()[0]-reading retrim
// bites the hole (wrong shape / wrong area / decline) instead of the rectangle's boundary.
func TestRetrimCurvedHost_InnerLoopSurvives(t *testing.T) {
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	w, ok := solveCurvedCorner(sphere, arms, res)
	if !ok {
		t.Fatalf("solveCurvedCorner rejected the certified B3 corner")
	}
	host, holePts := buildRadialFaceWithHole(t)
	ff, ok := retrimCurvedHost(host, w, res)
	if !ok {
		t.Fatalf("retrimCurvedHost declined the holed radial host")
	}
	if len(ff.loops) != 2 {
		t.Fatalf("retrimmed face has %d loops, want 2 (retrimmed outer + unchanged inner hole)", len(ff.loops))
	}
	assertNoArcs(t, ff.loops[0]) // outer retrim: same straight-rulings shape as the un-holed radial (§B)
	if a := developedLoopArea(host.Geometry(), ff.loops[0]); stdmath.Abs(a-3485.69) > areaTol(3485.69) {
		t.Fatalf("outer retrim area = %.4f, want 3485.69 (the hole must not perturb the boundary bite)", a)
	}
	assertLoopPointsEqual(t, ff.loops[1], holePts)
}

// assertLoopPointsEqual checks a filletLoop's points match want exactly, in order — proof an inner
// (hole) loop passed through the retrim byte-for-byte unchanged.
func assertLoopPointsEqual(t *testing.T, loop filletLoop, want []math.Point3) {
	t.Helper()
	if len(loop.pts) != len(want) {
		t.Fatalf("inner loop has %d points, want %d (%v)", len(loop.pts), len(want), want)
	}
	for i, p := range want {
		if float64(p.DistanceTo(loop.pts[i])) > 1e-9 {
			t.Fatalf("inner loop point %d = %v, want %v (unchanged)", i, loop.pts[i], p)
		}
	}
}

// TestRetrimCurvedHost_B3 drives the curved-host retrim on the three CORNER hosts and asserts each
// retrimmed loop reproduces the oracle-closed area (§B) AND is genuinely correct — CLOSED (every arc
// edge joins the two loop points it spans, no chord gap) and carrying the certified circular RAILS
// (not straight chords): the wall/cap torus arcs. The bottom cap is NOT a corner host — its far-runout
// cross-section bite is produced by spliceCornerBite (fillet_curved_farrunout.go), covered end-to-end
// by the B3 weld volume regression, not by retrimCurvedHost.
func TestRetrimCurvedHost_B3(t *testing.T) {
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	w, ok := solveCurvedCorner(sphere, arms, res)
	if !ok {
		t.Fatalf("solveCurvedCorner rejected the certified B3 corner")
	}
	wall, topCap, radial, _ := b3HostFaces(t)
	assertRetrim(t, wall, w, res, 5931.52, math.P3(0, 0, 90), 50)    // wall torus rail R=50 @ z=90
	assertRetrim(t, topCap, w, res, 860.844, math.P3(0, 0, 100), 40) // cap torus rail R=40 @ z=100
	assertRetrim(t, radial, w, res, 3485.69, math.Point3{}, 0)       // radial: two straight rulings, no arc
}

// assertRetrim retrims one host, then checks its loop is closed, has the certified circular rail (a
// zero wantRadius means the loop must be all straight), and reproduces the oracle developed area.
func assertRetrim(t *testing.T, host *topo.Face, w cornerWeld, res Resolution, wantArea float64, arcCenter math.Point3, arcRadius float64) {
	t.Helper()
	ff, ok := retrimCurvedHost(host, w, res)
	if !ok {
		t.Fatalf("retrimCurvedHost declined host %T", host.Geometry())
	}
	loop := ff.loops[0]
	assertLoopClosed(t, loop)
	if arcRadius == 0 {
		assertNoArcs(t, loop)
	} else {
		assertArcRail(t, loop, arcCenter, arcRadius)
	}
	if a := developedLoopArea(host.Geometry(), loop); stdmath.Abs(a-wantArea) > areaTol(wantArea) {
		t.Fatalf("%T retrimmed area = %.4f, want %.4f", host.Geometry(), a, wantArea)
	}
}

// areaTol is the per-loop area tolerance: 0.1 for the exact-corner planar loops, a touch looser for
// the developed cylinder wall whose curved rails are sampled.
func areaTol(want float64) float64 {
	if want > 3000 {
		return 0.6
	}
	return 0.1
}

// assertLoopClosed checks the loop is a single closed ring: every arc edge's endpoints are exactly
// the two consecutive loop points it spans (so no arc silently degenerates to a chord or leaves a gap).
func assertLoopClosed(t *testing.T, loop filletLoop) {
	t.Helper()
	n := len(loop.pts)
	if n < 3 {
		t.Fatalf("retrim loop has %d points (want ≥3)", n)
	}
	for i := 0; i < n; i++ {
		arc, ok := loop.curves[i].(geom.Arc3d)
		if !ok {
			continue
		}
		a, b := loop.pts[i], loop.pts[(i+1)%n]
		got := arc.PointAt(0).DistanceTo(a) + arc.PointAt(1).DistanceTo(b)
		rev := arc.PointAt(0).DistanceTo(b) + arc.PointAt(1).DistanceTo(a)
		if stdmath.Min(float64(got), float64(rev)) > 1e-4 {
			t.Fatalf("arc edge %d does not join its loop points %v→%v", i, a, b)
		}
	}
}

// assertNoArcs asserts the loop is a straight-edged polygon (the radial rectangle).
func assertNoArcs(t *testing.T, loop filletLoop) {
	t.Helper()
	for i, c := range loop.curves {
		if _, ok := c.(geom.Arc3d); ok {
			t.Fatalf("retrim loop edge %d is an arc, want all straight (radial rectangle)", i)
		}
	}
}

// assertArcRail asserts the loop carries a circular rail on the certified circle (centre+radius) —
// proof the contact rail is an exact arc, not a straight chord.
func assertArcRail(t *testing.T, loop filletLoop, center math.Point3, radius float64) {
	t.Helper()
	for _, c := range loop.curves {
		if arc, ok := c.(geom.Arc3d); ok {
			if float64(arc.Center.DistanceTo(center)) < 1e-4 && stdmath.Abs(arc.Radius-radius) < 1e-4 {
				return
			}
		}
	}
	t.Fatalf("retrim loop carries no arc on circle centre %v radius %.1f (rail collapsed to a chord?)", center, radius)
}

// squareRim returns a square's four corners (side `size`, lower-left corner at (origin,origin,0) in
// the z=0 plane) — a stand-in outer rim for the bittenLoop fakes below. bittenLoop reads only loop
// topology (Loops/EdgeUses/vertex points), never the host surface, so the fake face's actual plane is
// inert for these tests; it exists only so topo.Builder.AddFace has a valid geom.Surface.
func squareRim(origin, size float64) []math.Point3 {
	lo, hi := math.Scalar(origin), math.Scalar(origin+size)
	return []math.Point3{
		math.P3(lo, lo, 0),
		math.P3(hi, lo, 0),
		math.P3(hi, hi, 0),
		math.P3(lo, hi, 0),
	}
}

// notchWindow returns a small triangle whose FIRST vertex sits exactly at (x,y,z) — the fake inner
// notch loop's nearest-to-C vertex, standing in for N7's boolean-cut window whose corner-side vertex
// the trihedral corner actually bites.
func notchWindow(x, y, z float64) []math.Point3 {
	return []math.Point3{
		math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(z)),
		math.P3(math.Scalar(x+2), math.Scalar(y), math.Scalar(z)),
		math.P3(math.Scalar(x), math.Scalar(y+2), math.Scalar(z)),
	}
}

// ringLoop wires pts into a closed straight-edge topo.LoopSpec (outer or inner) on bld — the shared
// plumbing behind every bittenLoop fake host below.
func ringLoop(bld *topo.Builder, pts []math.Point3, outer bool) topo.LoopSpec {
	lin := topo.Lineage{}
	verts := make([]*topo.Vertex, len(pts))
	for i, p := range pts {
		verts[i] = bld.AddVertex(p, lin)
	}
	uses := make([]topo.Use, len(pts))
	for i := range pts {
		a, b := verts[i], verts[(i+1)%len(pts)]
		uses[i] = topo.Fwd(bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, lin))
	}
	if outer {
		return topo.OuterLoop(uses...)
	}
	return topo.InnerLoop(uses...)
}

// newTwoLoopFace builds a fake host with an OUTER loop and one INNER loop, both straight-edged — the
// N7-shaped fake: a wall's outer rim plus a boolean-cut inner notch window.
func newTwoLoopFace(t *testing.T, outerPts, innerPts []math.Point3) *topo.Face {
	t.Helper()
	bld := topo.NewBuilder(true, topo.Lineage{})
	pl := mustPlane(t, math.P3(0, 0, 0), math.V3(0, 0, 1))
	return bld.AddFace(pl, topo.Lineage{}, ringLoop(bld, innerPts, false), ringLoop(bld, outerPts, true))
}

// newSingleLoopFace builds a fake host with a single OUTER loop — the B3 shape bittenLoop must reduce
// to trivially (one loop, so it is always "the bitten loop" regardless of C).
func newSingleLoopFace(t *testing.T, pts []math.Point3) *topo.Face {
	t.Helper()
	bld := topo.NewBuilder(true, topo.Lineage{})
	pl := mustPlane(t, math.P3(0, 0, 0), math.V3(0, 0, 1))
	return bld.AddFace(pl, topo.Lineage{}, ringLoop(bld, pts, true))
}

// newTieLoopFace builds a fake host with two loops whose nearest vertices to C=(0,0,0) sit 0.01 apart
// — inside the 0.02 tol TestBittenLoop_TieRejects calls bittenLoop with — so neither loop is
// unambiguously "the" bitten one (do-no-harm: bittenLoop must decline, not guess which wire the
// trihedral corner actually bit).
func newTieLoopFace(t *testing.T) *topo.Face {
	t.Helper()
	a := []math.Point3{math.P3(1, 0, 0), math.P3(10, 10, 10), math.P3(10, -10, 10)}
	b := []math.Point3{math.P3(1.01, 0, 0), math.P3(10, 10, -10), math.P3(10, -10, -10)}
	bld := topo.NewBuilder(true, topo.Lineage{})
	pl := mustPlane(t, math.P3(0, 0, 0), math.V3(0, 0, 1))
	return bld.AddFace(pl, topo.Lineage{}, ringLoop(bld, a, false), ringLoop(bld, b, true))
}

// TestBittenLoop_SelectsInnerNotchLoop is the N7 shape: the corner-sphere centre lands nearest a
// vertex on the INNER notch loop (min dist 0, exact), far from the outer rim (min dist ≈51) — the
// bitten loop must be the inner one, not the outer rim T5.3's original code assumed.
func TestBittenLoop_SelectsInnerNotchLoop(t *testing.T) {
	c := math.P3(50, 0, 10) // corner-sphere centre near the inner loop
	host := newTwoLoopFace(t,
		squareRim(0, 100),      // outer rim vertices, min dist to c ≈ 51
		notchWindow(50, 0, 10)) // inner notch vertices, nearest vertex = (50,0,10) = c exactly
	tol := 0.02 // res.Weld()*r for r=5

	l, ok := bittenLoop(host, c, tol)

	if !ok {
		t.Fatalf("bittenLoop: expected the inner notch loop, got ok=false")
	}
	if l.IsOuter() {
		t.Fatalf("bittenLoop selected the OUTER rim; want the inner notch loop (nearest to C=%v)", c)
	}
}

// TestBittenLoop_SingleLoopReducesToOuter is the B3 reduction gate (R.0): every B3 corner host is
// single-loop, so bittenLoop must return that (outer) loop regardless of C — the generalized selector
// must not change behaviour on the clean wedge.
func TestBittenLoop_SingleLoopReducesToOuter(t *testing.T) {
	c := math.P3(10, -38.7298, 90) // B3 corner centre
	host := newSingleLoopFace(t, squareRim(0, 100))

	l, ok := bittenLoop(host, c, 0.02)

	if !ok || !l.IsOuter() {
		t.Fatalf("bittenLoop on a single-loop host must return that (outer) loop; ok=%v outer=%v", ok, l.IsOuter())
	}
}

// TestBittenLoop_TieRejects is the do-no-harm ambiguity gate: two loops equidistant to C within tol
// means the retrim cannot tell which wire the corner actually bit, so bittenLoop must decline rather
// than guess (a wrong pick would bite the wrong loop and corrupt the mesh silently).
func TestBittenLoop_TieRejects(t *testing.T) {
	c := math.P3(0, 0, 0)
	host := newTieLoopFace(t)

	if _, ok := bittenLoop(host, c, 0.02); ok {
		t.Fatalf("bittenLoop must reject an ambiguous two-loop tie (do-no-harm), got ok=true")
	}
}

// TestSegsFromLoop_MatchesLoopEdgeOrder proves segsFromLoop reads one loop's edge uses, in traversal
// order, as endSegs — the primitive that lets retrimCurvedHost retrim whichever loop bittenLoop picked
// (outer or inner), generalizing the old "always read the outer loop" originalHostSegs.
func TestSegsFromLoop_MatchesLoopEdgeOrder(t *testing.T) {
	pts := squareRim(0, 10)
	host := newSingleLoopFace(t, pts)
	loop := host.Loops()[0]

	segs := segsFromLoop(loop)

	if len(segs) != len(pts) {
		t.Fatalf("segsFromLoop returned %d segs, want %d (one per loop edge)", len(segs), len(pts))
	}
	for i, p := range pts {
		if float64(segs[i].from.DistanceTo(p)) > 1e-9 {
			t.Fatalf("seg %d starts at %v, want loop vertex %v", i, segs[i].from, p)
		}
	}
}

// TestLoopsExcept_CarriesEveryOtherLoopVerbatim proves loopsExcept keeps every loop but the excluded
// one — on the N7 wall that is the outer rim, once bittenLoop has picked the inner notch as L*,
// generalizing innerHostLoops' "carry every hole" to "carry every OTHER loop".
func TestLoopsExcept_CarriesEveryOtherLoopVerbatim(t *testing.T) {
	outer, inner := squareRim(0, 100), notchWindow(50, 0, 10)
	host := newTwoLoopFace(t, outer, inner)
	var bitten *topo.Loop
	for _, l := range host.Loops() {
		if !l.IsOuter() {
			bitten = l
		}
	}

	kept := loopsExcept(host, bitten)

	if len(kept) != 1 {
		t.Fatalf("loopsExcept kept %d loops, want 1 (the outer rim, excluding the bitten inner loop)", len(kept))
	}
	assertLoopPointsEqual(t, kept[0], outer)
}

// developedLoopArea is the true surface area a retrimmed loop bounds, measured in the host's isometric
// development (plane u,v; cylinder unrolled to R·θ × axial): each arc edge is densely sampled so it
// contributes its exact segment area, then a shoelace over the chart gives the area. This is the
// loop-area helper the T5.3 brief asks for — exact on the plane, isometry-exact on the cylinder.
func developedLoopArea(host geom.Surface, loop filletLoop) float64 {
	poly := densePolyline(loop, 128)
	return stdmath.Abs(shoelaceDev(host, poly))
}

// densePolyline expands the loop into a fine 3D point ring, sampling each arc edge with k chords.
func densePolyline(loop filletLoop, k int) []math.Point3 {
	n := len(loop.pts)
	var out []math.Point3
	for i := 0; i < n; i++ {
		a, b := loop.pts[i], loop.pts[(i+1)%n]
		out = append(out, a)
		if arc, ok := loop.curves[i].(geom.Arc3d); ok {
			out = append(out, arcInterior(arc, a, b, k)...)
		}
	}
	return out
}

// arcInterior samples an arc edge's interior points, oriented from a toward b.
func arcInterior(arc geom.Arc3d, a, b math.Point3, k int) []math.Point3 {
	fwd := arc.PointAt(0).DistanceTo(a) <= arc.PointAt(1).DistanceTo(a)
	out := make([]math.Point3, 0, k-1)
	for j := 1; j < k; j++ {
		t := float64(j) / float64(k)
		if !fwd {
			t = 1 - t
		}
		out = append(out, arc.PointAt(t))
	}
	return out
}

// shoelaceDev shoelaces the polyline in the host's development chart.
func shoelaceDev(host geom.Surface, poly []math.Point3) float64 {
	d := devPoints(host, poly)
	area := 0.0
	for i := range d {
		p, q := d[i], d[(i+1)%len(d)]
		area += float64(p.X*q.Y - q.X*p.Y)
	}
	return area / 2
}

// devPoints maps the polyline to isometric 2D chart coordinates (identity uv on a plane; unrolled
// R·θ × axial on a cylinder, with the azimuth unwrapped so the developed loop stays continuous).
func devPoints(host geom.Surface, poly []math.Point3) []math.Point2 {
	switch h := host.(type) {
	case geom.Plane:
		out := make([]math.Point2, len(poly))
		for i, p := range poly {
			w := h.Origin.VectorTo(p)
			out[i] = math.P2(w.Dot(h.UAxis.AsVector()), w.Dot(h.VAxis.AsVector()))
		}
		return out
	case geom.Cylinder:
		return cylinderDevPoints(h, poly)
	}
	return nil
}

// cylinderDevPoints unrolls the polyline onto the cylinder's development, unwrapping the azimuth.
func cylinderDevPoints(cyl geom.Cylinder, poly []math.Point3) []math.Point2 {
	axis, ref := cyl.AxisDir.AsVector(), cyl.Ref.AsVector()
	bin := axis.Cross(ref)
	out := make([]math.Point2, len(poly))
	prev := 0.0
	for i, p := range poly {
		w := cyl.Origin.VectorTo(p)
		th := stdmath.Atan2(float64(w.Dot(bin)), float64(w.Dot(ref)))
		if i > 0 {
			th = unwrapAngle(prev, th)
		}
		prev = th
		out[i] = math.P2(math.Scalar(cyl.Radius*th), w.Dot(axis))
	}
	return out
}

// unwrapAngle shifts th by multiples of 2π to sit within π of prev (continuous development).
func unwrapAngle(prev, th float64) float64 {
	for th-prev > stdmath.Pi {
		th -= 2 * stdmath.Pi
	}
	for th-prev < -stdmath.Pi {
		th += 2 * stdmath.Pi
	}
	return th
}
