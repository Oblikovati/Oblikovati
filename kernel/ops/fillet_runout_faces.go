// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	maps0 "maps"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// runoutSet is the fully-built runout rebuild for one filleted edge (the S1-shaped double-
// interference runout, ADR-5): replace holds ONLY the two re-clipped host-plane notches, keyed by
// body-face ID, whose boss footprint is re-cut so it no longer protrudes (reclipOuterHost/
// reclipInnerHost, fillet_setback_close.go). The two boss walls are kept INTACT — transformFace
// merely subdivides their footprint rim (maps.edgeInserts) so neighbours weld; a wall is NEVER
// split and NEVER placed in replace. extra holds the newly generated faces: the two plain cyl-R
// wings outside the freed span plus the resolved setback corner-blend patches that fill it. Mirrors
// obstacleSet (fillet_obstacle_faces.go) so the assembly path treats both rebuilds uniformly.
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
	if runoutDefersBody(body) {
		// A free-form (b-spline) survivor band loses its trim classification when the runout re-weld adds
		// vertices/edges and then tessellates over its full parametric domain (the T9 regression). It has
		// no band mesher, so defer the whole body to the corner engine (same scope decision as the obstacle
		// path). A TORUS survivor is NO LONGER deferred here: the intact-boss path never splits it, and the
		// chorded-rim torus-band tessellator (M4 Task 2) meshes it correctly — the per-boss-type gate
		// setbackBossesFaithful is the real torus gate now (M4 Task 3). See runoutDefersBody.
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
		maps0.Copy(replace, set.replace)
		extra[fils[i].edge.ID()] = set.extra
		handled[fils[i].edge.ID()] = true
	}
	return replace, extra, handled
}

// runoutDefersBody reports whether the RUNOUT rebuild must defer a whole body to the corner engine — the
// runout-path-scoped narrowing of the shared obstacle gate (bodyHasFragileBand, fillet_obstacle_faces.go,
// left UNTOUCHED). The obstacle gate defers BOTH torus and b-spline survivors; the intact-boss runout path
// keeps every boss wall INTACT (never splits a torus into a full-donut grid) and the chorded-rim torus-band
// tessellator (band_ring_chain.go, M4 Task 2) meshes a surviving torus band as a ruled strip, so a TORUS
// survivor no longer corrupts on re-weld — the per-boss-type setbackBossesFaithful whitelist is its real
// gate (M4 Task 3, greening T1/T4). Only a free-form (b-spline) survivor band still loses its trim
// classification on the runout re-weld with no band mesher to recover it (T9), so it alone defers here.
func runoutDefersBody(body *topo.Body) bool {
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.BSplineSurface); ok {
			return true
		}
	}
	return false
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
// faithful on: CYLINDER (S1), CONE (S4), the oblique ELLIPTICAL CYLINDER (T7, geom.EllipticalCylinder —
// the elementarised SurfaceOfLinearExtrusion of an ellipse whose footprint is a geom.EllipseFull), the
// TORUS (T1/T4, M4), and the SPHERE (S7, M5). Any other wall declines to BASELINE (do-no-harm), never a
// wrong intact fill. This is load-bearing, not defensive: each admitted wall re-welds cleanly and its
// footprint conic (circle / ellipse) rails exactly (fillet_setback_ellipse.go). The doubly-curved walls
// (torus, sphere) are admitted only because M4's fixes close them: the rim rebuilds as the full 360°
// σ-partition (fillet_setback_partition.go, M4 Task 1) — including the host DETOUR (M4 Task 3) so the host
// notch welds the major host arc — and the intact wall meshes as a chorded band (band_ring_chain.go, M4
// Task 2), NOT a full-domain blow-up. The sphere is the retroactive win: pre-M4 its periodic cap collapsed
// to ~0 through the split path (the reason it rode do-no-harm baseline), but the R=13 hemisphere now meshes
// intact at its 2πR²=1061.86 area and the result is watertight — an UPGRADE over the baseline, which left
// the boss footprints piercing the host planes (HolesContained=false, green only by area tolerance).
// Widening this whitelist further needs a per-type closure proof like these (TestFilletEdges_*Intact).
func setbackBossesFaithful(b setbackBands) bool {
	for _, boss := range b.bosses {
		switch boss.wall.(type) {
		case geom.Cylinder, geom.Cone, geom.EllipticalCylinder, geom.Torus, geom.Sphere:
		default:
			return false
		}
	}
	return true
}

// sampledArcSegs chops arc into ringSegSamples sub-segments oriented start→end, pinning the two ends
// to the wing's exact node points (start/end) while taking every interior vertex from arc.PointAt at
// i/n — the SAME points sampleCurve3Open hands the flank patch, so the two share every welded vertex.
// Each segment carries the arm arc RESTRICTED to its own sub-span (sampleCurveNTrimmed, the N7 rule) —
// the identical value the flank patch's loop offers for the shared edge — so the wing's own loop model
// bounds the true arc and the weld is a two-sided value agreement instead of the nil-vs-curve adoption
// this family used to ride (12 records per setback case; wing-arm-arcs-report.md). The count is fixed
// at ringSegSamples (the one ring granularity every blend boundary shares); a parameter would only
// ever receive it (unparam), and a mismatch here would open the wing↔patch weld.
func sampledArcSegs(arc geom.Curve3, start, end math.Point3) []endSeg {
	n := ringSegSamples
	lo, hi := arc.Domain()
	rev := arc.PointAt(lo).DistanceTo(start) > arc.PointAt(hi).DistanceTo(start)
	pts, curves := sampleCurveNTrimmed(arc, n, rev)
	pts = append(pts, end)
	pts[0] = start // pin to the wing's node points (weld-identical to the arc ends)
	segs := make([]endSeg, n)
	for i := range n {
		segs[i] = endSeg{from: pts[i], to: pts[i+1], curve: curves[i]}
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
