// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The bore-wall far-cap OUTWARD extension primitives (corner-blend-weld Piece 3, L9): a notch (R+r)
// fillet's far cross-section reaches r PAST the rim, so the retrim must GROW a collinear survivor edge
// to the rail landing / bite tip instead of biting an inward corner. These pin that growth directly, and
// the NO-OP guard that keeps every convex green (rail landings already on the loop) byte-identical.

const boreTol = 1e-6

// ptNear reports whether two points coincide within tol.
func ptNear(a, b math.Point3, tol float64) bool { return float64(a.DistanceTo(b)) <= tol }

// straightSeg is a bare straight endSeg for the extension tests.
func boreSeg(a, b math.Point3) endSeg { return endSeg{from: a, to: b} }

// TestGrownStraightSeg_ExtendsBeyondEndpoints pins grownStraightSeg: a point collinear and beyond an
// endpoint grows that side; an interior point, an off-line point, and an arc are rejected (ok=false).
func TestGrownStraightSeg_ExtendsBeyondEndpoints(t *testing.T) {
	t.Parallel()
	s := boreSeg(math.P3(50, 0, 100), math.P3(50, 20, 100))
	if g, ok := grownStraightSeg(s, math.P3(50, 25, 100), boreTol); !ok || !ptNear(g.to, math.P3(50, 25, 100), boreTol) {
		t.Fatalf("beyond-to: ok=%v to=%v, want (50,25,100)", ok, g.to)
	}
	if g, ok := grownStraightSeg(s, math.P3(50, -5, 100), boreTol); !ok || !ptNear(g.from, math.P3(50, -5, 100), boreTol) {
		t.Fatalf("beyond-from: ok=%v from=%v, want (50,-5,100)", ok, g.from)
	}
	if _, ok := grownStraightSeg(s, math.P3(50, 10, 100), boreTol); ok {
		t.Fatalf("interior point must not grow (insertSplits handles it)")
	}
	if _, ok := grownStraightSeg(s, math.P3(55, 25, 100), boreTol); ok {
		t.Fatalf("off-line point must not grow")
	}
	arc := endSeg{from: math.P3(0, 0, 0), to: math.P3(10, 0, 0), arc: true}
	if _, ok := grownStraightSeg(arc, math.P3(-5, 0, 0), boreTol); ok {
		t.Fatalf("an arc seg must never grow")
	}
}

// TestExtendStraightSegToLanding_NoOpWhenOnLoop pins the do-no-harm guard: a landing point already on
// the loop leaves the segs UNCHANGED (the convex-green byte-identity path); only an off-loop collinear
// point grows an edge.
func TestExtendStraightSegToLanding_NoOpWhenOnLoop(t *testing.T) {
	t.Parallel()
	segs := []endSeg{
		boreSeg(math.P3(0, 0, 100), math.P3(50, 0, 100)),
		boreSeg(math.P3(50, 0, 100), math.P3(50, 20, 100)),
		boreSeg(math.P3(50, 20, 100), math.P3(0, 20, 100)),
		boreSeg(math.P3(0, 20, 100), math.P3(0, 0, 100)),
	}
	if got := extendStraightSegToLanding(segs, math.P3(25, 0, 100), boreTol); len(got) != 4 || got[0].to != segs[0].to {
		t.Fatalf("on-loop landing must be a no-op (byte-identical), got %d segs", len(got))
	}
	got := extendStraightSegToLanding(segs, math.P3(50, 25, 100), boreTol) // off-loop, collinear with seg[1]
	if !ptNear(got[1].to, math.P3(50, 25, 100), boreTol) {
		t.Fatalf("off-loop collinear landing must grow seg[1] to (50,25,100), got to=%v", got[1].to)
	}
}

// TestBoreExtendBite_GrowsLoopOutward pins boreExtendBite on L9's flat radial notch face: the rectangle
// y∈[0,20] grows its top edge out to the tip (50,25,100) and splices the far-cap bite down to the foot
// (50,20,95) on the rim edge, dropping the (50,20,100) corner — a watertight loop that is longer than
// the input (an OUTWARD growth, not an inward bite).
func TestBoreExtendBite_GrowsLoopOutward(t *testing.T) {
	t.Parallel()
	segs := []endSeg{
		boreSeg(math.P3(50, 0, 0), math.P3(50, 0, 100)),
		boreSeg(math.P3(50, 0, 100), math.P3(50, 20, 100)), // top edge, grows to (50,25,100)
		boreSeg(math.P3(50, 20, 100), math.P3(50, 20, 0)),  // rim edge, carries the foot at z=95
		boreSeg(math.P3(50, 20, 0), math.P3(50, 0, 0)),
	}
	bite := arcBite(math.P3(50, 20, 95), math.P3(50, 25, 100)) // foot on the rim, tip off the loop
	out, ok := boreExtendBite(segs, bite, boreTol)
	if !ok {
		t.Fatalf("boreExtendBite declined L9's flat-radial far cap")
	}
	if !closedLoop(out, boreTol) {
		t.Fatalf("boreExtendBite produced an open loop: %v", segEndpoints(out))
	}
	if !hasVertex(out, math.P3(50, 25, 100), boreTol) || !hasVertex(out, math.P3(50, 20, 95), boreTol) {
		t.Fatalf("grown loop must reach the tip (50,25,100) and the foot (50,20,95): %v", segEndpoints(out))
	}
	// Convex do-no-harm: a bite whose endpoints are BOTH on the loop is not this case (declines here,
	// so spliceBite falls through to the byte-identical spliceCornerBite).
	onLoop := arcBite(math.P3(50, 20, 95), math.P3(50, 10, 100))
	if _, ok := boreExtendBite(segs, onLoop, boreTol); ok {
		t.Fatalf("boreExtendBite must decline a bite with both endpoints on the loop (convex path)")
	}
}

// arcBite builds a bite endSeg (arc) between two feet, with a placeholder mid — boreExtendBite reads
// only the endpoints and carries the curve through.
func arcBite(from, to math.Point3) endSeg {
	mid := from.Midpoint(to)
	arc, err := geom.Arc3dByThreePoints(from, math.P3(float64(mid.X)+0.01, float64(mid.Y), float64(mid.Z)), to)
	if err != nil {
		return endSeg{from: from, to: to, mid: mid, arc: true}
	}
	return endSeg{from: from, to: to, curve: arc, mid: mid, arc: true}
}

// closedLoop reports whether each seg's to meets the next seg's from (a closed ring).
func closedLoop(segs []endSeg, tol float64) bool {
	for i, s := range segs {
		if !ptNear(s.to, segs[(i+1)%len(segs)].from, tol) {
			return false
		}
	}
	return true
}

// hasVertex reports whether any seg endpoint equals p.
func hasVertex(segs []endSeg, p math.Point3, tol float64) bool {
	for _, s := range segs {
		if ptNear(s.from, p, tol) || ptNear(s.to, p, tol) {
			return true
		}
	}
	return false
}

// segEndpoints lists the from points for a failure message.
func segEndpoints(segs []endSeg) []math.Point3 {
	out := make([]math.Point3, len(segs))
	for i, s := range segs {
		out[i] = s.from
	}
	return out
}
