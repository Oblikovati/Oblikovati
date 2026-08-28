// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
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
// This is the adaptive midpoint bisection used for DISPLAY and every non-boolean consumer. The
// boolean input takes a CANONICAL absolute-angle sampling of its circle/arc edges instead, but
// installs it as a temporary snapped polyline (applyBooleanConformance), which discretizeEdge
// consults first — so only the boolean operands conform, and display meshing is untouched
// (ADR-0056/#2167; the global-canonical variant folded some occtparity display meshes — see #2168).
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

// conformalCircularSamples returns canonical absolute-angle samples for a circle or arc
// EDGE (ADR-0056) together with each sample's parameter in the curve's own [0,1] domain:
// identical interior points for any two coincident circular curves, so the two operands of
// a boolean conform on a shared curved boundary. A full circle is anchored on the edge's
// seam vertex; an arc runs between its endpoints. Returns ok=false for a non-circular
// curve, whose caller keeps the adaptive path.
func conformalCircularSamples(e *topo.Edge, chordTol, angleTol float64) (pts []math.Point3, params []float64, ok bool) {
	switch k := e.Geometry().(type) {
	case geom.Circle:
		pts, params = geom.CircleConformalSamples(k, e.StartVertex().Point(), chordTol, angleTol)
		return pts, params, true
	case geom.Arc3d:
		pts, params = geom.ArcConformalSamples(k, chordTol, angleTol)
		pts[0], pts[len(pts)-1] = e.StartVertex().Point(), e.EndVertex().Point()
		return pts, params, true
	}
	return nil, nil, false
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
