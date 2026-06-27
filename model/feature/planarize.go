// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
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

// curvedFaceCount counts a body's non-planar faces. A BARE analytic primitive (cylinder/cone/sphere/torus
// solid) has exactly one — its single curved wall; a composite (a washer's two cylinders, a filleted edge)
// has more, which the curved boolean does not cut as a primitive.
func curvedFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, planar := f.Geometry().(geom.Plane); !planar {
			n++
		}
	}
	return n
}

// exactlyOneCurvedPrimitive reports whether exactly one of target/tool is a BARE analytic primitive
// (a single curved face — an extruded cylinder/cone, a revolved torus, a sphere) and the other is
// all-planar. Only then can the exact M2 curved boolean keep the curved surface: a box drilled by a
// cylinder tool keeps its cylindrical hole wall (#1472), the mirror of a curved solid cut by a planar
// box (#1334/#1335). The single-curved-face gate is deliberately tight — a composite curved body (a
// washer's two cylinder walls, a filleted edge) is NOT a primitive the half-space cut handles, so it
// stays on the faceted planar path; CurvedBoolean can over-match such a body and cut it wrongly.
func exactlyOneCurvedPrimitive(target, tool *topo.Body) bool {
	tc, oc := curvedFaceCount(target), curvedFaceCount(tool)
	return (tc == 1 && oc == 0) || (tc == 0 && oc == 1)
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
	cyl, base, height, ok := simpleCylinderParams(b)
	if !ok {
		return nil
	}
	// Re-facet in the cylinder's ORIGINAL generating frame (Ref = the sketch +X recorded at
	// extrude time) AND under its ORIGINAL feature lineage, not an arbitrary axis-derived frame /
	// synthetic feature name. Reference keys are lineage-derived, so reproducing both makes this
	// prism byte-identical (geometry + keys) to a direct faceted extrude of the same circle —
	// keeping edge/face identity stable for downstream dress-up and boolean (#129).
	feat = originalFeature(b, feat)
	return buildPrism(regularPolygon(cyl.Radius, 24), planeFromFrame(base, cyl.AxisDir, cyl.Ref), span{near: 0, far: height}, 0, feat)
}

// simpleCylinderParams returns the geometry of a body that is EXACTLY one analytic cylinder (1
// geom.Cylinder side face + 2 planar caps) — the extrude-circle / revolved-disc result: the cylinder
// surface, the base centre (lower cap, on the axis), and the height between caps. ok is false for any
// other shape (already planar, a tube, a fillet/cone, a partially-cut cylinder, …).
func simpleCylinderParams(b *topo.Body) (cyl geom.Cylinder, base math.Point3, height float64, ok bool) {
	cyl, caps, ok := soleCylinderAndCaps(b)
	if !ok {
		return geom.Cylinder{}, math.Point3{}, 0, false
	}
	axis := cyl.AxisDir.AsVector()
	lo, hi := capExtents(cyl.Origin, axis, caps)
	if hi-lo <= 0 {
		return geom.Cylinder{}, math.Point3{}, 0, false
	}
	return cyl, cyl.Origin.TranslateBy(axis.Scale(math.Scalar(lo))), hi - lo, true
}

// soleCylinderAndCaps returns the body's single geom.Cylinder face and its two planar cap faces,
// or ok=false unless the body is EXACTLY one cylinder (radius>0) plus two planes.
func soleCylinderAndCaps(b *topo.Body) (cyl geom.Cylinder, caps []*topo.Face, ok bool) {
	if b == nil || len(b.Faces()) != 3 {
		return geom.Cylinder{}, nil, false
	}
	for _, f := range b.Faces() {
		switch g := f.Geometry().(type) {
		case geom.Cylinder:
			if ok {
				return geom.Cylinder{}, nil, false // a second cylinder: not a simple cylinder
			}
			cyl, ok = g, true
		case geom.Plane:
			caps = append(caps, f)
		default:
			return geom.Cylinder{}, nil, false
		}
	}
	if !ok || cyl.Radius <= 0 || len(caps) != 2 {
		return geom.Cylinder{}, nil, false
	}
	return cyl, caps, true
}

// capExtents returns the min and max axial offset (from origin along axis) of the two cap planes.
func capExtents(origin math.Point3, axis math.Vector3, caps []*topo.Face) (lo, hi float64) {
	p0 := float64(origin.VectorTo(caps[0].Geometry().(geom.Plane).Origin).Dot(axis))
	p1 := float64(origin.VectorTo(caps[1].Geometry().(geom.Plane).Origin).Dot(axis))
	return min(p0, p1), max(p0, p1)
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
