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
	// setSeamU moves the arrangement's artificial azimuth seam to absolute azimuth u (chosen clear of the
	// imprint by chooseSeamU); subsequent paramOf/assembleSegments/emitRun report u relative to it.
	setSeamU(u float64)
	// chooseSeamU returns an azimuth for the artificial seam that is clear of the imprint's u-crossings.
	chooseSeamU(imprint []geom.Curve3) float64
	// assembleSegments samples the imprint into the tagged (u,v) segment set the arrangement subdivides,
	// adding the surface's own frame (ruled: rim circles + seam; torus: the folded u/v seams, no rims).
	assembleSegments(imprint []geom.Curve3) []uvSeg
	// vPeriodic reports whether v wraps (a torus): the boundary welder then folds the v-seam too, so a
	// kept region wrapping the tube cancels its v-seam edges the way a wrapping band cancels its u-seam.
	vPeriodic() bool
	// emitRun re-emits one run of recovered boundary edges (all on one analytic curve) as a single loopEdge.
	emitRun(run []recoveredEdge) (loopEdge, bool)
	// wrapsAllU reports whether the kept region is non-empty at every azimuth (gates the rim-orientation flip).
	wrapsAllU() bool
	// orientLoops applies the surface's winding convention to the ordered boundary loops, returning the face
	// loops and the section (cut) arcs that bound the planar lid (reversed into it).
	orientLoops(loops []emittedLoop, wrapping bool) ([]curvedLoop, []loopEdge)
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
	c.setSeamU(c.chooseSeamU(imprint)) // move the artificial seam clear of the imprint before arranging
	segs := c.assembleSegments(imprint)
	kept := keptCells(arrangeBand(segs), materialOf())
	if len(kept) == 0 {
		return nil, nil, nil // the whole side is on the dropped side
	}
	emitted, ok := emitKeptLoops(c, chainLoops(keptBoundaryEdges(kept, c.vPeriodic())), segs)
	if !ok {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	faceLoops, lid := c.orientLoops(emitted, c.wrapsAllU())
	faceLoops = c.finalizeLoops(faceLoops)
	kf := curvedFace{surface: surface, reversed: f.reversed, lineage: f.lineage, loops: faceLoops}
	return []curvedFace{kf}, lid, nil
}
