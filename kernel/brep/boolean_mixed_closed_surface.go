// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The CLOSED-SURFACE buckets of the mixed per-face boolean — a sphere and a torus (ADR-0061 stage 3).
// Neither had a chart at all: both went to the pass-through bucket, which can carry a face only when it
// is provably clear of the other operand, so any tool that met one declined the whole boolean. The
// half-space cut is the only reason either had a split pipeline of its own, and charting them here is
// what lets that pipeline be deleted (ADR-0062).
//
// A plane sections a sphere in a CIRCLE and a torus in a spiric — one closed form each, no cases — and
// the section is written into both sides' imprint lists so the two faces split on identical coordinates
// and their fragments weld.

// closedSurfaceImprints plans every (closed-surface face, EXACT-FRAME face of other) pair's shared imprint, writing
// the SAME circle into both sides' lists so the two faces split on identical coordinates and their
// fragments weld. The receiving face is in the exact-frame bucket because promoteConicReceivers moved
// it there: a sampled polyline on one side and an exact circle on the other cannot weld, which is the
// same reason a wall's conic section promotes its receiver.
//
// ok=false declines the boolean when a sphere meets a face this pairing does not cover.
func closedSurfaceImprints(p, other *facePartition, otherUV [][]geom.Curve3, rec *diag.Recorder) ([][]geom.Curve3, bool) {
	faces, boxes := p.closedSurfaces()
	out := make([][]geom.Curve3, len(faces))
	for i, sf := range faces {
		box := inflateBox(boxes[i])
		if closedSurfaceUncovered(sf, box, other, rec) {
			return nil, false
		}
		curves, ok := closedSurfaceUVImprints(sf, box, other, otherUV, rec)
		if !ok {
			return nil, false
		}
		out[i] = curves
	}
	return out, true
}

// closedSurfaceUVImprints is one closed-surface face's imprints against every overlapping exact-frame
// face of other, written into both sides' lists so the two split on identical coordinates.
func closedSurfaceUVImprints(sf curvedFace, box math.Box, other *facePartition, otherUV [][]geom.Curve3, rec *diag.Recorder) ([]geom.Curve3, bool) {
	var out []geom.Curve3
	for j, uf := range other.uv {
		if !box.Intersects(inflateBox(other.uvBox[j])) {
			continue
		}
		curves, why, ok := closedSurfaceUVImprint(sf, uf)
		if !ok {
			recordSectionDecline(rec, why, sf, uf)
			return nil, false
		}
		out = append(out, curves...)
		otherUV[j] = append(otherUV[j], curves...)
	}
	return out, true
}

// closedSurfaceUncovered reports a closed-surface face meeting a face of the other operand this pairing does not
// cover. A POLYGONAL planar face is uncovered only when the sphere's section with its plane actually
// ENTERS its trim — a box overlap alone is not contact, and the bounding prism a half-space cut is has
// side walls whose boxes span the whole body while their planes miss the sphere entirely. A face the
// section does enter should have been promoted to the exact-frame bucket; one that was not (it carries
// detached holes, or its frame declined) is a genuine gap and declines.
//
// Another closed surface is carried when its crossing with this one is DECIDED (closedSurfacePairCarried);
// a pass face still declines on box overlap, its surface being one no chart frames.
func closedSurfaceUncovered(sf curvedFace, box math.Box, other *facePartition, rec *diag.Recorder) bool {
	// Each branch is a different question, and each RECORDS: all three decline the whole boolean, so a
	// silent one is #3525 again with a different gate (review round 1, finding 3).
	if of, found := planarFaceTheSectionEnters(sf, box, other); found {
		recordSectionDecline(rec, refusalf(geom.DeclineNoClosedForm,
			"this face's section enters a polygonal face's trim; that face was not promoted to a chart that can split it"), sf, of)
		return true
	}
	if of, why, carried := everyClosedSurfacePairCarried(sf, box, other); !carried {
		recordSectionDecline(rec, why, sf, of)
		return true
	}
	if of, found := passFaceOverlapping(box, other); found {
		recordSectionDecline(rec, refusalf(geom.DeclineNoClosedForm,
			"this face's box overlaps a pass-through face (%T), whose surface no chart frames", of.surface), sf, of)
		return true
	}
	return false
}

// planarFaceTheSectionEnters finds a POLYGONAL face of other whose plane cuts sf in a section that
// enters that face's own trim. Such a face should have been promoted to the exact-frame bucket; one
// that was not (it carries detached holes, or its frame declined) is a genuine gap.
func planarFaceTheSectionEnters(sf curvedFace, box math.Box, other *facePartition) (curvedFace, bool) {
	for i := range other.planar {
		if box.Intersects(paddedFaceBox(other.planar[i])) && sphereSectionEnters(sf, other.planarFull[i]) {
			return other.planarFull[i], true
		}
	}
	return curvedFace{}, false
}

// everyClosedSurfacePairCarried reports whether sf's crossing with every overlapping closed-surface
// face of other is DECIDED, and on the first that is not, returns that face and the named reason. A
// torus against a torus refuses HERE, before any pairing asks for a section, and reporting nothing is
// what made it read to the user exactly like an ill-conditioned lane: one generic "no exact curved
// path" (Oblikovati/Oblikovati#3525).
func everyClosedSurfacePairCarried(sf curvedFace, box math.Box, other *facePartition) (curvedFace, sectionRefusal, bool) {
	faces, boxes := other.closedSurfaces()
	for i, b := range boxes {
		if !box.Intersects(inflateBox(b)) {
			continue
		}
		if why, carried := closedSurfacePairCarried(sf, faces[i]); !carried {
			return faces[i], why, false
		}
	}
	return curvedFace{}, solved(), true
}

// passFaceOverlapping finds a PASS face of other whose box overlaps: its surface is one no chart
// frames, so a box overlap is as much contact as this pairing can prove and it declines.
func passFaceOverlapping(box math.Box, other *facePartition) (curvedFace, bool) {
	for i, b := range other.passBox {
		if box.Intersects(inflateBox(b)) && i < len(other.pass) {
			return other.pass[i], true
		}
	}
	return curvedFace{}, false
}

// sphereSectionEnters reports whether the plane of of cuts the sphere in a section that enters of's own
// trim — the exact contact test, not a box overlap.
//
// It asks whether the section MEETS the trim, not whether it sits wholly inside it. Asking the stronger
// question let a section that crosses the face's boundary read as "no contact": a sphere intersected
// with a box, whose section circle leaves through the box face's own edge, planned no imprint at all
// and the sphere passed through WHOLE — a valid solid of entirely the wrong shape, which is worse than
// any decline (ADR-0061 stage 4).
func sphereSectionEnters(sf, of curvedFace) bool {
	curves, _, handled := curvedImprint(facePlane(of), sf.surface, closedSurfaceRes(sf)) // a PREDICATE: see curvedImprint
	if !handled {
		return true
	}
	for _, cv := range curves {
		if sectionMeetsFace(cv, of) {
			return true
		}
	}
	return false
}

// sectionMeetsFace reports whether any point of a section lies inside a planar face's trim — contact,
// whether or not the section stays inside.
//
// Containment goes through faceContainsExact, which meets an ARC boundary by exact ray intervals. The
// polygon test reads one point per edge, and a face bounded by ONE closed circle — a rod's end cap, the
// commonest disc in CAD — then has a "polygon" of a single point that contains nothing at all: a rod
// stopping part way through a ball's shoulder had its cap section read as no contact, and the ball
// passed through whole (ADR-0061 stage 4).
func sectionMeetsFace(cv geom.Curve3, uf curvedFace) bool {
	lo, hi := cv.Domain()
	for k := 0; k <= closedSectionWalkSamples; k++ {
		if faceContainsExact(uf, cv.PointAt(lo+(hi-lo)*float64(k)/closedSectionWalkSamples)) {
			return true
		}
	}
	return false
}

// closedSurfaceUVImprint is the exact shared imprint of one (closed surface, exact-frame planar face)
// pair: the plane∩surface section — a circle on a sphere, a spiric on a torus — kept whole when it lies
// inside the planar face's trim, CLIPPED to that trim when it crosses it, and dropped when it stays
// clear.
func closedSurfaceUVImprint(sf, uf curvedFace) ([]geom.Curve3, sectionRefusal, bool) {
	curves, why, handled := curvedImprint(facePlane(uf), sf.surface, closedSurfaceRes(sf))
	if !handled {
		return nil, refusal(why), false
	}
	var out []geom.Curve3
	for _, cv := range curves {
		kept, why, ok := closedSurfaceUVCurve(cv, uf)
		if !ok {
			return nil, why, false
		}
		out = append(out, kept...)
	}
	return out, solved(), true
}

// closedSurfaceUVCurve keeps one section curve for the planar face: nothing when it is clear of the
// trim, the whole curve when it lies inside, and the CLIPPED arcs when it crosses. Clipping is the
// same rule wallSectionIsland follows, and the reason a sphere can be intersected with a box at all:
// every section circle leaves through a box face's own edge (ADR-0061 stage 4).
func closedSurfaceUVCurve(cv geom.Curve3, uf curvedFace) ([]geom.Curve3, sectionRefusal, bool) {
	if !sectionMeetsFace(cv, uf) {
		return nil, solved(), true // clear of this face: no imprint
	}
	if sectionInsideFace(cv, uf) {
		return []geom.Curve3{cv}, solved(), true
	}
	pieces, ok := clipSectionToFace(cv, uf)
	if !ok {
		// The clip's own scope, not the section's: the section crosses the trim and the clip could not
		// bound the arcs it leaves through.
		return nil, refusalf(geom.DeclineNoClosedForm,
			"the trim clip could not bound the %T section crossing this face", cv), false
	}
	return pieces, solved(), true
}

// closedSurfaceSplitFaces trims each closed-surface face by its imprints through its loop-framed chart,
// classifying kept cells by the boolean's keep table over the other operand's membership. A sphere with
// no imprint keeps the whole-face pass-through classification.
func closedSurfaceSplitFaces(p facePartition, imprints [][]geom.Curve3, other insideOracle, op Op, isB bool, rec *diag.Recorder) ([]curvedFace, bool) {
	var out []curvedFace
	faces2, _ := p.closedSurfaces()
	for i, sf := range faces2 {
		faces, ok := closedSurfaceSplitOne(sf, imprints[i], other, op, isB, rec)
		if !ok {
			return nil, false
		}
		out = append(out, faces...)
	}
	return out, true
}

// closedSurfaceSplitOne trims one closed-surface face (or classifies it whole when it has no imprints),
// through whichever of the two charts frames it.
func closedSurfaceSplitOne(sf curvedFace, imprint []geom.Curve3, other insideOracle, op Op, isB bool, rec *diag.Recorder) ([]curvedFace, bool) {
	if len(imprint) == 0 {
		return passThroughKept([]curvedFace{sf}, other, op, isB)
	}
	side, material, ok := closedSurfaceChart(sf, op, isB, other.inside)
	if !ok {
		return nil, false
	}
	faces, _, err := trimByImprint(side, sf, sf.surface, imprint, material)
	if err != nil {
		recordArrangementDecline(rec, siteClosedSurfaceTrim, err)
		return nil, false
	}
	faces = boundedTrims(faces)
	if op == Difference && isB {
		faces = reverseCurvedFaces(faces)
	}
	return faces, true
}

// boundedTrims drops a trimmed face that came back with NO boundary, unless it is the only one. A
// sphere's parameter rectangle is closed at the poles by degenerate edges, and a cell bounded by
// nothing else emits a face whose every edge finalizeLoops removes — the whole surface a second time,
// beside the cap that was actually cut. A boundary-less face is meaningful only when the trim left the
// surface whole.
func boundedTrims(faces []curvedFace) []curvedFace {
	if len(faces) <= 1 {
		return faces
	}
	out := make([]curvedFace, 0, len(faces))
	for _, f := range faces {
		if len(f.loops) > 0 {
			out = append(out, f)
		}
	}
	return out
}

// closedSurfaces is the partition's closed-surface set — spheres then toruses — with their boxes, in
// the one order the imprint and split passes both walk.
func (p facePartition) closedSurfaces() ([]curvedFace, []math.Box) {
	faces := append(append([]curvedFace(nil), p.sphere...), p.torus...)
	boxes := append(append([]math.Box(nil), p.sphereBox...), p.torusBox...)
	return faces, boxes
}

// closedSurfaceChart builds the loop-framed chart for a closed-surface face and its material predicate.
func closedSurfaceChart(sf curvedFace, op Op, isB bool, inside func(math.Point3) bool) (uvSide, func() materialPredicate, bool) {
	if s, ok := sphereFaceOf(sf); ok {
		c := newSphereFaceUV(sf, s, op, isB, inside)
		return c, sphereFaceMaterial(c), true
	}
	if t, ok := torusFaceOf(sf); ok {
		c := newTorusFaceUV(sf, t, op, isB, inside)
		return c, torusFaceMaterial(c), true
	}
	return nil, nil, false
}

// closedSurfaceRes is the resolution a closed surface's sections are taken at: its own overall size.
func closedSurfaceRes(sf curvedFace) geom.Resolution {
	return geom.ResolutionForBox(faceLoopBox(sf))
}

// sectionInsideFace reports whether a CLOSED section curve lies inside the planar face's trim, bucketed
// by the curve's REPRESENTATION rather than by a closed form that only one of them has: a conic answers
// exactly through conicIslandInFace, and any other analytic section — a torus's spiric, which is a
// quartic and no conic at all — answers by walking itself, which is exact evaluation of an exact curve.
//
// The conic form used to be the only one, so a spiric section declined the whole boolean and a torus
// could be cut only by the half-space pipeline (ADR-0062).
func sectionInsideFace(cv geom.Curve3, uf curvedFace) bool {
	if island, exact := conicIslandInFace(cv, uf); exact {
		return island
	}
	lo, hi := cv.Domain()
	for k := 0; k <= closedSectionWalkSamples; k++ {
		if !faceContainsExact(uf, cv.PointAt(lo+(hi-lo)*float64(k)/closedSectionWalkSamples)) {
			return false // it leaves the trim: not an island this pairing carries
		}
	}
	return true
}

// closedSectionWalkSamples walks a non-conic section finely enough to catch an excursion out of any trim
// a modelled tool face has.
const closedSectionWalkSamples = 96

// pairClosedSurfaceImprints imprints every (closed surface of p, closed surface of other) pair whose
// boxes overlap, writing the shared crossing into both lists. ok=false declines the boolean.
//
// Two spheres are the simplest curved-versus-curved crossing there is — they meet in the circle of
// their radical plane — and the mixed pipeline had no pairing for them at all, so a ball meeting a ball
// declined on box overlap alone (ADR-0061 stage 4).
func pairClosedSurfaceImprints(p, other *facePartition, impP, impOther [][]geom.Curve3, rec *diag.Recorder) bool {
	faces, boxes := p.closedSurfaces()
	otherFaces, otherBoxes := other.closedSurfaces()
	for i, sf := range faces {
		box := inflateBox(boxes[i])
		for k, of := range otherFaces {
			if !box.Intersects(inflateBox(otherBoxes[k])) {
				continue
			}
			curves, why, ok := closedSurfacePairImprint(sf, of)
			if !ok {
				recordSectionDecline(rec, why, sf, of)
				return false
			}
			impP[i] = append(impP[i], curves...)
			impOther[k] = append(impOther[k], curves...)
		}
	}
	return true
}

// closedSurfacePairCarried reports whether the crossing of two closed-surface faces is decided — the
// same reading the wall pairing takes, where an empty decided answer is a proof of clearness and not an
// inability.
func closedSurfacePairCarried(sf, of curvedFace) (sectionRefusal, bool) {
	_, why, ok := closedSurfacePairImprint(sf, of)
	return why, ok
}

// closedSurfacePairImprint is the exact shared imprint of two closed-surface faces, under the scope the
// closed-surface × wall pairing already takes: both faces BOUNDARY-LESS, so every crossing is inside
// both trims by construction, and every crossing CLOSED, so each side splits by even-odd containment
// alone. Two faces on ONE surface overlap in a region rather than a curve and carry no imprint; their
// shared material is settled by the ON/ON table (boolean_mixed_coincident.go).
func closedSurfacePairImprint(sf, of curvedFace) ([]geom.Curve3, sectionRefusal, bool) {
	if len(sf.loops) > 0 || len(of.loops) > 0 {
		return nil, refusalf(geom.DeclineNoClosedForm,
			"this pairing needs two boundary-less faces; they carry %d and %d loops", len(sf.loops), len(of.loops)), false
	}
	res := closedSurfaceRes(sf)
	if geom.SurfacesCoincide(sf.surface, of.surface, res) {
		return nil, solved(), true
	}
	// The closed form may SOLVE this pair and still hand back a section that does not close, which is
	// its own named refusal and not the anonymous DeclineNone a solved answer carries (#3525).
	return islandSection(sf.surface, of.surface, res)
}
