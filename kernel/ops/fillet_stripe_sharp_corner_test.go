// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
)

// Regression for Oblikovati#2083. An open tangent run that ends at a SHARP CORNER reaches the end of
// its rim, so the run-out section plane is the plane of the side face already sitting there. The flat
// setback cap built for it therefore landed ON that face — reversed and coincident — and the two
// corner connectors doubled back along boundaries they were collinear with. Every edge was still used
// twice and the flaps cancelled in the volume integral, so both the topology gate and the OCCT volume
// oracle passed a body with six degenerate artifacts in it.

// sharpCornerRunOut is the #2083 fixture: a 4 cm box with ONE vertical edge rounded at 0.5, whose top
// rim is then an OPEN straight–arc–straight run, filleted at 0.25. Both terminals are sharp corners.
func sharpCornerRunOut(t *testing.T) *topo.Body {
	t.Helper()
	box := csgBox(gmath.P3(0, 0, 0), 4, 4, 4)
	var one [][]byte
	for _, e := range box.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			one = append(one, e.ReferenceKey())
			break
		}
	}
	rounded, err := ops.FilletEdges(box, one, 0.5)
	if err != nil {
		t.Fatalf("single vertical fillet: %v", err)
	}
	chain, closed, err := ops.TangentEdgeChain(rounded, firstStraightTopEdge(t, rounded), ops.DefaultTangentChainAngle)
	if err != nil {
		t.Fatal(err)
	}
	if closed || len(chain) != 3 {
		t.Fatalf("expected an OPEN 3-edge chain, got closed=%v len=%d", closed, len(chain))
	}
	res, err := ops.FilletEdges(rounded, chain, 0.25)
	if err != nil {
		t.Fatalf("sharp-corner run-out fillet: %v", err)
	}
	return res
}

// TestSharpCornerRunOutHasNoCoincidentCap is the headline: no face of the result may pass through
// another. Before the fix there were two such pairs, each a reversed cap lying on an untrimmed side
// face and double-covering 0.0134 cm2. Both faces are PLANAR, so the #2077 faceting allowance is
// zero and this cannot be a meshing artifact.
func TestSharpCornerRunOutHasNoCoincidentCap(t *testing.T) {
	t.Parallel()
	res := sharpCornerRunOut(t)
	if hits := ops.SelfIntersections(res, ops.DefaultQuality()); len(hits) > 0 {
		t.Errorf("the run-out result has %d interpenetrating face pairs, first at %v",
			len(hits), hits[0].Witness)
	}
}

// TestSharpCornerRunOutBuildsNoCapFace: the side face carries the run-out itself, so the two flat
// caps must not be built at all. Their absence is what removes the coincidence — trimming the side
// face while KEEPING the cap would leave the region uncovered instead of double-covered.
func TestSharpCornerRunOutBuildsNoCapFace(t *testing.T) {
	t.Parallel()
	res := sharpCornerRunOut(t)
	// A cap would be a small planar face wholly inside the r=0.25 corner box at a terminal.
	for _, f := range res.Faces() {
		if _, planar := f.Geometry().(geom.Plane); !planar {
			continue
		}
		bb := f.RangeBox()
		if span := bb.Min.VectorTo(bb.Max).Length(); float64(span) < 1 {
			t.Errorf("a planar face only %.3f across survives at %v — that is a run-out cap", span, bb)
		}
	}
}

// TestSharpCornerRunOutConsumesTheCornerVertex: the corner is inside the material the rolling ball
// took away, so it must not remain a vertex of the body. Leaving it is what forced the connectors,
// and the connectors are what produced the collinear zero-area spikes in the top face and the wall.
func TestSharpCornerRunOutConsumesTheCornerVertex(t *testing.T) {
	t.Parallel()
	res := sharpCornerRunOut(t)
	for _, v := range res.Vertices() {
		if float64(v.Point().DistanceTo(gmath.P3(4, 4, 4))) < 1e-9 {
			t.Fatal("the terminal corner (4,4,4) is still a vertex — it was removed by the fillet")
		}
	}
}

// TestSharpCornerRunOutTrimsTheSideFace: the side face's boundary must now turn along the section
// arc, running foot → arc → foot instead of out to the corner and back.
func TestSharpCornerRunOutTrimsTheSideFace(t *testing.T) {
	t.Parallel()
	res := sharpCornerRunOut(t)
	side := sideFaceAtY4(t, res)
	feet := []gmath.Point3{gmath.P3(3.75, 4, 4), gmath.P3(4, 4, 3.75)}
	for _, want := range feet {
		if !loopTouches(side, want) {
			t.Errorf("the y=4 face's boundary misses the section foot %v", want)
		}
	}
	if loopTouches(side, gmath.P3(4, 4, 4)) {
		t.Error("the y=4 face's boundary still reaches the removed corner (4,4,4)")
	}
}

// TestSharpCornerRunOutKeepsTheRemovedVolume guards the point set the rebuild must NOT change: the
// old flaps cancelled exactly, so a correct rebuild has the same volume — and it must still match the
// OCCT oracle. A trim that removed the wrong side of the arc would show up here at once.
func TestSharpCornerRunOutKeepsTheRemovedVolume(t *testing.T) {
	t.Parallel()
	res := sharpCornerRunOut(t)
	// Points either side of the rolling ball's last position, at the run-out and deep in the run.
	for _, c := range []struct {
		p    gmath.Point3
		want bool
	}{
		{gmath.P3(3.8, 3.99, 3.8), true},    // inside the ball: kept, this is a convex-edge round
		{gmath.P3(3.98, 3.99, 3.98), false}, // the corner tip: taken away
		{gmath.P3(3.8, 3.5, 3.8), true},
		{gmath.P3(3.98, 3.5, 3.98), false},
	} {
		if got := ops.PointInsideBody(res, c.p); got != c.want {
			t.Errorf("PointInsideBody(%v) = %v, want %v", c.p, got, c.want)
		}
	}
}

// TestSharpCornerRunOutDropsTheDoubleCountedArea: each flap counted its region twice — once on the
// untrimmed side face and once on the cap. Removing both recovers 4·notch of area over the two
// terminals, where notch = r²(1−π/4) is the corner the fillet takes out.
//
// The claim is a DELTA against the old build's 94.207048, and that baseline is a DefaultQuality
// TESSELLATED measurement — the only meter ops.BodyGeometryProperties had when it was taken. Since
// c94e5b61 that function integrates the analytic B-rep instead, which reads this body 0.012998
// higher (an inscribed mesh under-measures every curved face); more than the 0.01 window, so a
// baseline in one meter can no longer be compared against a reading in the other. The delta is
// therefore measured where it is meaningful, and the exact surface is gated separately below
// (Oblikovati/Oblikovati#3453).
func TestSharpCornerRunOutDropsTheDoubleCountedArea(t *testing.T) {
	t.Parallel()
	const r = 0.25
	res := sharpCornerRunOut(t)
	notch := r * r * (1 - stdmath.Pi/4)
	const before = 94.207048
	mesh, _ := tessellate.TessellateBody(res, ops.DefaultQuality())
	got := meshArea(mesh)
	want := before - 4*notch
	if stdmath.Abs(got-want) > 0.01 {
		t.Errorf("faceted area = %.6f, want ≈%.6f (the old %.6f less 4·notch = %.6f)", got, want, before, 4*notch)
	}
}

// TestSharpCornerRunOutAreaConvergesOnTheExactSurface is what the faceted delta above cannot see: a
// chord measurement is blind to any error smaller than its own deficit. The analytic integral is not,
// and it is quality-independent, so the mesh must climb TOWARDS it as the facets refine and never
// past it — an inscribed mesh runs under a curved face, never over. A build whose exact surface had
// gained or lost a sliver would break the ordering here while sitting well inside the 0.01 window.
func TestSharpCornerRunOutAreaConvergesOnTheExactSurface(t *testing.T) {
	t.Parallel()
	res := sharpCornerRunOut(t)
	exact, ok := ops.AnalyticGeometryProperties(res)
	if !ok {
		t.Fatal("the run-out body no longer integrates analytically; the exact area has no oracle")
	}
	coarse, _ := tessellate.TessellateBody(res, ops.DefaultQuality())
	fine, _ := tessellate.TessellateBody(res, ops.PropertyQuality())
	cArea, fArea := meshArea(coarse), meshArea(fine)
	if !(cArea < fArea && fArea < exact.Area) {
		t.Errorf("areas must rise coarse < fine < exact; got %.6f, %.6f, %.6f", cArea, fArea, exact.Area)
	}
	if rel := (exact.Area - fArea) / exact.Area; rel > 1e-4 {
		t.Errorf("at PropertyQuality the mesh is still %.3g short of the exact %.6f (want ≤1e-4)", rel, exact.Area)
	}
}

// sideFaceAtY4 returns the result's planar face lying in y = 4.
func sideFaceAtY4(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok || float64(pl.Normal().Cross(gmath.V3(0, 1, 0)).Length()) > 1e-9 {
			continue
		}
		if stdmath.Abs(float64(pl.Origin.Y)-4) < 1e-9 {
			return f
		}
	}
	t.Fatal("no planar face in y = 4")
	return nil
}

// loopTouches reports whether any boundary vertex of f sits at p.
func loopTouches(f *topo.Face, p gmath.Point3) bool {
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			for _, v := range u.Edge().Vertices() {
				if float64(v.Point().DistanceTo(p)) < 1e-9 {
					return true
				}
			}
		}
	}
	return false
}
