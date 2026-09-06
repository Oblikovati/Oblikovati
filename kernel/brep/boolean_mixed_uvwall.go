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
	if len(other.wall) == 0 && len(other.sphere) == 0 && len(other.torus) == 0 {
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

// planarReceivesConic reports whether planar face i must move to the uv bucket: the exact-frame chart
// can frame it — with its curved holes, which are frame edges there like any other conic — and some
// wall of other sections it with a conic entering its trim (the same conicTouchesTool verdict
// wallCurveSegments declines on).
//
// A face carrying a detached curved hole was refused here, and the polygonal bucket it stayed in then
// declined the conic entering it: a plate with one bore could not take a second, and the second bore
// was rebuilt from the faceted engine's provenance instead (ADR-0061 stage 4). The demotion route
// (selectFacesDetached) already frames a holed face in this chart when an imprint MEETS its hole; a
// conic that merely enters the face is the same chart with less to solve.
func (p facePartition) planarReceivesConic(i int, other *facePartition) bool {
	f := p.planarFull[i]
	if _, ok := newPlaneFaceUV(f, geom.ResolutionForBox(faceLoopBox(f))); !ok {
		return false
	}
	return wallConicEntersFace(f, other) || closedSurfaceSectionEntersFace(f, other)
}

// closedSurfaceSectionEntersFace reports a sphere or torus of other whose plane section enters f's
// trim. The receiver must move to the exact-frame bucket for the same reason a wall's conic does: the
// closed surface carries the exact section, and a sampled polyline on the planar side would not weld to
// it (ADR-0061 stage 3).
//
// It asks whether the section MEETS the trim, not whether it sits wholly inside. The island question is
// the wrong one here for the commonest cut there is: a sphere intersected with a box is sectioned by a
// box face in a circle that leaves through that face's own edge, so the receiver never promoted, and the
// pairing then declined the whole boolean because a section entered a face it had left in the polygonal
// bucket (ADR-0061 stage 4). It is the same verdict wallConicEntersFace already takes.
func closedSurfaceSectionEntersFace(f curvedFace, other *facePartition) bool {
	box := paddedFaceBox(f)
	faces, boxes := other.closedSurfaces()
	for k, sf := range faces {
		if !box.Intersects(inflateBox(boxes[k])) {
			continue
		}
		curves, handled := geom.IntersectSurfacesAnalytic(facePlane(f), sf.surface, closedSurfaceRes(sf))
		if !handled {
			continue
		}
		for _, cv := range curves {
			if sectionMeetsFace(cv, f) {
				return true
			}
		}
	}
	return false
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
	rs, ok := ruledFaceOf(wf)
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
	rs, ok := ruledFaceOf(wf)
	if !ok {
		return nil, false
	}
	curves, handled := geom.IntersectSurfacesAnalytic(facePlane(uf), rs.surface, geom.ResolutionForSize(rs.size()))
	if !handled {
		return nil, false
	}
	return collectWallIslands(curves, uf, wf, rs)
}

// collectWallIslands keeps the section curves that are imprints (closed islands in both trims) and drops
// the ones clear of the pair; ok=false when any curve clips a trim. A section lying in the plane of one
// of the wall's OWN edges — a plate's underside meeting the rim of the boss beneath it — is a boundary
// contact, not an imprint, and contributes nothing (ADR-0060).
func collectWallIslands(curves []geom.Curve3, uf, wf curvedFace, rs ruledSide) ([]geom.Curve3, bool) {
	var out []geom.Curve3
	for _, cv := range curves {
		if sectionOnWallEdge(cv, wf) {
			continue
		}
		pieces, ok := wallSectionIsland(cv, uf, wf, rs)
		if !ok {
			// A section wallSectionIsland cannot classify is only a decline when it TOUCHES the pair.
			// A stub cap that ends ON a fat cylinder's AXIS sections that wall in two straight RULINGS,
			// three units clear of the cap's own rim: "not a conic" refused a whole partial penetration
			// over a section with no contact in it at all. The order matters — the clear test is asked
			// SECOND, so a section the island rule does carry keeps carrying it (ADR-0061 stage 4).
			if !conicEntersTrimInBand(cv, uf, rs.axis, rs.band) {
				continue
			}
			return nil, false
		}
		out = append(out, pieces...)
	}
	return out, true
}

// wallSectionIsland decides one plane∩wall section curve, returning the imprint pieces it contributes.
// A closed conic wholly inside the face polygon AND strictly inside the wall band contributes itself; a
// curve CROSSING the face's trim contributes the runs of it that lie inside (clipSectionToFace); a
// curve clear of either contributes nothing; and a curve that leaves through the WALL's own boundary
// contributes the runs inside that (clipSectionToWall). ok=false when it is not a conic, or when a run
// cannot be bounded.
//
// The last of those used to be a decline, and it is the commonest cut there is: a plane that wedges a
// corner off a cylinder sections it in an ellipse that leaves through the top rim. Bounding it here —
// once, for both sides — is the same rule clipSectionToFace already follows for the planar half
// (ADR-0062).
func wallSectionIsland(cv geom.Curve3, uf, wf curvedFace, rs ruledSide) ([]geom.Curve3, bool) {
	center, amp, isConic := conicAxialSpan(cv, rs.axis)
	if !isConic {
		return nil, false
	}
	inFace, exact := conicIslandInFace(cv, uf)
	if !exact {
		// The section CROSSES this face's trim rather than sitting inside it. Clip it HERE, once,
		// to the span between its outermost crossings, and hand that one bounded arc to both
		// sides. That is what makes the corners shared: each side would otherwise clip the
		// unbounded curve in its own chart — the face against its polygon, the wall against its
		// neighbouring sections — and arrive at the same corner by two routes, leaving a T-junction
		// the stitch cannot weld (#3459).
		return clipSectionToFace(cv, uf)
	}
	if !inFace {
		return nil, true
	}
	inside, clear := conicBandPlacement(center, amp, rs)
	if inside {
		return []geom.Curve3{cv}, true
	}
	if clear {
		return nil, true
	}
	return clipSectionToWall(cv, wf)
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
	return bandPlacement(v-amp, v+amp, rs.band)
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
	return faceContainsExact(f, cv.PointAt(0)), true
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

// sectionOnWallEdge reports a section curve coincident with one of the wall's edges on its surface.
func sectionOnWallEdge(cv geom.Curve3, wf curvedFace) bool {
	for _, l := range wf.loops {
		for _, e := range l.edges {
			if _, coincident := geom.SectionCrossingCandidates(wf.surface, e.curve, cv); coincident {
				return true
			}
		}
	}
	return false
}
