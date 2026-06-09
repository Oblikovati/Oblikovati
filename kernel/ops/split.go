// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
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
