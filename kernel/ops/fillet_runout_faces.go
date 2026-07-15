// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// runoutSet is the fully-built runout rebuild for one filleted edge (the S1-shaped double-
// interference runout, ADR-5): the host planes whose boss footprint is re-cut so it no longer
// protrudes AND the two boss walls whose closed footprint rim is split so each sub-arc welds to a
// neighbour (keyed by body-face ID), plus the newly generated faces (two surviving cylinder wings
// outside the freed span + the three corner-blend patches that fill it). Mirrors obstacleSet
// (fillet_obstacle_faces.go) so the assembly path treats both rebuilds uniformly.
type runoutSet struct {
	replace map[uint64]filletFace
	extra   []filletFace
}

// collectRunouts runs runout detection+tiling for every filleted edge not already handled by the
// mid-span obstacle rebuild, folding the results into the same lookup the face loop consumes: a
// face-ID -> rebuilt-face map, per-edge extra faces, and the handled-edge set whose default
// cylinderFace is suppressed. It is the runout sibling of collectObstacles (fillet_obstacle_faces.go).
// body+maps are threaded so the reconstructed host planes are produced by the SAME transformFace path
// as every other body face before the boss footprint is re-cut / dropped from them (Task 10b).
func collectRunouts(body *topo.Body, fils []edgeFillet, res Resolution,
	obHandled map[uint64]bool, maps filletRebuildMaps) (map[uint64]filletFace, map[uint64][]filletFace, map[uint64]bool) {
	replace := map[uint64]filletFace{}
	extra := map[uint64][]filletFace{}
	handled := map[uint64]bool{}
	if bodyHasFragileBand(body) {
		// A torus / free-form (b-spline) survivor band loses its trim classification when the runout
		// re-weld adds vertices/edges, and then tessellates over its full parametric domain (the S9/T3/
		// T9 / T4 regression — a surviving torus rim-fillet inflates the area). Same scope decision as
		// the mid-span obstacle path (collectObstacles, ADR-4): defer such bodies to the corner engine.
		return replace, extra, handled
	}
	for i := range fils {
		if obHandled[fils[i].edge.ID()] {
			continue // the obstacle rebuild already owns this edge
		}
		set, ok := runoutFacesFor(fils[i], res, maps)
		if !ok {
			continue
		}
		for id, f := range set.replace {
			replace[id] = f
		}
		extra[fils[i].edge.ID()] = set.extra
		handled[fils[i].edge.ID()] = true
	}
	return replace, extra, handled
}

// runoutFacesFor detects the intact-boss runout bands on a constant-radius straight fillet edge and,
// when present, tiles the interfered span into certified setback patches (extractSetbackPatches ->
// resolveBlend) that fair the trimmed fillet out to each crossing boss wall, kept INTACT — the
// dual-intact-survivor topology OCCT ships (t1-t7-oracle-forensics.md §8): every boss (cylinder, cone,
// torus, oblique-ellipse) is left byte-area-preserved, the two host planes are re-clipped single-loop,
// and buildSetbackFaces welds it all watertight. This REPLACES the old boss-SPLITTING path (whose
// green on S1/S4 was area-coincidental, not topology-faithful). ok=false leaves the whole edge on the
// existing path (honest-reject, ADR-3): a partial fill is a hole, forbidden. Constant-radius only.
func runoutFacesFor(ef edgeFillet, res Resolution, maps filletRebuildMaps) (runoutSet, bool) {
	if ef.varying {
		return runoutSet{}, false
	}
	b, ok := detectSetbackBands(ef, res)
	if !ok || !setbackBossesFaithful(b) {
		return runoutSet{}, false
	}
	loops, ok := extractSetbackPatches(b, ef, res)
	if !ok {
		return runoutSet{}, false
	}
	set := runoutSet{replace: map[uint64]filletFace{}}
	if !buildSetbackFaces(&set, ef, b, loops, res, maps) {
		return runoutSet{}, false // honest-reject the WHOLE edge (no partial fill)
	}
	return set, true
}

// setbackBossesFaithful gates the intact-boss path to the crossing-boss wall surface types it is proven
// faithful on: CYLINDER (S1) and CONE (S4). Any other wall — a SPHERE (S7), an oblique elliptical
// cylinder / SurfaceOfLinearExtrusion (T7), etc. — declines to BASELINE (do-no-harm), never a wrong
// intact fill. This is load-bearing, not defensive: a sphere boss is a periodic cap whose seam/pole the
// intact transformFace collapses to ~0 area (verified on S7: −5.5% area), and S7's do-no-harm baseline
// is already within OCCT's 1%, so deferring it there keeps the case green. Cylinder/cone re-weld
// cleanly (the same reason bodyHasFragileBand admits them); widening this whitelist needs a per-type
// closure proof like S1/S4's (a later task greens sphere/ellipse-cyl runouts).
func setbackBossesFaithful(b setbackBands) bool {
	for _, boss := range b.bosses {
		switch boss.wall.(type) {
		case geom.Cylinder, geom.Cone:
		default:
			return false
		}
	}
	return true
}

// appendRegionFaces tiles one region into its three patches and two surviving wings, reconstructs the
// two host planes and splits the two boss walls (buildRunoutHostsAndWalls), and folds them all into
// set. The host/wall faces are built FIRST so their shared arcs are sampled identically to the patch
// rails (sampleCurve3Open, ringSegSamples) and weld point-for-point. ok=false when the tiler declines,
// any loop fails to resolve, or a host/wall re-cut is malformed — the caller then honest-rejects.
func appendRegionFaces(set *runoutSet, region runoutRegion, ef edgeFillet, res Resolution, maps filletRebuildMaps) bool {
	loops, tl, ok := extractRunoutTiled(region, ef, res)
	if !ok {
		return false
	}
	if !buildRunoutHostsAndWalls(set, ef, tl, maps) {
		return false
	}
	// Wings are appended BEFORE the patches so the shared fillet-cut edge is built from the wing's
	// straight chords (a LineSegment on the cylinder), not the patch's full arc — otherwise the wing's
	// cylinder trim re-traces the whole quarter-arc over each 1/6-span sub-edge (the 56s pathology, 10b).
	set.extra = append(set.extra, runoutWings(ef, tl)...)
	parent := filletEdgeProvenance(ef.edge)
	for _, loop := range loops {
		patch, ok := resolveBlend(loop, res)
		if !ok {
			return false
		}
		set.extra = append(set.extra, patchToFilletFace(patch, parent))
	}
	return true
}

// runoutWings builds the two surviving cylinder wings flanking the freed span: the left wing runs
// from corner c0 to the low cut station cutL, the right wing from the high cut station cutR to corner
// c1. Each cut cross-section is the flank patch's arm arc (armSectionArc, the SAME curve the leftLoop/
// rightLoop tile from) sampled into ringSegSamples chords, so the wing and patch share those vertices
// and weld with no T-junction (class 1, Task 10b).
func runoutWings(ef edgeFillet, tl runoutTiling) []filletFace {
	leftArc, _ := armSectionArc(tl.cyl, tl.planeB, tl.planeA, tl.cutL)
	rightArc, _ := armSectionArc(tl.cyl, tl.planeA, tl.planeB, tl.cutR)
	leftCut, rightCut := wingCutAtSpine(ef, tl.cutL), wingCutAtSpine(ef, tl.cutR)
	left := buildWingFaceCut(ef, leftCut, true, sampledArcSegs(leftArc, leftCut.nodeTa, leftCut.nodeTb))
	right := buildWingFaceCut(ef, rightCut, false, sampledArcSegs(rightArc, rightCut.nodeTa, rightCut.nodeTb))
	return []filletFace{left, right}
}

// sampledArcSegs chops arc into ringSegSamples straight sub-chords oriented start→end, pinning the two
// ends to the wing's exact node points (start/end) while taking every interior vertex from arc.PointAt at
// i/n — the SAME points sampleCurve3Open hands the flank patch, so the two share every welded vertex. The
// count is fixed at ringSegSamples (the one ring granularity every blend boundary shares); a parameter
// would only ever receive it (unparam), and a mismatch here would open the wing↔patch weld.
func sampledArcSegs(arc geom.Curve3, start, end math.Point3) []endSeg {
	n := ringSegSamples
	lo, hi := arc.Domain()
	pts := make([]math.Point3, n+1)
	for i := 0; i <= n; i++ {
		pts[i] = arc.PointAt(lo + float64(i)/float64(n)*(hi-lo))
	}
	if pts[0].DistanceTo(start) > pts[n].DistanceTo(start) {
		pts = reversePts(pts)
	}
	pts[0], pts[n] = start, end // pin to the wing's node points (weld-identical to the arc ends)
	segs := make([]endSeg, n)
	for i := 0; i < n; i++ {
		segs[i] = endSeg{from: pts[i], to: pts[i+1]}
	}
	return segs
}

// reversePts returns pts in reverse order (a fresh slice).
func reversePts(pts []math.Point3) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}

// wingCutAtSpine resolves the fillet cross-section split points at axial station spine into a wingCut
// (the face-a/face-b tangent points and the arc midpoint), translating corner c0's constant radials
// along the straight cylinder axis — the section is translation-invariant, so c0's radials evaluated
// at spine give the exact section there (the same identity computeObstacleGeom relies on).
func wingCutAtSpine(ef edgeFillet, spine float64) wingCut {
	center := ef.cyl.Origin.TranslateBy(ef.cyl.AxisDir.AsVector().Scale(spine))
	ta := center.TranslateBy(ef.c0.cen.VectorTo(ef.c0.ta))
	tb := center.TranslateBy(ef.c0.cen.VectorTo(ef.c0.tb))
	mid := center.TranslateBy(ef.c0.cen.VectorTo(ef.c0.mid))
	return wingCut{nodeTa: ta, nodeTb: tb, mid: mid}
}
