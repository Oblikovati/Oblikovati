// SPDX-License-Identifier: GPL-2.0-only

package ops

// runoutSet is the fully-built runout rebuild for one filleted edge (the S1-shaped double-
// interference runout, ADR-5): the host planes whose boss footprint is re-cut so it no longer
// protrudes (keyed by body-face ID), plus the newly generated faces (two surviving cylinder
// wings outside the freed span + the three corner-blend patches that fill it). Mirrors
// obstacleSet (fillet_obstacle_faces.go) so the assembly path treats both rebuilds uniformly.
type runoutSet struct {
	replace map[uint64]filletFace
	extra   []filletFace
}

// collectRunouts runs runout detection+tiling for every filleted edge not already handled by the
// mid-span obstacle rebuild, folding the results into the same lookup the face loop consumes: a
// face-ID -> rebuilt-face map, per-edge extra faces, and the handled-edge set whose default
// cylinderFace is suppressed. It is the runout sibling of collectObstacles (fillet_obstacle_faces.go).
func collectRunouts(fils []edgeFillet, res Resolution,
	obHandled map[uint64]bool) (map[uint64]filletFace, map[uint64][]filletFace, map[uint64]bool) {
	replace := map[uint64]filletFace{}
	extra := map[uint64][]filletFace{}
	handled := map[uint64]bool{}
	for i := range fils {
		if obHandled[fils[i].edge.ID()] {
			continue // the obstacle rebuild already owns this edge
		}
		set, ok := runoutFacesFor(fils[i], res)
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

// runoutFacesFor detects the double-interference runout regions on a constant-radius straight fillet
// edge and, when present, tiles each region into three certified corner-blend patches (extractRunout
// -> resolveBlend, ADR-0051) bridging the two surviving cylinder wings. ok=false leaves the whole edge
// on the existing path (honest-reject, ADR-3): a partial fill — a wing without its patches, a region
// whose loops do not all resolve — is a hole, forbidden. Only constant-radius fillets are handled.
func runoutFacesFor(ef edgeFillet, res Resolution) (runoutSet, bool) {
	if ef.varying {
		return runoutSet{}, false
	}
	regions := detectRunoutRegions(ef, res)
	set := runoutSet{replace: map[uint64]filletFace{}}
	fired := false
	for _, region := range regions {
		if len(region.imprints) < 2 {
			continue // a single, uncoupled imprint is not this milestone's tiling target
		}
		if !appendRegionFaces(&set, region, ef, res) {
			return runoutSet{}, false // honest-reject the WHOLE edge (no partial fill)
		}
		fired = true
	}
	return set, fired
}

// appendRegionFaces tiles one region into its three patches and two surviving wings and folds them
// into set. ok=false when the tiler declines or any loop fails to resolve — the caller then
// honest-rejects the whole edge.
func appendRegionFaces(set *runoutSet, region runoutRegion, ef edgeFillet, res Resolution) bool {
	loops, ok := extractRunout(region, ef, res)
	if !ok {
		return false
	}
	parent := filletEdgeProvenance(ef.edge)
	for _, loop := range loops {
		patch, ok := resolveBlend(loop, res)
		if !ok {
			return false
		}
		set.extra = append(set.extra, patchToFilletFace(patch, parent))
	}
	set.extra = append(set.extra, runoutWings(ef, region)...)
	return true
}

// runoutWings builds the two surviving cylinder wings flanking the freed span: the left wing runs
// from corner c0 to the region's low cut station, the right wing from the high cut station to corner
// c1. Each reuses buildWingFace's winding logic, and its cut cross-section arc is shared (by value)
// with the flank patches' fillet quarter-circle rail so the two weld with no T-junction.
func runoutWings(ef edgeFillet, region runoutRegion) []filletFace {
	left := buildWingFace(ef, wingCutAtSpine(ef, region.loEdge), true)
	right := buildWingFace(ef, wingCutAtSpine(ef, region.hiEdge), false)
	return []filletFace{left, right}
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
