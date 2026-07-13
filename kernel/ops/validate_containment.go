// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// holeEdgeSamples is how many points per edge the containment check samples. A full-ellipse hole is a
// single closed edge whose two endpoints weld to ONE seam vertex, so a vertex-only bbox would collapse
// to a point and miss a protrusion — several samples per edge are required to trace the real rim.
const holeEdgeSamples = 6

// checkHoleContainment flags any PLANAR face whose hole loop is not strictly inside its outer loop — the
// B-rep invariant that a hole is an interior void, not a protrusion. A fillet that shrinks a face's outer
// loop into a coplanar hole leaves the hole poking through its own boundary (the base-plane defect behind
// the elliptical-prism blend cases): the tessellator then meshes malformed input and emits phantom
// "fill"/crack artifacts that look like meshing bugs. Reported as a distinct HolesContained flag + issues
// rather than folded into Valid, so it is a diagnostic tripwire until the fillet trim that fixes it lands.
// Curved-face containment lives in surface (u,v) space and is a separate concern, out of scope here.
func (r *ValidationReport) checkHoleContainment(b *topo.Body) {
	r.HolesContained = true
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Plane); !ok || len(f.Loops()) < 2 {
			continue
		}
		flat := planeProjector(f.Geometry().NormalAt(0, 0))
		outer := planarLoopSamples(outerLoopOf(f), flat)
		if len(outer) < 3 {
			continue
		}
		for _, l := range f.Loops() {
			if l.IsOuter() || loopWithin(planarLoopSamples(l, flat), outer) {
				continue
			}
			r.HolesContained = false
			r.Issues = append(r.Issues, fmt.Sprintf(
				"hole loop %d protrudes outside the outer loop of planar face %d (malformed B-rep face)",
				l.ID(), f.ID()))
		}
	}
}

// outerLoopOf returns the face's outer loop, or nil if it has none.
func outerLoopOf(f *topo.Face) *topo.Loop {
	for _, l := range f.Loops() {
		if l.IsOuter() {
			return l
		}
	}
	return nil
}

// planarLoopSamples projects holeEdgeSamples points from each of the loop's edges onto the face plane.
// Sampling the curve (not just its vertices) is required because a closed curved edge — a full-ellipse
// rim — has a single seam vertex yet a wide extent, so vertices alone would report a point, not the rim.
func planarLoopSamples(l *topo.Loop, flat func(math.Point3) math.Point2) []math.Point2 {
	if l == nil {
		return nil
	}
	var pts []math.Point2
	for _, u := range l.EdgeUses() {
		c := u.Edge().Geometry()
		for i := 0; i < holeEdgeSamples; i++ {
			t := float64(i) / holeEdgeSamples
			if u.Reversed() { // honour loop-traversal direction so the samples form an ordered ring
				t = 1 - t
			}
			pts = append(pts, flat(c.PointAt(t)))
		}
	}
	return pts
}

// loopWithin reports whether every sampled point of hole lies inside the outer polygon. A point is a
// genuine protrusion only when it is outside AND farther from the boundary than a model-relative slack —
// so a hole tangent to the outer loop (a vertex exactly on the boundary, MAXPROTRUDE≈0) is NOT falsely
// flagged, while the fillet-② protrusion (a rim poking whole units past the shrunken outer) is. Two-tier:
// a fast axis-aligned bbox accept skips the O(n·m) distance pass for a hole whose extent is clearly inside.
func loopWithin(hole, outer []math.Point2) bool {
	if len(hole) == 0 {
		return true
	}
	tol := containmentSlack(outer)
	if bboxInsideBy(hole, outer, tol) {
		return true // hole bbox strictly inside outer bbox by more than slack: cannot protrude
	}
	for _, p := range hole {
		if !pointInLoop2D(p, outer) && distToPolygon(p, outer) > tol {
			return false
		}
	}
	return true
}

// containmentSlack is the model-relative tangency tolerance: a hole point closer than this to the outer
// boundary counts as touching, not protruding. Scaled to the outer loop's extent (ADR-0042), never a
// bare constant.
func containmentSlack(outer []math.Point2) float64 {
	oxn, oxx, oyn, oyx := bounds2D(outer)
	return 1e-6 * (oxx - oxn + oyx - oyn)
}

// bboxInsideBy reports whether inner's axis-aligned bounds sit inside outer's by more than slack on every
// side — a conservative fast accept (a hole this far inside the outer bbox cannot reach the boundary).
func bboxInsideBy(inner, outer []math.Point2, slack float64) bool {
	ixn, ixx, iyn, iyx := bounds2D(inner)
	oxn, oxx, oyn, oyx := bounds2D(outer)
	return ixn > oxn+slack && ixx < oxx-slack && iyn > oyn+slack && iyx < oyx-slack
}

// distToPolygon returns the distance from p to the nearest edge of the closed polygon poly.
func distToPolygon(p math.Point2, poly []math.Point2) float64 {
	best := stdmath.Inf(1)
	for i, n := 0, len(poly); i < n; i++ {
		best = stdmath.Min(best, distToSegment(p, poly[i], poly[(i+1)%n]))
	}
	return best
}

// distToSegment returns the distance from p to segment ab.
func distToSegment(p, a, b math.Point2) float64 {
	vx, vy := b.X-a.X, b.Y-a.Y
	l2 := vx*vx + vy*vy
	t := 0.0
	if l2 > 0 {
		t = stdmath.Max(0, stdmath.Min(1, ((p.X-a.X)*vx+(p.Y-a.Y)*vy)/l2))
	}
	return stdmath.Hypot(p.X-(a.X+t*vx), p.Y-(a.Y+t*vy))
}
