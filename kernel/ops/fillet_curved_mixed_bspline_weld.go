// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// N4-class mixed-sense BSpline corner WELD (the sibling of fillet_curved_mixed_weld.go's 2r-torus weld).
// It welds the coons4 corner patch (fillet_curved_mixed_bspline.go) into a watertight solid. The two
// TERMINATING arms (concave-cyl, planar-band) reuse the M8 arm-bundle machinery (buildMixedArmBundle) with
// their radius-r cross-section arcs; the LATERAL convex-torus arm — which runs past the corner and borders
// the patch along the on-torus rail B→C rather than terminating at an arc — uses a bundle whose near
// boundary IS that rail (buildN4TorusLateral). The three two-arm corner hosts (vertical plane closing
// around the D→A rail, boss wall + top plane whose arm rails meet at C / B) retrim via the M8
// mixedCornerHostRetrim. Every curve is the byte-identical object the patch and its welded neighbour read
// (assembleBody welds by shared points). Gated strictly to the N4 role signature (classifyN4MixedArms), so
// it can touch NO existing green (M8's 2r-torus, pure-convex, N1/L9, the box-corner, N3/M4/N9).

// n4MixedCornerBody is trihedralCornerBody's N4-class branch: it welds the coons4 BSpline corner when the
// three arms classify as concave-cyl + convex-torus + planar-band AND solveN4Corner accepts. took=false
// leaves the corner to the sphere-coupled path untouched (do-no-harm); took=true with a non-empty reason
// floors the op with that diagnostic (never a partial body).
func n4MixedCornerBody(body *topo.Body, arms []edgeFillet, res Resolution) (*topo.Body, string, bool) {
	roles, ok := classifyN4MixedArms(arms)
	if !ok {
		return nil, "", false
	}
	r, ok := armTubeRadius(roles.torus.armSurface)
	if !ok {
		return nil, "", false
	}
	corner, ok := solveN4Corner(roles, r, res)
	if !ok {
		return nil, "", false // not the N4 BSpline corner — keep the sphere path
	}
	b, reason := assembleN4Body(body, roles, corner, r, res)
	return b, reason, true
}

// assembleN4Body welds the N4 corner into a watertight solid: the two terminating arm bundles + the lateral
// torus bundle + the coons4 patch, the three retrimmed two-arm hosts, the three grown/receded far caps, and
// every untouched face carried through. Any decline returns the do-no-harm floor (nil + reason).
func assembleN4Body(body *topo.Body, roles n4MixedArms, corner n4Corner, r float64, res Resolution) (*topo.Body, string) {
	filleted := map[uint64]bool{roles.ccyl.edge.ID(): true, roles.torus.edge.ID(): true, roles.band.edge.ID(): true}
	v := n4CornerVertex(roles, res.Weld()*r)
	ccyl, reason := buildMixedArmBundle(roles.ccyl, roles.ccyl.armSurface, corner.pts.arcCD, v, filleted, r, res)
	if reason != "" {
		return nil, "ccyl arm: " + reason
	}
	band, reason := buildMixedArmBundle(roles.band, roles.band.armSurface, corner.pts.arcAB, v, filleted, r, res)
	if reason != "" {
		return nil, "band arm: " + reason
	}
	torus, reason := buildN4TorusLateral(roles.torus, corner, v, filleted, r, res)
	if reason != "" {
		return nil, "torus arm: " + reason
	}
	faces := []filletFace{mixedArmFace(ccyl), mixedArmFace(band), mixedArmFace(torus), n4PatchFace(corner)}
	hostFaces, reason := n4HostFaces(body, roles, corner, ccyl, band, torus, v, res)
	if reason != "" {
		return nil, reason
	}
	return assembleBody(append(faces, hostFaces...)), ""
}

// buildN4TorusLateral terminates the LATERAL convex-torus arm: its NEAR boundary is the on-torus rail B→C
// (not a radius-r arc), its two host rails run from B (top plane) / C (boss wall) to the far-runout feet,
// and its far end runs through the shared far-runout engine. Declines on any foot/rail/runout obstruction.
func buildN4TorusLateral(ef edgeFillet, corner n4Corner, v math.Point3, filleted map[uint64]bool, r float64, res Resolution) (mixedArmBundle, string) {
	arm := ef.armSurface
	tol := res.Weld() * r
	nearA, nearB, ok := assignFeetToHosts(ef, corner.pts.b, corner.pts.c, tol)
	if !ok {
		return mixedArmBundle{}, fmt.Sprintf("torus rail B/C do not land on the two hosts %T/%T", ef.a.Geometry(), ef.b.Geometry())
	}
	railA, railB, reason := mixedArmHostRails(ef, arm, nearA, nearB, v, r, res)
	if reason != "" {
		return mixedArmBundle{}, reason
	}
	railA, railB, far, okF, reason := armFarRunout(ef, cornerWeld{center: v, radius: r}, railA, railB, filleted, res)
	if !okF {
		return mixedArmBundle{}, "far runout: " + reason
	}
	near := orientCurveSeg(corner.railBC, corner.pts.b, corner.pts.c, nearA, tol)
	return mixedArmBundle{ef: ef, arm: arm, railA: railA, railB: railB, cornerArc: near, far: far}, ""
}

// assignFeetToHosts returns the near boundary's endpoint on ef.a (nearA) and on ef.b (nearB) — the general-
// curve sibling of assignArcFeetToHosts (which is Arc3d-specific). ok=false when an endpoint lands on neither.
func assignFeetToHosts(ef edgeFillet, p0, p1 math.Point3, tol float64) (math.Point3, math.Point3, bool) {
	if onHostSurface(ef.a.Geometry(), p0, tol) && onHostSurface(ef.b.Geometry(), p1, tol) {
		return p0, p1, true
	}
	if onHostSurface(ef.a.Geometry(), p1, tol) && onHostSurface(ef.b.Geometry(), p0, tol) {
		return p1, p0, true
	}
	return math.Point3{}, math.Point3{}, false
}

// orientCurveSeg wraps a boundary curve (from p0 to p1) as an endSeg oriented from nearA (the ef.a foot).
func orientCurveSeg(c geom.BSplineCurve, p0, p1, nearA math.Point3, tol float64) endSeg {
	seg := endSeg{from: p0, to: p1, curve: c, mid: c.PointAt(0.5)}
	if float64(p0.DistanceTo(nearA)) > tol {
		return reverseEndSegs([]endSeg{seg})[0]
	}
	return seg
}

// n4PatchFace emits the coons4 BSpline corner patch bounded by the four sides as SINGLE curve-segs (arc
// A→B, rail B→C, arc C→D, rail D→A) — each byte-identical to the neighbour arm/host it welds to, and each
// a single seg (like the M8 torus patch) so the tessellator samples it identically on both sides.
func n4PatchFace(c n4Corner) filletFace {
	segs := []endSeg{
		{from: c.pts.a, to: c.pts.b, curve: c.pts.arcAB, mid: c.pts.arcAB.PointAt(0.5), arc: true},
		{from: c.pts.b, to: c.pts.c, curve: c.railBC, mid: c.railBC.PointAt(0.5)},
		{from: c.pts.c, to: c.pts.d, curve: c.pts.arcCD, mid: c.pts.arcCD.PointAt(0.5), arc: true},
		{from: c.pts.d, to: c.pts.a, curve: c.railDA, mid: c.railDA.PointAt(0.5)},
	}
	return filletFace{surface: c.patch.Surface, loops: []filletLoop{loopFromSegs(segs)}}
}

// n4HostFaces retrims the three two-arm corner hosts + the three far caps and carries every other face
// through unchanged.
func n4HostFaces(body *topo.Body, roles n4MixedArms, corner n4Corner, ccyl, band, torus mixedArmBundle, v math.Point3, res Resolution) ([]filletFace, string) {
	tol := res.Weld() * corner.pts.arcCD.Radius
	retrims, reason := n4CornerHosts(roles, corner, ccyl, band, torus, v, tol)
	if reason != "" {
		return nil, reason
	}
	caps, reason := n4CapFaces(roles, ccyl, band, torus, v, tol)
	if reason != "" {
		return nil, reason
	}
	for f, ff := range caps {
		retrims[f] = ff
	}
	out := make([]filletFace, 0, len(body.Faces()))
	for _, f := range body.Faces() {
		if ff, bitten := retrims[f]; bitten {
			out = append(out, ff)
			continue
		}
		out = append(out, passthroughFace(f))
	}
	return out, ""
}

// n4CornerHosts retrims the three two-arm corner hosts: the vertical plane (ccyl+band rails joined by the
// on-plane rail D→A), the boss wall (ccyl+torus rails meeting at C), and the top plane (band+torus rails
// meeting at B). Declines on any host retrim obstruction.
func n4CornerHosts(roles n4MixedArms, corner n4Corner, ccyl, band, torus mixedArmBundle, v math.Point3, tol float64) (map[*topo.Face]filletFace, string) {
	railDA := endSeg{from: corner.pts.d, to: corner.pts.a, curve: corner.railDA, mid: corner.railDA.PointAt(0.5)}
	vFace, _ := sharedPlaneHost(roles.ccyl, roles.band)
	tFace, _ := sharedPlaneHost(roles.torus, roles.band)
	bFace, okB := sharedCylFace(roles.ccyl, roles.torus)
	if vFace == nil || tFace == nil || !okB {
		return nil, "n4 corner: a shared host face is missing"
	}
	retrims := map[*topo.Face]filletFace{}
	var ok bool
	retrims[vFace], ok = mixedCornerHostRetrim(vFace, cornerBite{ccyl.railA, roles.ccyl.edge}, cornerBite{band.railA, roles.band.edge}, []endSeg{railDA}, v, tol)
	if !ok {
		return nil, "n4 corner: vertical-plane host retrim declined"
	}
	retrims[bFace], ok = mixedCornerHostRetrim(bFace, cornerBite{ccyl.railB, roles.ccyl.edge}, cornerBite{torus.railB, roles.torus.edge}, nil, v, tol)
	if !ok {
		return nil, "n4 corner: boss-wall host retrim declined"
	}
	retrims[tFace], ok = mixedCornerHostRetrim(tFace, cornerBite{band.railB, roles.band.edge}, cornerBite{torus.railA, roles.torus.edge}, nil, v, tol)
	if !ok {
		return nil, "n4 corner: top-plane host retrim declined"
	}
	return retrims, ""
}

// n4CapFaces retrims the three far-runout caps: the two concave arms (ccyl, band) GROW around their far
// cross-section arcs; the convex torus cap RECEDES.
func n4CapFaces(roles n4MixedArms, ccyl, band, torus mixedArmBundle, v math.Point3, tol float64) (map[*topo.Face]filletFace, string) {
	out := map[*topo.Face]filletFace{}
	segs := segsFromLoop(outerHostLoop(torus.far.capping))
	spliced, ok := spliceCornerBite(segs, torus.far.trim, tol)
	if !ok {
		return nil, "n4 corner: torus far-cap recede declined"
	}
	out[torus.far.capping] = capFaceFromSegs(torus.far.capping, spliced)
	concaves := []struct {
		mb  mixedArmBundle
		far math.Point3
	}{
		{ccyl, farVertexNotVid2(roles.ccyl.edge, v, tol)},
		{band, farVertexNotVid2(roles.band.edge, v, tol)},
	}
	work := map[*topo.Face][]endSeg{}
	for _, c := range concaves {
		cap := c.mb.far.capping
		if _, seen := work[cap]; !seen {
			work[cap] = segsFromLoop(outerHostLoop(cap))
		}
		grown, ok := growCapArc(work[cap], c.mb.far.trim, c.far, tol)
		if !ok {
			return nil, "n4 corner: concave far-cap grow declined"
		}
		work[cap] = grown
	}
	for cap, segs := range work {
		out[cap] = capFaceFromSegs(cap, segs)
	}
	return out, ""
}

// sharedCylFace returns the cylinder host face both arms share by identity (the boss wall).
func sharedCylFace(x, y edgeFillet) (*topo.Face, bool) {
	for _, fx := range [2]*topo.Face{x.a, x.b} {
		if _, ok := fx.Geometry().(geom.Cylinder); !ok {
			continue
		}
		if fx == y.a || fx == y.b {
			return fx, true
		}
	}
	return nil, false
}

// n4CornerVertex returns the shared trihedral vertex point the three arm edges meet at.
func n4CornerVertex(roles n4MixedArms, tol float64) math.Point3 {
	e := roles.ccyl.edge
	for _, p := range [2]math.Point3{e.StartVertex().Point(), e.EndVertex().Point()} {
		if edgeHasEndpoint(roles.torus.edge, p, tol) && edgeHasEndpoint(roles.band.edge, p, tol) {
			return p
		}
	}
	return e.StartVertex().Point()
}
