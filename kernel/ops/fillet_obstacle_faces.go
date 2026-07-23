// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// obstacleRimSamples is the per-rim discretization of the obstacle hole. Each segment carries its
// exact per-segment ellipse arc, so the sample COUNT sets only the weld granularity (where the
// notch/patch/tube split the rim), never the traced shape — 64 keeps the split vertices dense
// enough that the notch's absorbed arc and the tube's rebuilt rim weld point-for-point.
const obstacleRimSamples = 64

// obstacleDetection is a confirmed mid-span obstacle: the fillet's planar host face, the two other
// faces the rebuild touches (the fillet's wall face; the obstacle wall behind the hole rim), the
// hole rim as a sampled filletLoop, the two boundary crossings and their 3D nodes, and the host
// plane's flat/back projector pair (the plane's own orthonormal frame, its exact inverse).
type obstacleDetection struct {
	host          *topo.Face
	filletWall    *topo.Face // the fillet's OTHER planar face (the wall the patch is G1-tangent to)
	obstacleWall  *topo.Face // the face behind the hole rim (shares holeEdge with host)
	hostIsA       bool       // true when host == ef.a (so its tangent points are the c*.ta)
	holeEdge      *topo.Edge
	holeSampled   filletLoop
	nodes         [2]crossing
	pMinus, pPlus math.Point3 // nodes[0]/nodes[1] lifted onto the host plane (back(crossing.P))
	flat          func(math.Point3) math.Point2
	back          func(math.Point2) math.Point3
}

// obstacleGeom is the fillet-cylinder cross-section geometry the wings, patch and wall split share
// by VALUE: the two wall-tangent corners A,D, and each node's section arc + arc midpoint. Computed
// once so every consumer references identical coordinates (the weld invariant, spec §3).
type obstacleGeom struct {
	wallA, wallD math.Point3 // wall-tangent points at nodes[0], nodes[1] (WingStart/WingEnd far ends)
	startArc     geom.Arc3d  // cylinder section at nodes[0], oriented wall(A)->top(pMinus)
	endArc       geom.Arc3d  // cylinder section at nodes[1], oriented wall(D)->top(pPlus)
	startMid     math.Point3 // startArc midpoint (the cylinder bulge point at nodes[0])
	endMid       math.Point3 // endArc midpoint at nodes[1]
}

// filletRebuildMaps bundles the per-face substitution/insert/spread maps filletResultFaces already
// computes, so the obstacle host's receded outer loop is produced by the SAME transformFace path as
// every other face (no duplicated pull-back logic) before the hole is notched into it.
type filletRebuildMaps struct {
	abSubst     map[*topo.Face]map[uint64]math.Point3
	endCorner   map[*topo.Face]map[uint64]corner
	edgeInserts map[*topo.Face]map[uint64][]math.Point3
	spreads     map[*topo.Face]map[uint64]facePiece
}

// obstacleSet is the fully-built obstacle rebuild for one filleted edge (ADR-4, Task 6): the faces
// that REPLACE existing body faces (the notched host plane, the split-rim obstacle wall) keyed by
// body-face ID, plus the newly generated faces (two cylinder wings + the corner-blend patch).
// wall/wallInserts carry the split points the adjacent wall face's tangent seam must be cut at
// (injected through the existing edge-insert mechanism, #695) so the wings and patch weld to it.
type obstacleSet struct {
	replace     map[uint64]filletFace
	extra       []filletFace
	wall        *topo.Face
	wallEdge    uint64
	wallInserts []math.Point3
}

// collectObstacles runs obstacle detection+rebuild for every filleted edge and folds the results
// into one lookup the face loop consumes: a face-ID -> rebuilt-face map, per-edge extra faces, and
// the set of handled edges whose default cylinderFace must be skipped. It MUTATES edgeInserts to
// record each obstacle's wall split points so the wall face's transformFace cuts its seam at A,D.
//
// The whole operation is skipped when the body carries a delicate curved survivor band the obstacle
// re-weld could corrupt (bodyHasFragileBand). The obstacle rebuild re-welds the assembled shell around
// its notch/wings/patch, adding vertices/edges that shift the weld; a torus or free-form (b-spline)
// band elsewhere in the body then loses its trim classification and tessellates over its full
// parametric domain (the S9/T3/T9 regression — a surviving torus/BSpline rim-fillet from the imported
// STEP). Plane/cylinder/cone/elliptical-cylinder survivors re-weld robustly; torus/BSpline do not.
// Such bodies defer to the Phase-2 corner engine (Option 1 scope decision, 2026-07-14).
func collectObstacles(body *topo.Body, fils []edgeFillet, res Resolution,
	maps filletRebuildMaps) (map[uint64]filletFace, map[uint64][]filletFace, map[uint64]bool) {
	replace := map[uint64]filletFace{}
	extra := map[uint64][]filletFace{}
	handled := map[uint64]bool{}
	if bodyHasFragileBand(body) {
		return replace, extra, handled
	}
	for i := range fils {
		set, ok := obstacleFacesFor(fils[i], res, maps)
		if !ok {
			continue
		}
		for id, f := range set.replace {
			replace[id] = f
		}
		extra[fils[i].edge.ID()] = set.extra
		handled[fils[i].edge.ID()] = true
		recordWallInserts(maps.edgeInserts, set)
	}
	return replace, extra, handled
}

// bodyHasFragileBand reports whether the body carries a torus or free-form (b-spline) face whose trim
// classification the obstacle re-weld could disturb. These band surfaces (surviving rim fillets) mesh
// via a loft that depends on their rim edges staying classified; plane/cylinder/cone/elliptical-
// cylinder faces re-weld without that fragility, so they do not gate the rebuild.
func bodyHasFragileBand(body *topo.Body) bool {
	for _, f := range body.Faces() {
		switch f.Geometry().(type) {
		case geom.Torus, geom.BSplineSurface:
			return true
		}
	}
	return false
}

// recordWallInserts adds an obstacle's A,D split points to the wall face's edge-insert list, so
// transformFace subdivides the wall's tangent seam there (the mechanism variable fillets use to
// weld their ruling strips, #695).
func recordWallInserts(edgeInserts map[*topo.Face]map[uint64][]math.Point3, set obstacleSet) {
	if edgeInserts[set.wall] == nil {
		edgeInserts[set.wall] = map[uint64][]math.Point3{}
	}
	edgeInserts[set.wall][set.wallEdge] = set.wallInserts
}

// obstacleFacesFor detects a mid-span obstacle on the fillet's planar host face and, when present,
// rebuilds the local topology so the fillet band is notched around the obstacle and bridged by a
// certified corner-blend patch (ADR-4). ok=false leaves the whole case on the existing path
// (honest-reject, ADR-3): a partial rebuild — a wing without its patch, a notch without a wall
// split — is a hole, forbidden. Only constant-radius fillets are handled this slice. A qualifying==2
// (dual-host) edge routes to dualObstacleRoute instead of the single-host path below it (derivation
// §3.3): single-host providers (bsplineObstacleProvider) stay untouched by that branch.
func obstacleFacesFor(ef edgeFillet, res Resolution, maps filletRebuildMaps) (obstacleSet, bool) {
	if ef.varying {
		return obstacleSet{}, false
	}
	if set, dual, ok := dualObstacleRoute(ef, res); dual {
		return set, ok
	}
	d, ok := detectObstacle(ef, res)
	if !ok {
		return obstacleSet{}, false
	}
	of, og, geomOK := buildObstacleFeature(ef, d)
	if !geomOK {
		return obstacleSet{}, false
	}
	patch, patchOK := resolveCornerBlend(CornerBlendRequest{ObstacleFeature: of, Setback: res},
		[]CornerBlendProvider{bsplineObstacleProvider{}})
	if !patchOK {
		return obstacleSet{}, false // honest-reject the WHOLE obstacle (no wings-without-patch)
	}
	return assembleObstacleSet(ef, d, og, patch, maps)
}

// dualObstacleRoute reports whether ef is a qualifying==2 (dual-host) edge and, if so, is the FINAL
// answer for it (dual=true): U4-0's assembleDualObstacleSet is a stub that always answers ok=false, so
// a dual-host edge still lands on the do-no-harm baseline — corpus byte-identical — while the real
// rebuild (notches/walls/wings/panels) lands in U4-1..U4-5 (derivation §4, #2007 Group C). dual=false
// lets the caller fall through unchanged to detectObstacle/assembleObstacleSet for qualifying 0 or 1.
func dualObstacleRoute(ef edgeFillet, res Resolution) (obstacleSet, bool, bool) {
	dets, ok := detectObstacles(ef, res)
	if !ok || len(dets) != 2 {
		return obstacleSet{}, false, false
	}
	spans := partitionUnionStations(dets, ef)
	set, ok := assembleDualObstacleSet(ef, dets, spans)
	return set, true, ok
}

// assembleObstacleSet builds every rebuilt face once detection+patch succeeded: the notched host
// plane, the two cylinder wings, the patch face, and the split-rim obstacle wall.
func assembleObstacleSet(ef edgeFillet, d obstacleDetection, og obstacleGeom,
	patch CornerBlendPatch, maps filletRebuildMaps) (obstacleSet, bool) {
	notch, ok := buildNotchedHost(d, maps)
	if !ok {
		return obstacleSet{}, false
	}
	tube, ok := buildSplitObstacleWall(d)
	if !ok {
		return obstacleSet{}, false
	}
	wings := buildObstacleWings(ef, d, og)
	patchFace := buildPatchFace(ef, d, og, patch)
	set := obstacleSet{
		replace:     map[uint64]filletFace{d.host.ID(): notch, d.obstacleWall.ID(): tube},
		extra:       append(wings, patchFace),
		wall:        d.filletWall,
		wallEdge:    ef.edge.ID(),
		wallInserts: orderedWallInserts(ef, og),
	}
	return set, true
}

// orderedWallInserts returns the two wall split points A,D ordered from the filleted edge's start
// vertex, so addEdgeInserts places them in the wall face's tangent seam in traversal order.
func orderedWallInserts(ef edgeFillet, og obstacleGeom) []math.Point3 {
	start := ef.edge.StartVertex().Point()
	if start.DistanceTo(og.wallA) > start.DistanceTo(og.wallD) {
		return []math.Point3{og.wallD, og.wallA}
	}
	return []math.Point3{og.wallA, og.wallD}
}
