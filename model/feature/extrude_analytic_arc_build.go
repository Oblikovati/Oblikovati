// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// assembleAnalyticExtrusion wires the CCW segment ring into a watertight solid: one vertex per corner
// at each cap, one cap edge per segment (line or arc) at each cap, one vertical seam per corner, two
// planar caps, and one side face per segment (planar for a line, a true partial cylinder for an arc).
// It mirrors buildExtrusionShell's construction (prismEdges/addCaps/addSides) so lineage tokens match.
func assembleAnalyticExtrusion(segs []analyticSeg, plane sketch.Plane, sp span, feat string) *topo.Body {
	n := len(segs)
	normal := plane.Normal().AsVector()
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok(feat, "body", 0)))

	bottom := make([]*topo.Vertex, n)
	top := make([]*topo.Vertex, n)
	for i, s := range segs {
		bp := plane.ToModel(s.a).TranslateBy(normal.Scale(sp.near))
		tp := plane.ToModel(s.a).TranslateBy(normal.Scale(sp.far))
		bottom[i] = bld.AddVertex(bp, topo.NewLineage(topo.Tok(feat, "vertex", i)))
		top[i] = bld.AddVertex(tp, topo.NewLineage(topo.Tok(feat, "vertex", n+i)))
	}

	be, te, ve := analyticPrismEdges(bld, segs, bottom, top, plane, sp, normal, feat)
	addAnalyticCaps(bld, bottom, top, be, te, normal, feat)
	addAnalyticSides(bld, segs, bottom, top, be, te, ve, plane, sp, normal, feat)
	return bld.Build()
}

// analyticPrismEdges builds the bottom cap edge, top cap edge (each a line or arc following its
// segment), and the vertical seam edge for every segment, returning the three edge rings.
func analyticPrismEdges(bld *topo.Builder, segs []analyticSeg, bottom, top []*topo.Vertex, plane sketch.Plane, sp span, normal math.Vector3, feat string) (be, te, ve []*topo.Edge) {
	n := len(segs)
	be, te, ve = make([]*topo.Edge, n), make([]*topo.Edge, n), make([]*topo.Edge, n)
	for i, s := range segs {
		j := (i + 1) % n
		be[i] = capEdge(bld, s, bottom[i], bottom[j], plane, sp.near, normal, topo.NewLineage(topo.Tok(feat, "bottom-edge", i)))
		te[i] = capEdge(bld, s, top[i], top[j], plane, sp.far, normal, topo.NewLineage(topo.Tok(feat, "top-edge", i)))
		ve[i] = bld.AddEdge(geom.NewLineSegment(bottom[i].Point(), top[i].Point()), bottom[i], top[i], topo.NewLineage(topo.Tok(feat, "side-edge", i)))
	}
	return be, te, ve
}

// capEdge builds one cap edge for a segment at height z along the normal: a straight line for a line
// segment, a geom.Arc3d through the segment's swept midpoint for an arc. A degenerate arc (collinear
// three points) falls back to a line so the loop still closes.
func capEdge(bld *topo.Builder, s analyticSeg, vStart, vEnd *topo.Vertex, plane sketch.Plane, z float64, normal math.Vector3, lineage topo.Lineage) *topo.Edge {
	if !s.isArc {
		return bld.AddEdge(geom.NewLineSegment(vStart.Point(), vEnd.Point()), vStart, vEnd, lineage)
	}
	mid := plane.ToModel(s.mid).TranslateBy(normal.Scale(z))
	arc, err := geom.Arc3dByThreePoints(vStart.Point(), mid, vEnd.Point())
	if err != nil {
		return bld.AddEdge(geom.NewLineSegment(vStart.Point(), vEnd.Point()), vStart, vEnd, lineage)
	}
	return bld.AddEdge(arc, vStart, vEnd, lineage)
}

// addAnalyticCaps builds the two planar caps (start = down, end = up), mirroring addCaps: the top cap
// walks the top edges forward, the bottom cap walks the bottom edges reversed so it faces outward-down.
func addAnalyticCaps(bld *topo.Builder, bottom, top []*topo.Vertex, be, te []*topo.Edge, normal math.Vector3, feat string) {
	n := len(bottom)
	bottomPlane, _ := geom.NewPlane(bottom[0].Point(), normal.Negate())
	topPlane, _ := geom.NewPlane(top[0].Point(), normal)
	bottomLoop := make([]topo.Use, n)
	topLoop := make([]topo.Use, n)
	for i := range n {
		bottomLoop[i] = topo.Rev(be[n-1-i])
		topLoop[i] = topo.Fwd(te[i])
	}
	bld.AddFace(bottomPlane, topo.NewLineage(topo.Tok(feat, "start-cap", 0)), topo.OuterLoop(bottomLoop...))
	bld.AddFace(topPlane, topo.NewLineage(topo.Tok(feat, "end-cap", 0)), topo.OuterLoop(topLoop...))
}

// addAnalyticSides builds one side face per segment through the same corner loop the faceted prism
// uses: a planar wall for a line, a true partial-cylinder wall for an arc.
func addAnalyticSides(bld *topo.Builder, segs []analyticSeg, bottom, top []*topo.Vertex, be, te, ve []*topo.Edge, plane sketch.Plane, sp span, normal math.Vector3, feat string) {
	n := len(segs)
	for i, s := range segs {
		j := (i + 1) % n
		loop := topo.OuterLoop(topo.Fwd(be[i]), topo.Fwd(ve[j]), topo.Rev(te[i]), topo.Rev(ve[i]))
		lineage := topo.NewLineage(topo.Tok(feat, "side", i))
		if s.isArc {
			addArcSideFace(bld, s, plane, sp, normal, loop, lineage)
			continue
		}
		bld.AddFace(sideSurface(bottom[i].Point(), bottom[j].Point(), top[i].Point(), 1), lineage, loop)
	}
}

// addArcSideFace builds the partial-cylinder wall for an arc segment: a geom.Cylinder about the arc's
// axis (the plane normal through its centre), bounded by the segment's arc cap edges and vertical
// seams. The face is added natural-side-out (AddFace) when the region's outward normal agrees with the
// cylinder's radial normal — a convex boundary arc, centre inside the region — and reversed for a
// concave one. A degenerate cylinder falls back to a chord plane so the shell still closes.
func addArcSideFace(bld *topo.Builder, s analyticSeg, plane sketch.Plane, sp span, normal math.Vector3, loop topo.LoopSpec, lineage topo.Lineage) {
	center := plane.ToModel(s.center).TranslateBy(normal.Scale(sp.near))
	radius := float64(s.center.DistanceTo(s.a))
	cyl, err := geom.NewCylinderWithRef(center, normal, plane.XAxis().AsVector(), radius)
	if err != nil {
		a := plane.ToModel(s.a).TranslateBy(normal.Scale(sp.near))
		b := plane.ToModel(s.b).TranslateBy(normal.Scale(sp.near))
		t := plane.ToModel(s.a).TranslateBy(normal.Scale(sp.far))
		bld.AddFace(sideSurface(a, b, t, 1), lineage, loop)
		return
	}
	if arcFaceOutwardRadial(s) {
		bld.AddFace(cyl, lineage, loop)
		return
	}
	bld.AddReversedFace(cyl, lineage, loop)
}

// arcFaceOutwardRadial reports whether the region's outward normal at the arc points along the
// cylinder's radial-outward normal (away from the centre). For a CCW loop the interior is on the LEFT
// of travel, so outward is on the RIGHT (the tangent — the chord for a circular arc's midpoint —
// rotated −90°); it agrees with the radial-from-centre direction for a convex boundary arc.
func arcFaceOutwardRadial(s analyticSeg) bool {
	tangent := s.a.VectorTo(s.b)              // chord ∥ tangent at the arc midpoint
	outward := math.V2(tangent.Y, -tangent.X) // right of CCW travel = region exterior
	radial := s.center.VectorTo(s.mid)        // cylinder radial-outward at the midpoint
	return float64(outward.X*radial.X+outward.Y*radial.Y) > 0
}
