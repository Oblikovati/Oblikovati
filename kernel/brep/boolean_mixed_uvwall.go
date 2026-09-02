// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// uv×wall pairing for the mixed per-face boolean (ADR-0058, #2247/#3460). A ruled wall (cylinder or cone
// side) sectioned by a planar face yields either RULING lines — which boolean_mixed_wall.go clips to the
// face's polygon and mirrors as straight segments — or a CONIC (circle/ellipse). A conic cannot be
// mirrored onto a face in the `planar` bucket at all: that bucket's imprint currency is [][2]math.Point3.
// So the receiving face is moved to the `uv` bucket, whose imprint currency is []geom.Curve3, and the
// pair is imprinted here: ONE section curve handed to BOTH sides, so the two faces split on identical
// coordinates and their fragments weld exactly (the invariant wallPairImprint keeps for rulings by
// mirroring the identical segments). Scope is conservative and every decline is named: the section must
// be a CLOSED conic island lying wholly inside both trims — the face polygon and the wall's axial band —
// because a conic that CLIPS either trim would need partial arcs the two arrangements terminate
// differently. Nothing is approximated and nothing is tessellated.

// promoteConicReceivers moves every polygonal-planar face of p that a wall of other sections with a conic
// ENTERING its trim out of the `planar` bucket and into the exact-frame `uv` bucket, which carries curved
// imprints. planar/planarFull/planarHoles stay index-aligned. Only a face with no detached curved holes is
// promoted: the conic-island gate reads the face's polygon rings exactly, and a circular hole degenerates
// to its seam point in that ring (planarRings), so a holed face keeps the old named decline instead.
//
// Called after the pass-through clearance gate and BEFORE crossingFaceCandidates, so every index derived
// from the partitions is computed from the promoted buckets.
func promoteConicReceivers(p, other *facePartition) {
	if len(other.wall) == 0 {
		return
	}
	planar, full, holes := p.planar[:0:0], p.planarFull[:0:0], p.planarHoles[:0:0]
	for i := range p.planar {
		if p.planarReceivesConic(i, other) {
			p.uv = append(p.uv, p.planarFull[i])
			p.uvBox = append(p.uvBox, faceLoopBox(p.planarFull[i]))
			continue
		}
		planar, full, holes = append(planar, p.planar[i]), append(full, p.planarFull[i]), append(holes, p.planarHoles[i])
	}
	p.planar, p.planarFull, p.planarHoles = planar, full, holes
}

// planarReceivesConic reports whether planar face i must move to the uv bucket: it has no detached curved
// holes, the exact-frame chart can frame it, and some wall of other sections it with a conic entering its
// trim (the same conicTouchesTool verdict wallCurveSegments declines on).
func (p facePartition) planarReceivesConic(i int, other *facePartition) bool {
	f := p.planar[i]
	if len(p.planarHoles[i]) > 0 {
		return false
	}
	if _, ok := newPlaneFaceUV(f, geom.ResolutionForBox(faceLoopBox(f))); !ok {
		return false
	}
	return wallConicEntersFace(f, other)
}

// wallConicEntersFace reports a wall of other whose plane∩wall section is a conic that enters f's trim.
func wallConicEntersFace(f curvedFace, other *facePartition) bool {
	box := paddedFaceBox(f)
	for k, wf := range other.wall {
		if !box.Intersects(inflateBox(other.wallBox[k])) {
			continue
		}
		if wallSectionConicsTouch(f, wf) {
			return true
		}
	}
	return false
}

// wallSectionConicsTouch reports any circle/ellipse section of (f's plane, the wall) entering f's trim.
func wallSectionConicsTouch(f, wf curvedFace) bool {
	rs, ok := ruledSideBandOf(wf)
	if !ok {
		return false
	}
	curves, handled := geom.IntersectSurfacesAnalytic(facePlane(f), rs.surface, geom.ResolutionForSize(rs.size()))
	if !handled {
		return false
	}
	for _, cv := range curves {
		if _, amp, isConic := conicAxialSpan(cv, rs.axis); isConic && conicTouchesTool(cv, f, rs.axis, rs.band, amp) {
			return true
		}
	}
	return false
}

// pairUVWallImprints imprints every (exact-frame face of p, ruled wall of other) pair, appending the SAME
// section curve to the uv face's list and to the wall's list — the shared-coordinate invariant that makes
// the two sides' fragments weld. ok=false declines the boolean with a named reason (see uvWallSharedImprint).
func pairUVWallImprints(p, other *facePartition, uvImp, wallImp [][]geom.Curve3) bool {
	for i, uf := range p.uv {
		box := inflateBox(p.uvBox[i])
		for k, wf := range other.wall {
			if !box.Intersects(inflateBox(other.wallBox[k])) {
				continue
			}
			curves, ok := uvWallSharedImprint(uf, wf)
			if !ok {
				return false
			}
			uvImp[i] = append(uvImp[i], curves...)
			wallImp[k] = append(wallImp[k], curves...)
		}
	}
	return true
}

// uvWallSharedImprint is the exact shared imprint of one (uv face, ruled wall) pair: the plane∩ruled
// section curves, kept when they are closed conic islands inside both trims. ok=false declines — an
// unhandled surface pair, a section that clips either trim, or a boundary edge whose crossings cannot
// be decided in closed form. A conic-framed receiver is no longer among them: conicEdgeCrossings
// meets an arc boundary with the conic×conic substitution (#3503).
func uvWallSharedImprint(uf, wf curvedFace) ([]geom.Curve3, bool) {
	rs, ok := ruledSideBandOf(wf)
	if !ok {
		return nil, false
	}
	curves, handled := geom.IntersectSurfacesAnalytic(facePlane(uf), rs.surface, geom.ResolutionForSize(rs.size()))
	if !handled {
		return nil, false
	}
	return collectWallIslands(curves, uf, rs)
}

// collectWallIslands keeps the section curves that are imprints (closed islands in both trims) and drops
// the ones clear of the pair; ok=false when any curve clips a trim.
func collectWallIslands(curves []geom.Curve3, uf curvedFace, rs ruledSide) ([]geom.Curve3, bool) {
	var out []geom.Curve3
	for _, cv := range curves {
		island, ok, clipped := wallSectionIsland(cv, uf, rs)
		if !ok {
			return nil, false
		}
		if island {
			out = append(out, clipped)
		}
	}
	return out, true
}

// wallSectionIsland decides one plane∩wall section curve. island=true: a closed conic wholly inside the
// face polygon AND strictly inside the wall band — a genuine shared imprint. island=false, ok=true: the
// curve is clear of the face or of the band, so this pair has no imprint. ok=false: it clips a trim (a
// partial arc the two arrangements would terminate differently), or it is not a conic at all.
func wallSectionIsland(cv geom.Curve3, uf curvedFace, rs ruledSide) (island, ok bool, clipped geom.Curve3) {
	center, amp, isConic := conicAxialSpan(cv, rs.axis)
	if !isConic {
		return false, false, nil
	}
	inFace, exact := conicIslandInFace(cv, uf)
	if !exact {
		// The section CROSSES this face's trim rather than sitting inside it. Clip it HERE, once,
		// to the span between its outermost crossings, and hand that one bounded arc to both
		// sides. That is what makes the corners shared: each side would otherwise clip the
		// unbounded curve in its own chart — the face against its polygon, the wall against its
		// neighbouring sections — and arrive at the same corner by two routes, leaving a T-junction
		// the stitch cannot weld (#3459).
		arc, clipped := clipSectionToFace(cv, uf)
		if !clipped {
			return false, false, nil
		}
		return true, true, arc
	}
	if !inFace {
		return false, true, nil
	}
	inside, clear := conicBandPlacement(center, amp, rs)
	return inside, inside || clear, cv
}

// conicAxialSpan returns the section conic's centre and its axial half-amplitude about that centre — zero
// for a circle (its plane is perpendicular to the wall axis). isConic=false for any other curve kind.
func conicAxialSpan(cv geom.Curve3, axis math.Vector3) (center math.Point3, amp float64, isConic bool) {
	cf, ok := geom.AsConic(cv)
	if !ok {
		return math.Point3{}, 0, false
	}
	return cf.Center, cf.AxialAmplitude(axis), true
}

// conicBandPlacement classifies a conic's axial span against the wall band: inside it strictly (a clean
// island that splits the band in two), or clear of it (no contact). Neither means the conic straddles a
// rim, which the caller declines.
func conicBandPlacement(center math.Point3, amp float64, rs ruledSide) (inside, clear bool) {
	v := bandV(center, rs.axis, rs.band)
	lo, hi := v-amp, v+amp
	inside = lo > rs.band.vMin+facePairCullPad && hi < rs.band.vMax-facePairCullPad
	return inside, !spansOverlap(lo, hi, rs.band.vMin, rs.band.vMax, facePairCullPad)
}

// conicIslandInFace reports whether a closed conic lies WHOLLY inside an all-straight planar face's trim:
// no exact crossing with any boundary edge, and one curve point inside the polygon. exact=false declines —
// the conic crosses (or grazes) the boundary, so it is not an island, or it is not a conic in this plane.
func conicIslandInFace(cv geom.Curve3, f curvedFace) (island, exact bool) {
	pl := facePlane(f)
	pc, ok := toPlaneConic(cv, pl)
	if !ok {
		return false, false
	}
	crosses, decided := conicCrossesFaceBoundary(pc, f)
	if !decided || crosses {
		return false, false
	}
	return pointInFace2D(to2D(pl, cv.PointAt(0)), f), true
}

// conicCrossesFaceBoundary reports an exact crossing of, or a grazing tangency to, the conic on any
// boundary edge of the face — in closed form, never by sampling. ok=false when an edge carries a
// curve with no conic form, which the caller must treat as undecided rather than as "clear".
//
// It walks the face's EDGES rather than its ring points. Those agree while every edge is straight,
// and they part company the moment one is an arc: a ring walk would chord it, which is exact for
// neither the crossing count nor the tangency (#3503).
func conicCrossesFaceBoundary(pc planeConic, f curvedFace) (crosses, ok bool) {
	pl := facePlane(f)
	res := geom.ResolutionForBox(faceLoopBox(f))
	for _, l := range f.loops {
		for _, e := range l.edges {
			hits, tangent, got := conicEdgeCrossings(pc, e, pl, res)
			if !got {
				return false, false
			}
			if tangent || hits > 0 {
				return true, true
			}
		}
	}
	return false, true
}

// faceLoopBox is a planar face's exact loop-point bounding box with NO cull pad — the uv bucket's box
// convention (partitionFaces takes it from the topo face's range box, which is unpadded too).
func faceLoopBox(f curvedFace) math.Box {
	box := math.EmptyBox()
	for _, ring := range planarRings(f) {
		for _, p := range ring {
			box = box.ExtendPoint(p)
		}
	}
	return box
}
