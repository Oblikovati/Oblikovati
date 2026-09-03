// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Flush contact on an exact-framed face (ADR-0060, #3508). Two coplanar faces interact only over their
// overlap; the polygonal split imprints each with the other's outline clipped to its own material and
// classifies the fragments by the ON/ON table (coplanarKeep). A face that carries conic loops — a cap
// whose outer rim is a circle, a plate with an analytic bore — takes the same two steps EXACTLY: the
// other's straight edges are clipped to its conic even-odd intervals, the other's whole conic edges
// enter it as islands or clipped arcs, and its cells are covered-tested by exact containment. The
// polygonal face across such a contact is promoted to the exact-frame chart first, so the conic edge it
// receives has a frame to land in.

// promoteCoplanarReceivers moves every polygonal face of p that is coplanar with, and box-overlaps, a
// conic-edged exact-frame face of other into p's uv bucket, holes and all.
func promoteCoplanarReceivers(p, other *facePartition) {
	planar, full, holes := p.planar[:0:0], p.planarFull[:0:0], p.planarHoles[:0:0]
	for i := range p.planar {
		if p.planarMeetsConicFlush(i, other) {
			p.uv = append(p.uv, p.planarFull[i])
			p.uvBox = append(p.uvBox, faceLoopBox(p.planarFull[i]))
			continue
		}
		planar, full, holes = append(planar, p.planar[i]), append(full, p.planarFull[i]), append(holes, p.planarHoles[i])
	}
	p.planar, p.planarFull, p.planarHoles = planar, full, holes
}

// planarMeetsConicFlush reports a conic-edged uv face of other coplanar with planar face i and
// overlapping its box.
func (p facePartition) planarMeetsConicFlush(i int, other *facePartition) bool {
	f := p.planarFull[i]
	box := paddedFaceBox(f)
	for k, g := range other.uv {
		if allStraightFace(g) || !box.Intersects(inflateBox(other.uvBox[k])) || !coplanar(f, g) {
			continue
		}
		if _, ok := newPlaneFaceUV(f, geom.ResolutionForBox(faceLoopBox(f))); ok {
			return true
		}
	}
	return false
}

// faceContainsExact reports whether a point of the face's plane lies in its material: the polygon
// test for an all-straight face, else even-odd over the exact conic intervals of a line through the
// point (a second direction when the first grazes an edge).
func faceContainsExact(f curvedFace, p math.Point3) bool {
	if allStraightFace(f) {
		return pointInFace2D(to2D(facePlane(f), p), f)
	}
	pl := facePlane(f)
	for _, dir := range []math.Vector3{pl.UAxis.AsVector(), pl.VAxis.AsVector(), pl.UAxis.AsVector().Add(pl.VAxis.AsVector())} {
		ivs, exact := curvedFaceLineIntervals(f, p, dir)
		if !exact {
			continue
		}
		for _, iv := range ivs {
			if iv[0] < 0 && 0 < iv[1] {
				return true
			}
		}
		return false
	}
	return false
}

// coplanarCoverExact is coplanarCover with exact containment on a conic-edged covering face.
func coplanarCoverExact(f curvedFace, ip math.Point3, others []curvedFace) (covered, sameNormal bool) {
	for _, o := range others {
		if coplanar(f, o) && faceContainsExact(o, ip) {
			return true, faceNormal(f).Dot(faceNormal(o)) > 0
		}
	}
	return false, false
}

// coplanarStraightImprints clips the other coplanar face's straight edges to target's exact material,
// dropping any piece lying on target's own boundary (a shared edge is not an imprint).
func coplanarStraightImprints(target, other curvedFace) ([][2]math.Point3, bool) {
	var out [][2]math.Point3
	for _, l := range other.loops {
		for _, e := range l.edges {
			if !geom.IsStraightCurve(e.curve) {
				continue
			}
			a, b := e.start(), e.end()
			ivs, exact := exactFaceLineIntervals(target, a, a.VectorTo(b))
			if !exact {
				return nil, false
			}
			for _, s := range lineIntervalSegments(a, a.VectorTo(b), intersectIntervals(ivs, [][2]float64{{0, 1}})) {
				if !segmentOnFaceBoundary(target, s) {
					out = append(out, s)
				}
			}
		}
	}
	return out, true
}

// coplanarConicImprints enters the other coplanar face's whole conic edges into target: an island when
// wholly inside its material, the clipped arcs when it crosses the boundary, nothing when clear.
// ok=false for a conic edge that is not a whole closed curve (an arc), which this does not model yet.
func coplanarConicImprints(target, other curvedFace) ([]geom.Curve3, bool) {
	var out []geom.Curve3
	for _, l := range other.loops {
		for _, e := range l.edges {
			if geom.IsStraightCurve(e.curve) {
				continue
			}
			if !geom.CurveIsClosed(e.curve) || !isFullDomain(e.t0, e.t1) {
				return nil, false
			}
			pieces, ok := conicPiecesInFace(e.curve, target)
			if !ok {
				return nil, false
			}
			out = append(out, pieces...)
		}
	}
	return out, true
}

// conicPiecesInFace is the part of a whole closed conic that lies in the face's material.
func conicPiecesInFace(cv geom.Curve3, f curvedFace) ([]geom.Curve3, bool) {
	inside, exact := conicIslandInFace(cv, f)
	if exact {
		if inside {
			return []geom.Curve3{cv}, true
		}
		return nil, true
	}
	return clipSectionToFace(cv, f)
}

// coplanarFaceImprints is the exact exchange of one coplanar pair as CURVES on each side: the other's
// straight edges clipped to the face plus its whole conic edges as islands or arcs.
func coplanarFaceImprints(target, other curvedFace) ([]geom.Curve3, bool) {
	segs, ok := coplanarStraightImprints(target, other)
	if !ok {
		return nil, false
	}
	out := make([]geom.Curve3, 0, len(segs))
	for _, s := range segs {
		out = append(out, geom.NewLineSegment(s[0], s[1]))
	}
	conics, ok := coplanarConicImprints(target, other)
	if !ok {
		return nil, false
	}
	return append(out, conics...), true
}

// faceInteriorPoint returns a point strictly inside a planar face's material: the midpoint of the
// first exact interval of a line through its loop box centre.
func faceInteriorPoint(f curvedFace) (math.Point3, bool) {
	pl := facePlane(f)
	c := faceLoopBox(f).Center()
	c = to3D(pl, to2D(pl, c))
	for _, dir := range []math.Vector3{pl.UAxis.AsVector(), pl.VAxis.AsVector(), pl.UAxis.AsVector().Add(pl.VAxis.AsVector())} {
		ivs, exact := exactFaceLineIntervals(f, c, dir)
		if !exact || len(ivs) == 0 {
			continue
		}
		return c.TranslateBy(dir.Scale(math.Scalar((ivs[0][0] + ivs[0][1]) / 2))), true
	}
	return math.Point3{}, false
}

// uvKeepAt is the exact-frame chart's keep test: a cell covered by a coplanar face of the other operand
// follows the ON/ON table; every other cell follows the keep table over the membership oracle.
func uvKeepAt(uf curvedFace, others []curvedFace, other insideOracle, op Op, isB bool) func(math.Point3) bool {
	return func(pt math.Point3) bool {
		covered, same := coplanarCoverExact(uf, pt, others)
		if covered {
			return coplanarKeep(op, isB, same)
		}
		return keep(op, isB, other.inside(pt))
	}
}
