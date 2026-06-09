// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati/kernel/geom"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// Extruding a circle now yields a TRUE cylinder (an analytic geom.Cylinder side face) so thread works
// (Oblikovati/Oblikovati#129). But the planar B-rep boolean (and hull) cannot consume a FULL PERIODIC
// cylinder face — the boolean hangs on one. (Trimmed cylinder PATCHES, e.g. fillet blends, the boolean
// handles fine — so we must NOT touch those.) Until the boolean/hull are curved-aware, an op that needs
// a planar B-rep re-facets only a SIMPLE cylinder body (1 cylinder face + 2 planar caps — the
// extrude-circle result) into a clean N-gon prism; any other body (a fillet/cone/partially-cut
// cylinder) is left untouched, so this never disturbs existing curved geometry.

// hasCurvedFace reports whether any face of b is non-planar.
func hasCurvedFace(b *topo.Body) bool {
	for _, f := range b.Faces() {
		if _, planar := f.Geometry().(geom.Plane); !planar {
			return true
		}
	}
	return false
}

// planarized converts a body with analytic curved faces into a planar B-rep the exact boolean can
// consume (it hangs on a full periodic curved face, #129). A SIMPLE extrude-circle cylinder becomes
// a clean, key-stable N-gon prism (the fast path that keeps downstream edge identity); any OTHER
// curved body (a revolved tube/shaft, a multi-segment surface of revolution) is faceted into a
// triangle cage via ops.Facet. An already-planar body is returned unchanged. nil-safe.
func planarized(b *topo.Body, feat string) *topo.Body {
	if prism := planarizeSimpleCylinder(b, feat+"/planar"); prism != nil {
		return prism
	}
	if b != nil && hasCurvedFace(b) {
		if faceted := ops.Facet(b, originalFeature(b, feat)); faceted != nil {
			return faceted
		}
	}
	return b
}

// planarizeSimpleCylinder rebuilds a body that is exactly one analytic cylinder (1 geom.Cylinder side
// face + 2 planar caps) as a clean 24-gon prism (24 = the sketch's circle sampling, so the facet
// topology matches a faceted extrude of the same circle) with the same axis/radius/extent. nil if the
// body is not that simple shape.
func planarizeSimpleCylinder(b *topo.Body, feat string) *topo.Body {
	if b == nil {
		return nil
	}
	faces := b.Faces()
	if len(faces) != 3 {
		return nil
	}
	var cyl geom.Cylinder
	haveCyl := false
	for _, f := range faces {
		switch g := f.Geometry().(type) {
		case geom.Cylinder:
			if haveCyl {
				return nil
			}
			cyl, haveCyl = g, true
		case geom.Plane:
		default:
			return nil
		}
	}
	if !haveCyl || cyl.Radius <= 0 {
		return nil
	}
	axis := cyl.AxisDir
	var proj []float64
	for _, f := range faces {
		if pl, ok := f.Geometry().(geom.Plane); ok {
			proj = append(proj, cyl.Origin.VectorTo(pl.Origin).Dot(axis.AsVector()))
		}
	}
	if len(proj) != 2 {
		return nil
	}
	lo, hi := proj[0], proj[1]
	if lo > hi {
		lo, hi = hi, lo
	}
	height := hi - lo
	if height <= 0 {
		return nil
	}
	base := cyl.Origin.TranslateBy(axis.AsVector().Scale(math.Scalar(lo)))
	// Re-facet in the cylinder's ORIGINAL generating frame (Ref = the sketch +X recorded at
	// extrude time) AND under its ORIGINAL feature lineage, not an arbitrary axis-derived frame /
	// synthetic feature name. Reference keys are lineage-derived, so reproducing both makes this
	// prism byte-identical (geometry + keys) to a direct faceted extrude of the same circle —
	// keeping edge/face identity stable for downstream dress-up and boolean (#129).
	feat = originalFeature(b, feat)
	return buildPrism(regularPolygon(cyl.Radius, 24), planeFromFrame(base, axis, cyl.Ref), span{near: 0, far: height}, 0, feat)
}

// originalFeature recovers the name of the feature that generated body b from its body-level
// lineage token, falling back to fallback when the lineage is empty. A re-faceted prism reuses
// this so its entities carry the SAME lineage (and reference keys) a direct faceted extrude
// would mint (#129); the synthetic "combine-*"/"…-planar" names broke that, shifting edge[0].
func originalFeature(b *topo.Body, fallback string) string {
	toks := b.Lineage().Tokens()
	if len(toks) == 0 || toks[0].Feature == "" {
		return fallback
	}
	return toks[0].Feature
}

// planarizeForEdges re-facets a simple cylinder body for an edge op (chamfer/fillet) and maps each
// selected edge onto the prism's edges that lie along it — a circular rim maps to its faceted segments,
// a straight edge to its single counterpart. Returns the body and edges unchanged when the body is not
// a simple cylinder (so existing dress-up on fillets/cones/blocks is untouched). A true conical
// chamfer/fillet of a circular edge is future work (#127); today we re-facet so the edge op stays
// robust on a cylinder instead of failing on a degenerate closed edge.
func planarizeForEdges(body *topo.Body, edges []*topo.Edge, feat string) (*topo.Body, []*topo.Edge) {
	pb := planarized(body, feat)
	if pb == body {
		return body, edges
	}
	var mapped []*topo.Edge
	for _, pe := range pb.Edges() {
		for _, orig := range edges {
			if edgeOnEdge(pe, orig) {
				mapped = append(mapped, pe)
				break
			}
		}
	}
	return pb, mapped
}

// edgeOnEdge reports whether both endpoints of pe lie on orig's curve — i.e. pe is a faceted segment of
// orig (or orig itself).
func edgeOnEdge(pe, orig *topo.Edge) bool {
	c := orig.Geometry()
	return pointOnCurve(pe.StartVertex().Point(), c) && pointOnCurve(pe.EndVertex().Point(), c)
}

// pointOnCurve reports whether p lies on curve c (within 1 µm), by sampling c.
func pointOnCurve(p math.Point3, c geom.Curve3) bool {
	lo, hi := c.Domain()
	const n = 96
	for i := 0; i <= n; i++ {
		if p.DistanceTo(c.PointAt(lo+(hi-lo)*float64(i)/float64(n))) < 1e-4 {
			return true
		}
	}
	return false
}
