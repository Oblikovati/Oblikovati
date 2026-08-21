// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// SplitSolidByPlane divides a solid into the pieces lying on each side of an infinite plane,
// returning the negative-normal side first then the positive. It intersects the body with a
// large half-space box on each side (so each piece is capped by a clean planar cross-section).
// A plane that misses the body yields a single piece (the whole body). Used by the Split feature
// and Trim Solid (keep one side).
func SplitSolidByPlane(body *topo.Body, plane geom.Plane) ([]*topo.Body, error) {
	ext := splitExtent(body, plane.Origin)
	var pieces []*topo.Body
	for _, positive := range []bool{false, true} {
		piece, err := Boolean(Intersect, body, halfSpaceBox(plane, ext, positive))
		if err != nil {
			return nil, err
		}
		if piece != nil && len(piece.Faces()) > 0 {
			pieces = append(pieces, piece)
		}
	}
	return pieces, nil
}

// SplitFacesByPlane imprints the plane's section curves onto the body's faces (the
// reference's Split Faces mode, #330): faces crossing the plane split along it, NO material
// is removed, and the volume is unchanged. Built on the body imprint (M07) against a
// half-space box whose near face lies exactly on the plane.
func SplitFacesByPlane(body *topo.Body, plane geom.Plane) (*topo.Body, error) {
	tool := halfSpaceBox(plane, splitExtent(body, plane.Origin), true)
	ra, _, err := brep.ImprintBodies(body, tool)
	if err != nil {
		return nil, fmt.Errorf("split faces: %w", err)
	}
	return ra.Body, nil
}

// SplitFacesByPath imprints a projected polyline path onto a body's faces (the split-FACE-by-path
// mode, #2068): each path segment, extruded along dir far enough to cross the body, is a planar
// strip; together they form an open sheet tool whose intersection with each face scores it — like
// SplitFacesByPlane, but along an arbitrary polyline rather than a straight plane section. No
// material is removed, so the volume is unchanged.
//
// path is the polyline in MODEL space (already projected onto the sketch plane); dir is the
// projection direction (the sketch plane normal). Straight segments only — a curved path segment
// has no planar extrusion, so the caller facets it or refuses it. Needs at least two points.
func SplitFacesByPath(body *topo.Body, path []math.Point3, dir math.Vector3) (*topo.Body, error) {
	if len(path) < 2 {
		return nil, fmt.Errorf("split faces by path: need at least 2 path points, got %d", len(path))
	}
	tool, err := pathSheetTool(body, path, dir)
	if err != nil {
		return nil, err
	}
	ra, _, err := brep.ImprintBodies(body, tool)
	if err != nil {
		return nil, fmt.Errorf("split faces by path: %w", err)
	}
	return ra.Body, nil
}

// pathSheetTool builds the open sheet whose faces are the path segments extruded ±ext along dir, so
// the sheet reaches through the body on both sides of every face it must score. One planar quad per
// segment, sharing the extruded rail edges at each interior path point.
func pathSheetTool(body *topo.Body, path []math.Point3, dir math.Vector3) (*topo.Body, error) {
	d, err := math.UnitVector3FromVector(dir)
	if err != nil {
		return nil, fmt.Errorf("split faces by path: projection direction is degenerate: %w", err)
	}
	ext := math.Scalar(splitExtent(body, path[0]))
	off := d.AsVector().Scale(ext)
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("splitpath", "sheet", 0)))
	// Two rail vertices per path point: bottom (−dir) and top (+dir).
	bot := make([]*topo.Vertex, len(path))
	top := make([]*topo.Vertex, len(path))
	for i, p := range path {
		bot[i] = bld.AddVertex(p.TranslateBy(off.Scale(-1)), topo.NewLineage(topo.Tok("splitpath", "vbot", i)))
		top[i] = bld.AddVertex(p.TranslateBy(off), topo.NewLineage(topo.Tok("splitpath", "vtop", i)))
	}
	rail := make([]*topo.Edge, len(path)) // the ±dir edge at each path point, shared by adjacent quads
	rail[0] = bld.AddEdge(geom.NewLineSegment(bot[0].Point(), top[0].Point()), bot[0], top[0],
		topo.NewLineage(topo.Tok("splitpath", "rail", 0)))
	for i := 0; i+1 < len(path); i++ {
		if err := addPathSheetQuad(bld, i, bot, top, rail); err != nil {
			return nil, err
		}
	}
	return bld.Build(), nil
}

// addPathSheetQuad adds the strip quad for segment i (path[i]→path[i+1]): its two along-path edges
// (bottom and top) plus the shared rail edge at path[i+1], wound CCW about the segment's newell
// normal so the loop and the face plane agree.
func addPathSheetQuad(bld *topo.Builder, i int, bot, top []*topo.Vertex, rail []*topo.Edge) error {
	corners := []math.Point3{bot[i].Point(), bot[i+1].Point(), top[i+1].Point(), top[i].Point()}
	n := newellUnit(corners)
	pl, err := geom.NewPlane(quadCentroid(corners), n)
	if err != nil {
		return fmt.Errorf("split faces by path: segment %d is degenerate (a zero-length step or a "+
			"step along the projection direction): %w", i, err)
	}
	bottom := bld.AddEdge(geom.NewLineSegment(bot[i].Point(), bot[i+1].Point()), bot[i], bot[i+1],
		topo.NewLineage(topo.Tok("splitpath", "bottom", i)))
	upper := bld.AddEdge(geom.NewLineSegment(top[i].Point(), top[i+1].Point()), top[i], top[i+1],
		topo.NewLineage(topo.Tok("splitpath", "top", i)))
	rail[i+1] = bld.AddEdge(geom.NewLineSegment(bot[i+1].Point(), top[i+1].Point()), bot[i+1], top[i+1],
		topo.NewLineage(topo.Tok("splitpath", "rail", i+1)))
	// Ring bot[i] → bot[i+1] → top[i+1] → top[i], reversing the shared rail edges that are stored
	// bottom→top so the loop still travels the ring direction.
	uses := []topo.Use{
		{Edge: bottom, Reversed: false},
		{Edge: rail[i+1], Reversed: false},
		{Edge: upper, Reversed: true},
		{Edge: rail[i], Reversed: true},
	}
	bld.AddFace(pl, topo.NewLineage(topo.Tok("splitpath", "face", i)), topo.OuterLoop(uses...))
	return nil
}

// splitExtent returns a half-width large enough for a cutting half-space anchored at the plane
// origin to cover the whole body — twice the farthest body-corner distance from that origin, so
// the box reaches the body even when the plane sits well outside it.
func splitExtent(b *topo.Body, origin math.Point3) float64 {
	bx := b.RangeBox()
	far := 0.0
	for _, x := range [2]math.Scalar{bx.Min.X, bx.Max.X} {
		for _, y := range [2]math.Scalar{bx.Min.Y, bx.Max.Y} {
			for _, z := range [2]math.Scalar{bx.Min.Z, bx.Max.Z} {
				if d := float64(origin.DistanceTo(math.Point3{X: x, Y: y, Z: z})); d > far {
					far = d
				}
			}
		}
	}
	return 2*far + 1
}

// halfSpaceBox builds a large oriented box covering one side of the plane: its near face lies on
// the plane and it extends `ext` along ±normal and ±ext in the plane's U/V. For the negative
// side U and V swap so the (u,v,n) frame stays right-handed — the canonical box rings are wound
// outward only for a right-handed frame.
func halfSpaceBox(plane geom.Plane, ext float64, positive bool) *topo.Body {
	u, v := plane.UAxis.AsVector(), plane.VAxis.AsVector()
	n := unit(plane.Normal())
	if !positive {
		u, v, n = v, u, n.Scale(-1)
	}
	o := plane.Origin.TranslateBy(u.Scale(math.Scalar(-ext))).TranslateBy(v.Scale(math.Scalar(-ext)))
	corner := func(x, y, z float64) math.Point3 {
		return o.TranslateBy(u.Scale(math.Scalar(2 * ext * x))).
			TranslateBy(v.Scale(math.Scalar(2 * ext * y))).
			TranslateBy(n.Scale(math.Scalar(ext * z)))
	}
	c := [8]math.Point3{
		corner(0, 0, 0), corner(1, 0, 0), corner(1, 1, 0), corner(0, 1, 0),
		corner(0, 0, 1), corner(1, 0, 1), corner(1, 1, 1), corner(0, 1, 1),
	}
	return boxFromCorners(c)
}

// boxFromCorners builds a closed box B-rep from 8 corners in the canonical 0..7 order. Each of
// the six quad faces derives its outward normal from the ring's Newell normal flipped to point
// away from the box centre, reversing the ring when needed so the loop winds CCW about that
// outward normal (plane normal and winding stay consistent — the boolean and volume both rely on
// it). (Lives here so the split has no dependency on the subd package.)
//
//nolint:funlen // assembles a box body element-by-element (8 verts, 12 edges, 6 faces); length is the geometry, not logic.
func boxFromCorners(c [8]math.Point3) *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("split", "box", 0)))
	tv := make([]*topo.Vertex, 8)
	for i := range c {
		tv[i] = bld.AddVertex(c[i], topo.NewLineage(topo.Tok("split", "vertex", i)))
	}
	edges := map[[2]int]*topo.Edge{}
	// Edges are stored canonically (min→max vertex) so the Reversed: a>b convention in the
	// loops below is consistent regardless of the order a face first traverses them; each gets a
	// distinct lineage (colliding reference keys break the downstream boolean).
	edge := func(a, b int) *topo.Edge {
		k := [2]int{a, b}
		if a > b {
			k = [2]int{b, a}
		}
		if e, ok := edges[k]; ok {
			return e
		}
		e := bld.AddEdge(geom.NewLineSegment(c[k[0]], c[k[1]]), tv[k[0]], tv[k[1]],
			topo.NewLineage(topo.Tok("split", "edge", len(edges))))
		edges[k] = e
		return e
	}
	// The six rings are pre-wound CCW about their outward normals (the canonical box layout),
	// so each face's plane uses the ring's own Newell normal directly — winding and normal agree.
	rings := [6][4]int{
		{0, 3, 2, 1}, {4, 5, 6, 7}, {0, 1, 5, 4}, {3, 7, 6, 2}, {0, 4, 7, 3}, {1, 2, 6, 5},
	}
	for fi, ring := range rings {
		pts := []math.Point3{c[ring[0]], c[ring[1]], c[ring[2]], c[ring[3]]}
		pl, _ := geom.NewPlane(quadCentroid(pts), newellUnit(pts))
		uses := make([]topo.Use, 4)
		for i := 0; i < 4; i++ {
			a, b := ring[i], ring[(i+1)%4]
			uses[i] = topo.Use{Edge: edge(a, b), Reversed: a > b}
		}
		bld.AddFace(pl, topo.NewLineage(topo.Tok("split", "face", fi)), topo.OuterLoop(uses...))
	}
	return bld.Build()
}

// quadCentroid averages corner positions.
func quadCentroid(pts []math.Point3) math.Point3 {
	var sx, sy, sz math.Scalar
	for _, p := range pts {
		sx, sy, sz = sx+p.X, sy+p.Y, sz+p.Z
	}
	n := math.Scalar(len(pts))
	return math.Point3{X: sx / n, Y: sy / n, Z: sz / n}
}

// unit returns v normalized, or v unchanged if it is degenerate.
func unit(v math.Vector3) math.Vector3 {
	if u, err := math.UnitVector3FromVector(v); err == nil {
		return u.AsVector()
	}
	return v
}
