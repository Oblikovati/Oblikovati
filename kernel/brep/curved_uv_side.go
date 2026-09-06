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
	// uPeriodic reports whether u wraps (every ruled/torus side: u is an azimuth, u=0≡2π). A bounded plane
	// (planeUV) is NOT periodic in u — u is a real world distance — so the boundary welder must NOT fold u≈2π
	// onto u=0, which would weld a genuine face vertex to the origin ruling (#1591).
	uPeriodic() bool
	// emitRun re-emits one run of recovered boundary edges (all on one analytic curve) as a single loopEdge.
	emitRun(run []recoveredEdge) (loopEdge, bool)
	// wrapsAllU reports whether the kept region is non-empty at every azimuth (gates the rim-orientation flip).
	wrapsAllU() bool
	// multiFace reports whether the kept region may be DISCONNECTED, so the boundary loops must be grouped
	// into separate faces by containment (groupLoopFaces). Only the general curved∩curved cut needs this
	// (a fat cone's wall punched by a rod yields two disjoint lens caps); a plane half-space always leaves
	// one connected region, so the ruled/torus half-space paths return false and emit a single face (#1403).
	multiFace() bool
	// wrappingSolidFaces emits the kept region directly as one curvedFace per connected band when it WRAPS
	// the whole azimuth (the cut/join OUTSIDE/tunnel wall) — a tube the ordinary contractible-outer emission
	// mis-files. ok=false for every other case (half-space, torus, the non-wrapping intersect), so they fall
	// through to the standard (u,v) emission below (Oblikovati#1476).
	//
	// It returns the section arcs alongside the faces, exactly as the standard emission does: a
	// wrapping band's imprint sub-arcs are as real as a patch's, and a caller that assembles no lid
	// simply ignores them.
	wrappingSolidFaces(kept []Face2D, segs []uvSeg, surface geom.Surface, f curvedFace) ([]curvedFace, []loopEdge, bool)
	// orientLoops applies the surface's winding convention to the ordered boundary loops, returning the face
	// loops, the section (cut) arcs that bound the planar lid (reversed into it), and whether the kept face is
	// outerless — a closed-surface face whose loops are all holes (the genus-1 torus complement).
	orientLoops(loops []emittedLoop, wrapping bool) (faceLoops []curvedLoop, lid []loopEdge, outerless bool)
	// finalizeLoops drops or re-marks degenerate loops (the ruled apex-pole rim; a torus hole vs outer loop).
	finalizeLoops(loops []curvedLoop) []curvedLoop
	// seamOrigin is the surface parameter the arrangement's (u,v) origin sits at — where placeSeams put
	// the artificial seams. The chart records its contours in the SURFACE's own parameters, so it is what
	// the seam-relative trace is offset by (ADR-0063).
	seamOrigin() math.Point2
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
	// A solid-membership side that WRAPS the whole azimuth (the cut/join OUTSIDE/tunnel wall) is a tube the
	// contractible-outer emission below mis-files; emit it directly as one face per connected band (#1476).
	if faces, lid, ok := c.wrappingSolidFaces(kept, segs, surface, f); ok {
		return dropDegenerateLoops(faces, f), lid, nil
	}
	// The same cells traced with the seams UNFOLDED: closed contours, one set per connected component
	// (ADR-0063). Traced from the same cells so the chart and the loops cannot disagree.
	charts := chartsOfComponents(c, kept)
	var faces []curvedFace
	var lid []loopEdge
	// A kept region can be DISCONNECTED — the two lens caps a rod punches in a fat cone's wall, or the two
	// caps a ball keeps when a rod's shoulder takes a belt out of it — and each component is a face. Its
	// loops are then filed by (u,v) containment WITHIN the component, which is the only place containment
	// means anything: two loops in different components need not contain one another at all, and grouping
	// the whole set by containment merged a ball's two surviving caps into one face (#1403, ADR-0061
	// stage 4).
	for _, comp := range keptComponents(kept, c.uPeriodic(), c.vPeriodic()) {
		compFaces, compLid, ok := componentPatchFaces(c, comp, segs, surface, f, charts)
		if !ok {
			return nil, nil, ErrUnsupportedHalfSpace
		}
		faces, lid = append(faces, compFaces...), append(lid, compLid...)
	}
	return dropDegenerateLoops(faces, f), lid, nil
}

// componentPatchFaces emits ONE connected component of the kept region as its patches: the component's
// boundary loops filed by (u,v) containment, each group a face with the surface's orientation convention
// applied. ok=false when a boundary run cannot be re-emitted as an exact edge.
func componentPatchFaces(c uvSide, comp []Face2D, segs []uvSeg, surface geom.Surface, f curvedFace,
	charts []keptChart) ([]curvedFace, []loopEdge, bool) {
	loops := dropArtificialLoops(chainLoops(keptBoundaryEdges(comp, c.uPeriodic(), c.vPeriodic())), segs)
	var faces []curvedFace
	var lid []loopEdge
	for _, group := range groupLoopFaces(c.multiFace(), c.wrapsAllU(), loops) {
		emitted, ok := emitKeptLoops(c, group, segs)
		if !ok {
			return nil, nil, false
		}
		faceLoops, faceLid, outerless := c.orientLoops(emitted, c.wrapsAllU())
		faces = append(faces, curvedFace{
			surface: surface, reversed: f.reversed, lineage: f.lineage,
			loops: c.finalizeLoops(faceLoops), outerless: outerless,
			chart: chartForGroup(charts, group),
		})
		lid = append(lid, faceLid...)
	}
	return faces, lid, true
}

// dropDegenerateLoops applies the degenerate-edge rule to every face a trim produced, whichever emission
// built it. It is ONE rule about the faces — a zero-length straight edge bounds nothing, and left in it
// has a single use and the body reads as open — and it lives here because every chart's WRAPPING
// emission is its own (ruledFaceUV assembles bands, loopFrame assembles components), so a copy inside
// each of them is a copy that will be missed (ADR-0061 stage 4).
func dropDegenerateLoops(faces []curvedFace, f curvedFace) []curvedFace {
	res := geom.ResolutionForBox(faceLoopBox(f))
	out := make([]curvedFace, 0, len(faces))
	for _, cf := range faces {
		cf.loops = dropDegenerateEdges(cf.loops, res)
		out = append(out, cf)
	}
	return out
}

// dropArtificialLoops removes boundary loops made entirely of artificial seam edges — a loop that bounds
// nothing real, because the surface is closed (or degenerate) across every edge of it.
//
// A torus's genus-1 complement is the case it was written for: the kept cell has the whole parameter
// rectangle as its outer loop with the cut as a hole, and that rectangle is all seam (#1406). A SPHERE's
// complement is the same shape with a different artificial boundary — the two POLE segments, which
// poleSegments already says bound no geometry and weld to nothing — and it was excluded by a
// v-periodicity gate that named the torus rather than the property. A ball with a coaxial rod cut out of
// it came back as two boundary-less faces the trim then dropped, so the difference lost its sphere
// entirely (ADR-0061 stage 4). For a ruled side the seam edges cancel pairwise, so no all-seam loop
// survives and this stays the no-op it always was.
func dropArtificialLoops(loops [][]dedge, segs []uvSeg) [][]dedge {
	ix := newUVSegIndex(segs)
	out := loops[:0]
	for _, lp := range loops {
		if !loopAllSeam(lp, ix) {
			out = append(out, lp)
		}
	}
	return out
}

// loopAllSeam reports whether every dedge of a loop recovers to an artificial seam segment.
func loopAllSeam(loop []dedge, ix *uvSegIndex) bool {
	for _, d := range loop {
		re, ok := recoverEdge(d, ix)
		if !ok || re.kind != segSeam {
			return false
		}
	}
	return len(loop) > 0
}
