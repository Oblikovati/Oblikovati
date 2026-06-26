// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// uvSide abstracts the periodic surface a (u,v)-arrangement trim runs on, so the one arrangement pipeline
// serves every analytic family (Oblikovati/Oblikovati#1406). The ruled sides (cylinder/cone, ruledUV) are
// singly periodic — u wraps, v is the bounded axial band between two rim circles. A torus (torusUV) is
// doubly periodic — both u and v wrap — and has no rim circles, only the spiric section as its boundary.
// trimByImprint and the cross-surface helpers (emitKeptLoops, emitLoopEdges, meanEdgeV) work through this
// interface; the surface-agnostic arrangement core (Arrange, keptCells, chainLoops, the seam welder) is
// plain free functions either way.
//
// The boundary kept deliberately small: the surface contributes the (u,v) projection (paramOf), the seam
// placement and segment assembly (chooseSeamU/assembleSegments — where the ruled v-band clip and rim frame
// differ most from the torus v-seam fold), the re-emission of one boundary run to an exact analytic edge
// (emitRun), and the orientation/finalisation conventions. Everything else is shared.
type uvSide interface {
	// paramOf inverts a 3D point on the surface to its (u,v) parameters (seam-relative u, the v the
	// surface's natural second parameter — axial distance for a ruled side, tube angle for a torus).
	paramOf(p math.Point3) math.Point2
	// placeSeams moves the arrangement's artificial seam(s) clear of the imprint and stores them: a ruled
	// side places only the azimuth seam; a torus places both the azimuth (u) and tube (v) seams. Subsequent
	// paramOf/assembleSegments/emitRun report parameters relative to the placed seams.
	placeSeams(imprint []geom.Curve3)
	// assembleSegments samples the imprint into the tagged (u,v) segment set the arrangement subdivides,
	// adding the surface's own frame (ruled: rim circles + seam; torus: the folded u/v seams, no rims).
	assembleSegments(imprint []geom.Curve3) []uvSeg
	// vPeriodic reports whether v wraps (a torus): the boundary welder then folds the v-seam too, and an
	// all-seam (artificial-frame) boundary loop is dropped — a closed surface has no real boundary there.
	vPeriodic() bool
	// emitRun re-emits one run of recovered boundary edges (all on one analytic curve) as a single loopEdge.
	emitRun(run []recoveredEdge) (loopEdge, bool)
	// wrapsAllU reports whether the kept region is non-empty at every azimuth (gates the rim-orientation flip).
	wrapsAllU() bool
	// orientLoops applies the surface's winding convention to the ordered boundary loops, returning the face
	// loops, the section (cut) arcs that bound the planar lid (reversed into it), and whether the kept face is
	// outerless — a closed-surface face whose loops are all holes (the genus-1 torus complement).
	orientLoops(loops []emittedLoop, wrapping bool) (faceLoops []curvedLoop, lid []loopEdge, outerless bool)
	// finalizeLoops drops or re-marks degenerate loops (the ruled apex-pole rim; a torus hole vs outer loop).
	finalizeLoops(loops []curvedLoop) []curvedLoop
}

// trimByImprint is the general (u,v)-arrangement side split, surface-agnostic via uvSide: it moves the seam
// clear of the imprint, assembles the tagged segment set, subdivides it, classifies each cell by the
// material predicate, then re-emits the kept region's boundary as exact analytic edges and applies the
// surface's orientation convention. A plane cut passes its section conic + the half-space predicate; a
// general curved∩curved cut passes its projected imprint + membership test. It replaces the per-surface
// analytic split families — ruled and torus alike route through it (#1405, generalised #1406).
//
// materialOf is a builder (not the predicate) so the predicate binds the seam-shifted frame: it is invoked
// only after setSeamU has run, and the caller's closure reads the shifted receiver live.
func trimByImprint(c uvSide, f curvedFace, surface geom.Surface, imprint []geom.Curve3, materialOf func() materialPredicate) ([]curvedFace, []loopEdge, error) {
	c.placeSeams(imprint) // move the artificial seam(s) clear of the imprint before arranging
	segs := c.assembleSegments(imprint)
	kept := keptCells(arrangeBand(segs), materialOf())
	if len(kept) == 0 {
		return nil, nil, nil // the whole side is on the dropped side
	}
	loops := dropArtificialLoops(c, chainLoops(keptBoundaryEdges(kept, c.vPeriodic())), segs)
	emitted, ok := emitKeptLoops(c, loops, segs)
	if !ok {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	faceLoops, lid, outerless := c.orientLoops(emitted, c.wrapsAllU())
	faceLoops = c.finalizeLoops(faceLoops)
	kf := curvedFace{surface: surface, reversed: f.reversed, lineage: f.lineage, loops: faceLoops, outerless: outerless}
	return []curvedFace{kf}, lid, nil
}

// dropArtificialLoops removes boundary loops made entirely of artificial seam edges. On a v-periodic closed
// surface (a torus) the genus-1 complement's kept cell has the whole parameter rectangle as its outer loop
// with the cut as a hole; that rectangle is all seam edges and bounds nothing real (the surface is closed
// there), so it is dropped, leaving the cut alone as the face's hole (#1406). For a non-periodic ruled side
// the seam edges cancel pairwise instead, so no all-seam loop survives and this is a no-op.
func dropArtificialLoops(c uvSide, loops [][]dedge, segs []uvSeg) [][]dedge {
	if !c.vPeriodic() {
		return loops
	}
	out := loops[:0]
	for _, lp := range loops {
		if !loopAllSeam(lp, segs) {
			out = append(out, lp)
		}
	}
	return out
}

// loopAllSeam reports whether every dedge of a loop recovers to an artificial seam segment.
func loopAllSeam(loop []dedge, segs []uvSeg) bool {
	for _, d := range loop {
		re, ok := recoverEdge(d, segs)
		if !ok || re.kind != segSeam {
			return false
		}
	}
	return len(loop) > 0
}
