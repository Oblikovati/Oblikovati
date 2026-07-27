// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The CONSUMER of the band-imprint walk: turning the surviving region's boundary into faces.
//
// The walk (fillet_band_imprint.go) answers "which chain"; chainRetrimLoop (fillet_retrim_chain.go)
// answers "absorb this chain into that boundary". This file is the seam between them — it reads each
// boundary run off the imprint, finds the body face the run lies on, and re-trims that face's ORIGINAL
// ring across the run. The band's own loop is the whole cycle of runs.
//
// ★ IT IS ALL-OR-NOTHING, and that is the point. simple/Y2 is five wrong faces and simple/Y4 is seven,
// and they SHARE EDGES: the host plane, the slot's three walls and the wall above the slot all meet at
// vertices the imprint deletes. Re-trimming the host plane alone takes Y2's 8475 → 8450 while the band
// still claims x ∈ [0,100] at z = 85, so the shell OPENS along a 10-unit edge and the case goes
// PASS → FAIL(faulty). Any decline anywhere below therefore declines the WHOLE edge, leaving it on the
// existing path exactly as it is today.
//
// STRANGLER. A run that reproduces what the existing transform already builds is NOT adopted: the face
// is left to transformFace byte for byte (bandSameOuterLoop). So the walk can only ever move a face
// whose loop it disagrees with, and a body with no obstacle in its band never reaches this file at all.

// bandImprintSet is one filleted edge's finished rebuild: the body faces whose loops the imprint
// re-trims (keyed by face ID, replacing their default transform) and the band face itself.
type bandImprintSet struct {
	replace map[uint64]filletFace
	band    filletFace
}

// collectBandImprints runs the band∩obstacle imprint for the filleted edge when it qualifies, folding
// the result into the same face-ID → face / per-edge-extra / handled-edge lookup collectObstacles
// uses. An edge another rebuild already owns is left alone, and so is any body carrying MORE than one
// blend — a second fillet may transform one of the same faces, and this walk rebuilds a face's whole
// outer loop (a limit, not a law; see the report's "what remains").
//
// Example: simple/Y2 (a 100³ box with a 10×10 through-slot at x ∈ [90,100], z ∈ [80,90], filleted
// r = 15 along its y = 0 ∧ z = 100 edge) yields five replaced faces plus the interrupted band.
func collectBandImprints(body *topo.Body, fils []edgeFillet, maps filletRebuildMaps,
	caps map[uint64][]cornerPiece, replace map[uint64]filletFace, extra map[uint64][]filletFace,
	handled map[uint64]bool) (map[uint64]filletFace, map[uint64][]filletFace, map[uint64]bool) {
	if len(fils) != 1 || handled[fils[0].edge.ID()] {
		return replace, extra, handled
	}
	set, ok := bandImprintFacesFor(body, fils[0], maps, caps)
	if !ok {
		return replace, extra, handled
	}
	for id, f := range set.replace {
		replace[id] = f
	}
	extra[fils[0].edge.ID()] = []filletFace{set.band}
	handled[fils[0].edge.ID()] = true
	return replace, extra, handled
}

// bandImprintFacesFor solves one edge's imprint and builds every face it changes. ok=false is an
// honest decline: the blend is not of the kind the walk can imprint, no obstacle reaches its band, the
// arrangement could not be verified, or ANY of the faces would not re-trim (§ file comment).
func bandImprintFacesFor(body *topo.Body, ef edgeFillet, maps filletRebuildMaps,
	caps map[uint64][]cornerPiece) (bandImprintSet, bool) {
	if !bandImprintQualifies(ef, caps) {
		return bandImprintSet{}, false
	}
	res := ResolutionForBody(body)
	imp, ok := solveBandImprint(body, ef, res.Weld())
	if !ok {
		return bandImprintSet{}, false
	}
	segs := bandRunSegs(imp, ef)
	faces, ok := bandRunFaces(body, imp, segs)
	if !ok {
		return bandImprintSet{}, false
	}
	return bandRebuiltFaces(ef, segs, faces, maps, res)
}

// bandImprintQualifies is the kind gate: a CONSTANT-radius convex blend between two PLANAR hosts,
// capped at both ends by a plain flat section (no corner blend, miter, run-out apex, spread cap or
// far-end trim curve). Everything else keeps its existing path untouched — the walk's chart is the
// cylinder's, and each of those ends carries a boundary the chart does not draw.
func bandImprintQualifies(ef edgeFillet, caps map[uint64][]cornerPiece) bool {
	if ef.varying || ef.flip || ef.armConcave || ef.armSurface != nil || ef.armEllipticRim != nil {
		return false
	}
	if _, ok := ef.a.Geometry().(geom.Plane); !ok {
		return false
	}
	if _, ok := ef.b.Geometry().(geom.Plane); !ok {
		return false
	}
	return bandPlainEnd(ef.c0, caps) && bandPlainEnd(ef.c1, caps)
}

// bandPlainEnd reports whether a section end is the flat cap the chart's u = const side describes.
func bandPlainEnd(c corner, caps map[uint64][]cornerPiece) bool {
	if c.blend || c.miter || c.runout || c.endCurve != nil || c.endFace == nil {
		return false
	}
	if c.vertex == nil {
		return true
	}
	_, spread := caps[c.vertex.ID()]
	return !spread
}

// bandRunSegs turns every boundary run into the 3D segment the loops carry: a section ARC at constant
// u, a straight RULING at constant v. A run that is a whole side of the ideal box is built from the
// fillet's OWN solved section (the corner's tangent points and arc midpoint), so an untouched
// neighbour's edge keeps exactly the curve it has today.
func bandRunSegs(imp bandImprint, ef edgeFillet) []endSeg {
	out := make([]endSeg, len(imp.runs))
	for i, r := range imp.runs {
		out[i] = bandRunSeg(imp.chart, r, ef)
	}
	return out
}

// bandRunSeg is one run as a boundary segment, oriented from→to along the traversal.
func bandRunSeg(c bandChart, r bandRun, ef edgeFillet) endSeg {
	if !r.constU {
		return endSeg{from: c.bandBoxCorner(r.from, r.at), to: c.bandBoxCorner(r.to, r.at)}
	}
	if r.bandFullSide(c) {
		return bandWholeSectionSeg(c, r, ef)
	}
	from, to := c.bandBoxCorner(r.at, r.from), c.bandBoxCorner(r.at, r.to)
	mid := c.bandPointAt(r.at, (r.from+r.to)/2)
	arc, _ := geom.Arc3dByThreePoints(from, mid, to)
	return endSeg{from: from, to: to, curve: arc, mid: mid, arc: true}
}

// bandWholeSectionSeg is an end section the imprint does not cut, rebuilt through the corner's OWN
// recorded arc midpoint — the identical construction cornerEndSegs uses, so the face on the other side
// of that edge sees no change at all.
func bandWholeSectionSeg(c bandChart, r bandRun, ef edgeFillet) endSeg {
	end := ef.c0
	if r.at == c.uMax {
		end = ef.c1
	}
	arc, _ := geom.Arc3dByThreePoints(end.ta, end.mid, end.tb)
	seg := endSeg{from: end.ta, to: end.tb, curve: arc, mid: end.mid, arc: true}
	if r.from > r.to {
		return reversedEndSeg(seg)
	}
	return seg
}

// bandRunFaces resolves, for every run, the ONE body face that run lies on — the face whose boundary
// must absorb it. ok=false when a run lands on no face, on more than one, or when two runs land on the
// same face (a chain this slice does not assemble).
func bandRunFaces(body *topo.Body, imp bandImprint, segs []endSeg) ([]*topo.Face, bool) {
	out := make([]*topo.Face, len(segs))
	seen := map[uint64]bool{}
	for i, r := range imp.runs {
		f, ok := bandFaceUnderRun(body, bandRunMidpoint(imp.chart, r))
		if !ok || seen[f.ID()] {
			return nil, false
		}
		seen[f.ID()], out[i] = true, f
	}
	return out, true
}

// bandFaceUnderRun finds the single body face containing the run's own midpoint.
func bandFaceUnderRun(body *topo.Body, mid math.Point3) (*topo.Face, bool) {
	var hit *topo.Face
	for _, f := range body.Faces() {
		if !topo.NewFaceEvaluator(f).Contains(mid) {
			continue
		}
		if hit != nil {
			return nil, false // two faces claim the run — ambiguous, decline
		}
		hit = f
	}
	return hit, hit != nil
}

// bandRunMidpoint is the run's own chart midpoint in 3D.
func bandRunMidpoint(c bandChart, r bandRun) math.Point3 {
	if r.constU {
		return c.bandPointAt(r.at, (r.from+r.to)/2)
	}
	return c.bandPointAt((r.from+r.to)/2, r.at)
}

// bandRebuiltFaces re-trims each run's face across that run and assembles the band, declining the
// whole set if any one of them will not re-trim (the atomicity requirement).
func bandRebuiltFaces(ef edgeFillet, segs []endSeg, faces []*topo.Face,
	maps filletRebuildMaps, res geom.Resolution) (bandImprintSet, bool) {
	set := bandImprintSet{replace: map[uint64]filletFace{}, band: bandImprintBandFace(ef, segs)}
	for i, f := range faces {
		ff, changed, ok := bandRetrimmedFace(f, segs[i], maps, res)
		if !ok {
			return bandImprintSet{}, false
		}
		if changed {
			set.replace[f.ID()] = ff
		}
	}
	return set, len(set.replace) > 0
}

// bandRetrimmedFace re-trims one face's original outer ring across its run. changed=false means the
// rebuild agrees with what transformFace already produces, so the face is left alone byte for byte.
func bandRetrimmedFace(f *topo.Face, run endSeg, maps filletRebuildMaps,
	res geom.Resolution) (filletFace, bool, bool) {
	ring := originalHostSegs(f)
	if len(ring) < 3 {
		return filletFace{}, false, false
	}
	spliced, ok := chainRetrimLoop(ring, []endSeg{run}, res.Weld())
	if !ok {
		return filletFace{}, false, false
	}
	loop := loopFromSegs(spliced)
	if bandSameOuterLoop(loop, bandExistingLoop(f, maps, res.Size()), res.Weld()) {
		return filletFace{}, false, true
	}
	return filletFace{surface: f.Geometry(), loops: append([]filletLoop{loop}, innerHostLoops(f)...),
		parent: f.Lineage()}, true, true
}

// bandExistingLoop is the outer loop the default path would build for this face — what the rebuild
// must differ from before it is allowed to replace it.
func bandExistingLoop(f *topo.Face, maps filletRebuildMaps, scale float64) filletLoop {
	ff := transformFace(f, maps.abSubst[f], maps.endCorner[f], maps.edgeInserts[f], maps.spreads[f], scale)
	if len(ff.loops) == 0 {
		return filletLoop{}
	}
	return ff.loops[0]
}

// bandSameOuterLoop reports whether two loops are the same cycle of points, up to where the list
// starts. Direction is NOT normalised: the splice keeps the ring's own winding, so a reversed loop is
// a real difference.
func bandSameOuterLoop(a, b filletLoop, tol float64) bool {
	if len(a.pts) != len(b.pts) || len(a.pts) == 0 {
		return false
	}
	for off := range a.pts {
		if bandLoopMatchesFrom(a.pts, b.pts, off, tol) {
			return true
		}
	}
	return false
}

// bandLoopMatchesFrom reports whether b matches a rotated by off, point for point within tol.
func bandLoopMatchesFrom(a, b []math.Point3, off int, tol float64) bool {
	for i := range a {
		if float64(a[(i+off)%len(a)].DistanceTo(b[i])) > tol {
			return false
		}
	}
	return true
}

// bandImprintBandFace is the blend's own face: the whole cycle of runs, wound to the cylinder's
// outward normal exactly as cylinderFace winds the uncut box.
func bandImprintBandFace(ef edgeFillet, segs []endSeg) filletFace {
	out := segs
	if cylinderSegsFlipped(ef, out) != ef.flip {
		out = reverseEndSegs(out)
	}
	return filletFace{surface: ef.cyl, loops: []filletLoop{loopFromSegs(out)},
		parent: filletEdgeProvenance(ef.edge)}
}
