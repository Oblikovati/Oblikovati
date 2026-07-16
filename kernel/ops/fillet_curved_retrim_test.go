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

// TestRetrimCurvedHost_B3 drives the curved-host retrim on all four B3 hosts and asserts each
// retrimmed loop reproduces the oracle-closed area (§B) AND is genuinely correct — CLOSED (every arc
// edge joins the two loop points it spans, no chord gap) and carrying the certified circular RAILS
// (not straight chords): the wall/cap torus arcs and the bottom-cap foot arc. The bottom-cap 1932.47
// (≠ the naive quarter-disk 1963.50) proves the through-arm foot-bite is applied (§B.5).
func TestRetrimCurvedHost_B3(t *testing.T) {
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	w, ok := solveCurvedCorner(sphere, arms, res)
	if !ok {
		t.Fatalf("solveCurvedCorner rejected the certified B3 corner")
	}
	wall, topCap, radial, botCap := b3HostFaces(t)
	cy := b3CornerCY
	assertRetrim(t, wall, w, res, 5931.52, math.P3(0, 0, 90), 50)    // wall torus rail R=50 @ z=90
	assertRetrim(t, topCap, w, res, 860.844, math.P3(0, 0, 100), 40) // cap torus rail R=40 @ z=100
	assertRetrim(t, radial, w, res, 3485.69, math.Point3{}, 0)       // radial: two straight rulings, no arc
	assertRetrim(t, botCap, w, res, 1932.47, math.P3(10, cy, 0), 10) // bottom-cap foot arc R=10
	assertFootBiteApplied(t, botCap, w, res)
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

// assertFootBiteApplied is the mutation check: the retrimmed bottom cap must be the foot-bitten
// 1932.47, distinctly LESS than the naive quarter-disk 1963.50 — so a build that skipped the P1
// foot-bite (leaving the quarter disk) would fail this by ~31.
func assertFootBiteApplied(t *testing.T, botCap *topo.Face, w cornerWeld, res Resolution) {
	t.Helper()
	ff, ok := retrimCurvedHost(botCap, w, res)
	if !ok {
		t.Fatalf("retrimCurvedHost declined the bottom cap")
	}
	quarterDisk := stdmath.Pi * 50 * 50 / 4 // 1963.495 — the un-bitten cap
	a := developedLoopArea(botCap.Geometry(), ff.loops[0])
	if quarterDisk-a < 20 {
		t.Fatalf("bottom cap = %.4f, only %.4f below the un-bitten quarter disk %.4f — foot-bite not applied",
			a, quarterDisk-a, quarterDisk)
	}
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
