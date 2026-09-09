// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A NOTCHED cylinder band (ADR-0061 stage 5, Task 7). A wall whose second rim steps axially at ONE
// azimuth — the chord edges a D-prism seated on a cocylindrical wall leaves, and every mitre rail like
// them — is not a v(u) graph, so the ruled loft declines it. It used to land on the best-fit-plane CDT,
// which flattens a band that wraps the seam: 61 free edges on the merged cocylindrical wall. The chart
// the face carries says exactly which region it is, and the chart-driven mesher meshes it.

// notchStep* are the two azimuths the fixture's rim steps at, and notchLowV the axial station it steps
// down to from the wallH plateau.
const (
	notchStepLoAngle = 0.6
	notchStepHiAngle = 2.1
	notchLowV        = 9.0
)

// notchedRimWall builds a cylinder side (radius wallR, z ∈ [0, wallH]) bounded below by its bottom circle
// and above by a NOTCHED rim: the plateau at wallH everywhere but the sector between the two step angles,
// where it drops to notchLowV, joined by two axial runs at ONE azimuth each. It carries the chart that
// region determines (ADR-0063).
//
// The rim is wound CLOCKWISE, the way a hole loop of an outward-facing wall is.
func notchedRimWall(t *testing.T) *topo.Face {
	t.Helper()
	bottom, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), wallR)
	side, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), wallR)
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("notch", "body", 0)))
	vb := bld.AddVertex(bottom.PointAt(0), topo.NewLineage(topo.Tok("notch", "vb", 0)))
	eb := bld.AddEdge(bottom, vb, vb, topo.NewLineage(topo.Tok("notch", "eb", 0)))
	bld.AddFace(side, topo.NewLineage(topo.Tok("notch", "face", 0)),
		topo.OuterLoop(topo.Fwd(eb)), notchedRimLoop(bld, side))
	f := bld.Build().Faces()[0]
	f.SetChart(notchedRimChart())
	return f
}

// wallWorldAngle is the angle about +z at which the cylinder's own parameter u sits. A cylinder picks
// its own reference direction, so its u is NOT the world azimuth (here it trails it by π/2) — the arcs
// below are placed in world angles while the chart is written in the surface's, and this converts.
func wallWorldAngle(side geom.Cylinder, u float64) float64 {
	p := side.PointAt(u, 0)
	return stdmath.Atan2(float64(p.Y), float64(p.X))
}

// wallParamPoint is the wall point at the SURFACE's own (u, v).
func wallParamPoint(side geom.Cylinder, u, v float64) math.Point3 {
	return side.PointAt(u, v)
}

// notchedRimChart is the band's parametric trim: the whole period up to the plateau, bitten by the notch
// sector — one closed contour in the cylinder's own (u,v).
func notchedRimChart() [][]math.Point2 {
	return [][]math.Point2{{
		math.P2(0, 0), math.P2(2*stdmath.Pi, 0), math.P2(2*stdmath.Pi, wallH),
		math.P2(notchStepHiAngle, wallH), math.P2(notchStepHiAngle, notchLowV),
		math.P2(notchStepLoAngle, notchLowV), math.P2(notchStepLoAngle, wallH), math.P2(0, wallH),
	}}
}

// notchedRimLoop is that rim as a hole loop: plateau the long way, axial run, notch-floor arc, axial run.
func notchedRimLoop(bld *topo.Builder, side geom.Cylinder) topo.LoopSpec {
	z, x := math.V3(0, 0, 1), math.V3(1, 0, 0)
	vHiLo := bld.AddVertex(wallParamPoint(side, notchStepLoAngle, wallH), topo.NewLineage(topo.Tok("notch", "hiLo", 0)))
	vHiHi := bld.AddVertex(wallParamPoint(side, notchStepHiAngle, wallH), topo.NewLineage(topo.Tok("notch", "hiHi", 0)))
	vLoLo := bld.AddVertex(wallParamPoint(side, notchStepLoAngle, notchLowV), topo.NewLineage(topo.Tok("notch", "loLo", 0)))
	vLoHi := bld.AddVertex(wallParamPoint(side, notchStepHiAngle, notchLowV), topo.NewLineage(topo.Tok("notch", "loHi", 0)))
	plateau, _ := geom.NewArc3d(math.P3(0, 0, wallH), z, x, wallR, wallWorldAngle(side, notchStepHiAngle),
		notchStepLoAngle-notchStepHiAngle+2*stdmath.Pi) // the LONG way round, high step back to low
	floor, _ := geom.NewArc3d(math.P3(0, 0, notchLowV), z, x, wallR, wallWorldAngle(side, notchStepLoAngle),
		notchStepHiAngle-notchStepLoAngle)
	down := axialRun(bld, side, "down", vHiLo, vLoLo, notchStepLoAngle)
	eFloor := bld.AddEdge(floor, vLoLo, vLoHi, topo.NewLineage(topo.Tok("notch", "floor", 0)))
	up := axialRun(bld, side, "up", vLoHi, vHiHi, notchStepHiAngle)
	ePlateau := bld.AddEdge(plateau, vHiHi, vHiLo, topo.NewLineage(topo.Tok("notch", "plateau", 0)))
	// Head-to-tail ascending: down at the low step, the notch floor, up at the high step, the plateau
	// the long way back. Reversed, that is the descending rim a wall's hole loop is.
	ascending := append(append(append(down, eFloor), up...), ePlateau)
	return topo.InnerLoop(reversedUses(ascending)...)
}

// notchRunSteps is how many edges each axial run is built from. A single segment carries only its two
// endpoints and reads the same either way round; a real notched rim's run is a chord edge the boolean
// split at every vertex it met, so it carries interior stations. This reproduces that.
const notchRunSteps = 4

// axialRun builds one same-azimuth run as notchRunSteps collinear edges between two rim vertices.
func axialRun(bld *topo.Builder, side geom.Cylinder, name string, from, to *topo.Vertex, u float64) []*topo.Edge {
	lo, hi := float64(from.Point().Z), float64(to.Point().Z)
	prev, out := from, []*topo.Edge(nil)
	for i := 1; i <= notchRunSteps; i++ {
		next := to
		if i < notchRunSteps {
			next = bld.AddVertex(wallParamPoint(side, u, lo+(hi-lo)*float64(i)/notchRunSteps),
				topo.NewLineage(topo.Tok("notch", name+"v", i)))
		}
		out = append(out, bld.AddEdge(geom.NewLineSegment(prev.Point(), next.Point()), prev, next,
			topo.NewLineage(topo.Tok("notch", name, i))))
		prev = next
	}
	return out
}

// reversedUses walks a head-to-tail edge chain backwards, each use reversed — the descending rim.
func reversedUses(chain []*topo.Edge) []topo.Use {
	out := make([]topo.Use, 0, len(chain))
	for i := len(chain) - 1; i >= 0; i-- {
		out = append(out, topo.Rev(chain[i]))
	}
	return out
}

// notchedBandArea is the analytic band: the full lateral wall less the notch sector.
const notchedBandArea = 2*stdmath.Pi*wallR*wallH -
	(notchStepHiAngle-notchStepLoAngle)*wallR*(wallH-notchLowV)

// TestTheNotchedBandIsMeshedFromItsChart drives the fixture through the ROUTER, which is where the
// regression lives: no wrapping mesher reduces a rim that steps axially, and the exit for a singly
// periodic surface used to be the best-fit-plane CDT. The band must come back bounded by exactly its two
// rims and carrying the analytic area less a chord deficit.
func TestTheNotchedBandIsMeshedFromItsChart(t *testing.T) {
	t.Parallel()
	for _, q := range []Quality{DefaultQuality(), PropertyQuality()} {
		f := notchedRimWall(t)
		m := tessellateCurvedFace(f, q)
		want := len(FaceOuterBoundary(f, q))
		for _, h := range faceHoleBoundaries(f, q) {
			want += len(h)
		}
		if free := WeldedFreeEdgeCount(m); free != want {
			t.Errorf("tol %g: the notched band has %d unpaired edges against its %d rim segments",
				q.ChordTolerance, free, want)
		}
		if rel := (m.Area() - notchedBandArea) / notchedBandArea; rel > 0 || rel < -0.01 {
			t.Errorf("tol %g: the notched band meshes %.5f mm², want %.5f less a chord deficit (rel %+.4f)",
				q.ChordTolerance, m.Area(), notchedBandArea, rel)
		}
	}
}

// TestTheNotchedBandDeclinesTheRuledLoft keeps the row above from passing for the wrong reason: the
// fixture must actually reach the general path, which it does because its rim is not a v(u) graph.
func TestTheNotchedBandDeclinesTheRuledLoft(t *testing.T) {
	t.Parallel()
	f := notchedRimWall(t)
	if _, ok := saddleBandLoftMesh(f, f.Geometry(), DefaultQuality()); ok {
		t.Error("the ruled loft claimed a rim that steps axially — the fixture no longer exercises the general path")
	}
}
