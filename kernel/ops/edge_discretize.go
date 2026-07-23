// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved edges must be sampled into the SAME chord polyline by every face that uses
// them, or two faces meeting at an arc would tessellate it differently and leave a
// T-junction (a crack) in the mesh. discretizeEdge derives the polyline only from the
// edge (its curve, in its natural start→end parameter order), so any caller gets the
// identical result; loopBoundary then orients per edge use and concatenates. A straight
// edge yields just its two endpoints, so polyhedral faces are unaffected.

// discretizeEdge samples an edge's curve into a chord polyline (start vertex → end
// vertex) meeting the chordal tolerance. Endpoints are snapped to the edge's vertices
// so adjacent edges share exact points. A healed (imported) edge returns its stored
// on-surface polyline verbatim (M25 PBI-324) so both faces mesh the identical boundary.
//
// #2009: a straight/near-straight edge's 2-point result is then passed through
// densifyStarvedRail (nurbs_pcurve_mesh.go), which extends it ONLY when the edge is shared by a
// high-aspect B-spline face (aspectDensifyThreshold) — otherwise a no-op, so this stays the exact
// 2-point straight-edge result for every other caller. Applying the extension HERE, inside the one
// function every face already shares for an edge (see the package comment above), is what keeps
// both sides of a shared rail in agreement: a lower-aspect or non-B-spline neighbour calling
// discretizeEdge for the SAME edge gets the IDENTICAL denser polyline, not just a geometrically
// coincident but topologically cracked one (see densifyStarvedRail's doc for the regression this
// closes).
func discretizeEdge(e *topo.Edge, q Quality) []math.Point3 {
	if snapped := e.SnappedCurve(); snapped != nil {
		return densifyStarvedRail(e, snapped)
	}
	return densifyStarvedRail(e, sampleEdgeCurve(e, q))
}

// sampleEdgeCurve samples an edge's curve into a chord polyline directly (ignoring any healing
// snap) — the raw discretization [discretizeEdge] derives from, and what [snapEdge] re-projects.
func sampleEdgeCurve(e *topo.Edge, q Quality) []math.Point3 {
	c := e.Geometry()
	lo, hi := c.Domain()
	ts := adaptiveParams(func(t float64) math.Point3 { return c.PointAt(t) }, lo, hi, q.tol(), q.angleTol())
	pts := make([]math.Point3, len(ts))
	for i, t := range ts {
		pts[i] = c.PointAt(t)
	}
	pts[0] = e.StartVertex().Point()
	pts[len(pts)-1] = e.EndVertex().Point()
	return pts
}

// loopBoundary returns a loop's boundary as an ordered point ring, each edge sampled
// by [discretizeEdge] and oriented to the edge use, with shared vertices not repeated.
// Both faces of a curved edge get matching interior points (watertight, no T-junctions).
func loopBoundary(l *topo.Loop, q Quality) []math.Point3 {
	var out []math.Point3
	for _, u := range l.EdgeUses() {
		pts := discretizeEdge(u.Edge(), q)
		if u.Reversed() {
			pts = reverse3(pts)
		}
		if len(out) > 0 {
			pts = pts[1:] // drop the point shared with the previous edge's end
		}
		out = append(out, pts...)
	}
	// Model-relative closure test (ADR-0042): the duplicate-end threshold scales with
	// the loop's size so a sub-µm loop is not falsely seen as closed.
	if n := len(out); n > 1 && out[0].DistanceTo(out[n-1]) < ResolutionForPoints(out).Weld() {
		out = out[:n-1] // drop the closing duplicate (last == first)
	}
	return out
}

// reverse3 returns the points in reverse order (new slice).
func reverse3(pts []math.Point3) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}
