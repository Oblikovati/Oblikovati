// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cross-operand vertex-on-edge imprinting for the exact boolean (ADR-0054).
//
// The mesh boolean co-refines each operand INDEPENDENTLY, so it imprints edge-edge and
// face-face crossings but not a VERTEX of one operand lying on an EDGE of the other. When
// a D-profile's chord corner sits on a cylinder's rim circle at a NON-canonical angle (the
// #2167 join), the cylinder's rim has no vertex there, so near the corner the two operands'
// tessellations do not conform — a fan of tiny opposite-facing sliver triangles the exact
// classification then keeps as spurious internal caps. The corner is OUTSIDE the cylinder's
// inscribed rim polygon, so it cannot be recovered by splitting a soup edge; it has to enter
// at edge discretization.
//
// So before tessellating an operand for the boolean, imprint the OTHER operand's vertices
// that lie on this operand's edges: build the edge's discretization with those points
// inserted and install it as the edge's snapped polyline. Both faces of the edge already
// share that polyline (topo.Edge.SnappedCurve), so both conform, and the two operands now
// meet exactly at the shared corner. The snapped polyline is set only for the duration of
// the tessellation and then restored, so the operand bodies are left unchanged.

// crossOperandImprints returns, for each edge of onto, the vertices of from that lie in the
// edge's interior (on its curve, not at either endpoint) — the points onto must sample so
// its tessellation conforms with from along that edge.
func crossOperandImprints(onto, from *topo.Body, tol float64) map[*topo.Edge][]math.Point3 {
	verts := from.Vertices()
	out := make(map[*topo.Edge][]math.Point3)
	for _, e := range onto.Edges() {
		// Only a full CIRCLE rim needs imprinting: its faceted chord dips inside the true
		// radius, so a corner at a non-canonical angle lands OUTSIDE it and leaves a sliver.
		// A straight edge carries a coincident vertex exactly on its chord (no gap), and
		// reconstruction has no sub-segment matcher for it — imprinting it would only make an
		// unmatchable run and decline the rebuild. Sub-arcs of an imprinted circle are handled
		// by matchSubArc; other curved kinds are a follow-up when their SSI layer lands.
		if _, isCircle := e.Geometry().(geom.Circle); !isCircle {
			continue
		}
		for _, v := range verts {
			if p := v.Point(); vertexOnEdgeInterior(e, p, tol) {
				out[e] = append(out[e], p)
			}
		}
	}
	return out
}

// vertexOnEdgeInterior reports whether p lies on edge e's curve strictly between its
// endpoints (so imprinting it splits the edge, rather than duplicating a vertex).
func vertexOnEdgeInterior(e *topo.Edge, p math.Point3, tol float64) bool {
	if p.DistanceTo(e.StartVertex().Point()) <= tol || p.DistanceTo(e.EndVertex().Point()) <= tol {
		return false
	}
	return onCurve(e.Geometry(), p, tol)
}

// applyBooleanConformance installs, on every CIRCLE/ARC edge of the body, a snapped polyline equal
// to its CANONICAL absolute-angle sampling (with any cross-operand imprint points inserted), and
// returns a closure that restores each edge afterwards. Because discretizeEdge (and
// tessellateEdgeWithParams) consult the snapped polyline first, the boolean tessellates b's
// circular boundaries canonically — so two operands sharing a circle discretize it identically —
// while DISPLAY and every other consumer, which never call this, keep the adaptive sampling. This
// scopes the conformance to exactly the boolean input (ADR-0054/#2167). The snapped polyline is set
// only for the tessellation and restored, so the operand body is left unchanged.
func applyBooleanConformance(b *topo.Body, imprints map[*topo.Edge][]math.Point3, q Quality) func() {
	type prev struct {
		e    *topo.Edge
		poly []math.Point3
		res  float64
	}
	var saved []prev
	for _, e := range b.Edges() {
		poly, ok := conformalPolyline(e, imprints[e], q)
		if !ok {
			continue // non-circular edge: adaptive sampling already conforms (its chords are exact)
		}
		saved = append(saved, prev{e, e.SnappedCurve(), e.Tolerance()})
		e.SetSnappedCurve(poly, e.Tolerance())
	}
	return func() {
		for _, s := range saved {
			s.e.SetSnappedCurve(s.poly, s.res)
		}
	}
}

// taggedSoupWithImprints is bodyToTaggedSoup with the boolean-input conformance sampling (canonical
// circle/arc edges + cross-operand imprints) applied to the body's edges for the duration of the
// tessellation, then restored.
func taggedSoupWithImprints(b *topo.Body, q Quality, tagBase int, imprints map[*topo.Edge][]math.Point3) (meshbool.TaggedSoup, []faceSurfaceRef) {
	defer applyBooleanConformance(b, imprints, q)()
	return bodyToTaggedSoup(b, q, tagBase)
}

// conformalPolyline returns a circle/arc edge's canonical absolute-angle sampling with the given
// imprint points inserted at the segment each lies on; ok=false for a non-circular edge.
func conformalPolyline(e *topo.Edge, imprints []math.Point3, q Quality) ([]math.Point3, bool) {
	pts, _, ok := conformalCircularSamples(e, q.tol(), q.angleTol())
	if !ok {
		return nil, false
	}
	for _, p := range imprints {
		pts = insertOnPolyline(pts, p)
	}
	return pts, true
}

// insertOnPolyline inserts p into the segment of poly it lies on (least added detour),
// skipping insertion when p already coincides with a segment endpoint.
func insertOnPolyline(poly []math.Point3, p math.Point3) []math.Point3 {
	best, bestDetour := -1, stdmath.Inf(1)
	for i := 0; i+1 < len(poly); i++ {
		d := poly[i].DistanceTo(p) + p.DistanceTo(poly[i+1]) - poly[i].DistanceTo(poly[i+1])
		if d < bestDetour {
			best, bestDetour = i, d
		}
	}
	if best < 0 {
		return poly
	}
	out := append([]math.Point3(nil), poly[:best+1]...)
	out = append(out, p)
	return append(out, poly[best+1:]...)
}
