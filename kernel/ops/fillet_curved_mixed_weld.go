// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	"maps"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Mixed-sense curved-host trihedral corner WELD (corner-blend-weld Slice-1b, M8). This is the assembly
// half of the 2r-torus mixed corner engine whose geometry (solveCurvedMixedCorner, the 2r-torus patch +
// four point-identity weld arcs) is committed in fillet_curved_corner_torus.go. A trihedral vertex where
// ONE convex Cyl∧Plane fillet + TWO concave fillets (one curved cove torus arm, one planar cyl arm) meet
// on a shared curved host is NOT a sphere corner — no single ball is tangent to the boss wall at both R−r
// and R+r at once. OCCT resolves it with an analytic 2r-torus corner patch (fillet_curved_corner_torus.go),
// the curved-host lift of the planar box-corner mechanism (fillet_corner_torus.go, K9/M2/L6).
//
// This file welds that patch into a watertight solid: it trims the three incident arm faces at the
// corner arcs (a/b/c), retrims the ~5 host faces the corner touches (the two arm rails meet at a triple
// point on the two walls; the shared top plane closes around arc (d)), emits the torus patch, and
// assembles the 14-face body. The three arm faces + host retrims reuse the concave far-runout / single-
// arm grow-retrim machinery (armRunoutRail, armFarRunout, concaveArmHostRetrim, concaveCapRetrim,
// spliceCornerBiteChain); every rail is the byte-identical curve object both the arm face and its host
// neighbour read (assembleBody welds by shared points). Gated strictly to the 2r-torus mixed corner so
// it can touch NO existing green (pure-convex, N1/L9, the planar box-corner, the concave runouts N3/M4/N9).

// mixedRoleArms is the three role-classified arms of a mixed-sense curved-host corner: the convex pivot
// (an exact geom.Cylinder arm), the curved concave cove (a geom.Torus arm), and the planar concave band
// (a Plane∧Plane fillet whose rolling-ball cylinder is the line spine). Each carries its edgeFillet so
// the weld reads its edge (far runout) and its two host faces.
type mixedRoleArms struct {
	convex edgeFillet // convex Cyl∧Plane arm (armSurface = geom.Cylinder)
	cove   edgeFillet // concave cove arm (armSurface = geom.Torus, major = R_host+r)
	planar edgeFillet // planar Plane∧Plane concave arm (armSurface nil; rolling-ball cyl = ef.cyl)
}

// classifyMixedRoleArms partitions the (already role-normalized) arms at the trihedral vertex into the
// convex pivot, the curved cove torus arm, and the planar band — or ok=false when the corner is not this
// 1-convex + 1-cove-torus + 1-planar mixed config (every other valence/sense keeps the sphere path).
func classifyMixedRoleArms(rawArms []edgeFillet) (mixedRoleArms, bool) {
	if len(rawArms) != 3 {
		return mixedRoleArms{}, false // this corner is exactly a 3-arm trihedral vertex; any other valence is a different class
	}
	var out mixedRoleArms
	var seen [3]bool // [convex, cove, planar]
	for _, ef := range rawArms {
		if !assignMixedRole(&out, &seen, ef) {
			return mixedRoleArms{}, false
		}
	}
	if !seen[0] || !seen[1] || !seen[2] {
		return mixedRoleArms{}, false
	}
	return out, true
}

// assignMixedRole files one arm into its role slot on out, rejecting (ok=false) an arm of no recognised
// mixed role OR a second assignment to an already-filled role — the M8-review dup-role guard, so a corner
// with two arms of the same role (never the 1-convex + 1-cove + 1-planar M8 config) declines. seen indexes
// [convex, cove, planar].
func assignMixedRole(out *mixedRoleArms, seen *[3]bool, ef edgeFillet) bool {
	switch {
	case isCoveTorusArm(ef):
		if seen[1] {
			return false
		}
		out.cove, seen[1] = ef, true
	case isConvexCylArm(ef):
		if seen[0] {
			return false
		}
		out.convex, seen[0] = ef, true
	case isPlanarBandArm(ef):
		if seen[2] {
			return false
		}
		out.planar, seen[2] = ef, true
	default:
		return false // an arm of no recognised mixed role — not this corner
	}
	return true
}

// isCoveTorusArm reports a concave cove arm: an exact geom.Torus arm surface built by the concave curved-
// rim path (armConcave).
func isCoveTorusArm(ef edgeFillet) bool {
	_, ok := ef.armSurface.(geom.Torus)
	return ok && ef.armConcave
}

// isConvexCylArm reports the convex pivot arm: an exact geom.Cylinder arm surface on a convex Cyl∧Plane
// rim (not the concave line arm, not the planar band whose armSurface was nil before normalization).
func isConvexCylArm(ef edgeFillet) bool {
	_, ok := ef.armSurface.(geom.Cylinder)
	return ok && !ef.armConcave && !ef.flip
}

// isPlanarBandArm reports the planar concave band: a Plane∧Plane concave fillet (flip) whose two host
// faces are both planes (its arm surface was nil before cornerArms normalized it to ef.cyl).
func isPlanarBandArm(ef edgeFillet) bool {
	if !ef.flip || ef.armConcave {
		return false
	}
	_, aPlaneOK := ef.a.Geometry().(geom.Plane)
	_, bPlaneOK := ef.b.Geometry().(geom.Plane)
	return aPlaneOK && bPlaneOK
}

// buildCurvedMixedArms assembles the solver input curvedMixedArms from the role-classified arms: the
// convex ball-centre cylinder, the cove torus + its boss host cylinder, the planar band spine cylinder,
// and the shared top plane both concave bands contact (with its material-outward normal). ok=false when a
// required host is missing (the cove arm has no cylinder host, or the two concave bands share no plane).
func buildCurvedMixedArms(roles mixedRoleArms) (curvedMixedArms, bool) {
	convexCyl, ok := roles.convex.armSurface.(geom.Cylinder)
	if !ok {
		return curvedMixedArms{}, false
	}
	coveTor, ok := roles.cove.armSurface.(geom.Torus)
	if !ok {
		return curvedMixedArms{}, false
	}
	boss, ok := cylinderHostOf(roles.cove)
	if !ok {
		return curvedMixedArms{}, false // the cove arm rolls on no cylinder wall
	}
	topFace, ok := sharedPlaneHost(roles.cove, roles.planar)
	if !ok {
		return curvedMixedArms{}, false // the two concave bands share no host plane
	}
	topPl := topFace.Geometry().(geom.Plane)
	topOut, err := math.UnitVector3FromVector(outwardFaceNormalAt(topFace, faceCentroid(topFace)))
	if err != nil {
		return curvedMixedArms{}, false
	}
	return curvedMixedArms{
		convex: convexCyl, cove: coveTor, boss: boss, planar: roles.planar.cyl,
		top: topPl, topOut: topOut.AsVector(),
	}, true
}

// cylinderHostOf returns the cylinder host face's surface among an arm's two hosts (the boss wall for the
// cove arm), or ok=false when neither host is a cylinder.
func cylinderHostOf(ef edgeFillet) (geom.Cylinder, bool) {
	for _, f := range [2]*topo.Face{ef.a, ef.b} {
		if cyl, ok := f.Geometry().(geom.Cylinder); ok {
			return cyl, true
		}
	}
	return geom.Cylinder{}, false
}

// sharedPlaneHost returns the plane host face both arms touch (the top plane both concave bands contact),
// or ok=false when they share none.
func sharedPlaneHost(x, y edgeFillet) (*topo.Face, bool) {
	for _, fx := range [2]*topo.Face{x.a, x.b} {
		if _, ok := fx.Geometry().(geom.Plane); !ok {
			continue
		}
		if fx == y.a || fx == y.b {
			return fx, true
		}
	}
	return nil, false
}

// curvedMixedCornerBody is trihedralCornerBody's mixed-sense curved-host branch: it welds the 2r-torus
// corner into a watertight solid when the corner classifies as the 1-convex + cove-torus + planar mixed
// config AND solveCurvedMixedCorner accepts (the R=2r pivot gate). took=false leaves the corner to the
// sphere-coupled path untouched (do-no-harm); took=true with a non-empty reason floors the whole op with
// that diagnostic (never a partial body).
func curvedMixedCornerBody(body *topo.Body, arms []edgeFillet, res Resolution) (*topo.Body, string, bool) {
	roles, ok := classifyMixedRoleArms(arms)
	if !ok {
		return nil, "", false
	}
	cma, ok := buildCurvedMixedArms(roles)
	if !ok {
		return nil, "", false
	}
	r, ok := armTubeRadius(roles.cove.armSurface)
	if !ok {
		return nil, "", false
	}
	corner, ok := solveCurvedMixedCorner(cma, r, res)
	if !ok {
		return nil, "", false // not the analytic 2r-torus mixed corner — keep the sphere path
	}
	b, reason := assembleCurvedMixedBody(body, roles, corner, r, res)
	return b, reason, true
}

// mixedArmBundle is one incident arm's trimmed-face rails: the two host-contact rails (oriented far→near,
// on ef.a and ef.b), the corner arc oriented nearA→nearB (its near boundary, shared byte-identically with
// the corner patch), and the far runout (trim + capping). The corner-host retrim reads railA/railB back so
// the host and the arm face weld by one curve object.
type mixedArmBundle struct {
	ef        edgeFillet
	arm       geom.Surface
	railA     endSeg // on ef.a, far foot → near foot (= corner-arc endpoint on ef.a)
	railB     endSeg // on ef.b, far foot → near foot (= corner-arc endpoint on ef.b)
	cornerArc endSeg // corner arc, oriented nearA → nearB
	far       armRunout
}

// buildMixedArmBundle terminates one incident arm: it lands the two host-contact rails from the corner-arc
// feet (near) to the far-runout feet, runs the far end through the general far-runout engine (armFarRunout,
// perpendicular or oblique), and returns the rails + far runout + corner arc oriented nearA→nearB. Declines
// (reason) on any foot/rail/runout obstruction — the do-no-harm floor.
func buildMixedArmBundle(ef edgeFillet, arm geom.Surface, cornerArc geom.Arc3d, vPoint math.Point3, filleted map[uint64]bool, r float64, res Resolution) (mixedArmBundle, string) {
	tol := res.Weld() * r
	nearA, nearB, ok := assignArcFeetToHosts(ef, cornerArc, tol)
	if !ok {
		return mixedArmBundle{}, fmt.Sprintf("mixed arm: corner-arc endpoints do not land on the two hosts %T/%T", ef.a.Geometry(), ef.b.Geometry())
	}
	railA, railB, reason := mixedArmHostRails(ef, arm, nearA, nearB, vPoint, r, res)
	if reason != "" {
		return mixedArmBundle{}, reason
	}
	railA, railB, far, okF, reason := armFarRunout(ef, cornerWeld{center: vPoint, radius: r}, railA, railB, filleted, res)
	if !okF {
		return mixedArmBundle{}, "mixed arm far runout: " + reason
	}
	return mixedArmBundle{ef: ef, arm: arm, railA: railA, railB: railB, cornerArc: orientArcSeg(cornerArc, nearA, tol), far: far}, ""
}

// mixedArmHostRails lands the arm's two host contact rails oriented FAR foot → NEAR foot (= the corner-arc
// endpoint on each host): the far foot is the arm ball's tangent point on the host at the far vertex. The
// far→near orientation is what armFarRunout reads (h0.from is the far terminus). Declines on any obstruction.
func mixedArmHostRails(ef edgeFillet, arm geom.Surface, nearA, nearB, vPoint math.Point3, r float64, res Resolution) (endSeg, endSeg, string) {
	farA, farB, reason := mixedArmFarFeet(ef, arm, vPoint, r, res)
	if reason != "" {
		return endSeg{}, endSeg{}, reason
	}
	railA, okRA := armRunoutRail(ef.a, ef.edge, arm, farA, nearA, res)
	railB, okRB := armRunoutRail(ef.b, ef.edge, arm, farB, nearB, res)
	if !okRA || !okRB {
		return endSeg{}, endSeg{}, "mixed arm: a host contact rail could not be built"
	}
	return railA, railB, ""
}

// mixedArmFarFeet is the rolling ball's two contact feet at the arm's far vertex — the outer ends both
// mixedArmHostRails and the corner-weld layer's own rail builder land their rails on. Split out so the two
// share the foot solve without sharing a rail construction (the layer needs a rail form the three-point fit
// cannot express for a half-turn span). Declines when the spine is undefined there or the ball is not
// internally tangent to a host at radius r.
func mixedArmFarFeet(ef edgeFillet, arm geom.Surface, vPoint math.Point3, r float64, res Resolution) (math.Point3, math.Point3, string) {
	tol := res.Weld() * r
	mBall, ok := armBallCenter(arm, farVertexNotVid2(ef.edge, vPoint, tol))
	if !ok {
		return math.Point3{}, math.Point3{}, "mixed arm: arm spine undefined at the far vertex"
	}
	farA, okA := armRunoutFoot(ef.a, mBall, r, tol)
	farB, okB := armRunoutFoot(ef.b, mBall, r, tol)
	if !okA || !okB {
		return math.Point3{}, math.Point3{}, fmt.Sprintf("mixed arm: far ball not internally tangent to a host (a=%v b=%v)", okA, okB)
	}
	return farA, farB, ""
}

// orientArcSeg wraps the corner arc as an endSeg oriented from the ef.a endpoint (nearA) to the ef.b one —
// so the trimmed arm face's near boundary reads the arc in loop order.
func orientArcSeg(arc geom.Arc3d, nearA math.Point3, tol float64) endSeg {
	seg := endSeg{from: arc.PointAt(0), to: arc.PointAt(1), curve: arc, mid: arc.PointAt(0.5), arc: true}
	if float64(arc.PointAt(0).DistanceTo(nearA)) > tol {
		return reverseEndSegs([]endSeg{seg})[0]
	}
	return seg
}

// assignArcFeetToHosts returns the corner arc's endpoint on ef.a (nearA) and on ef.b (nearB). ok=false when
// an endpoint lands on neither host (an inconsistent solve).
func assignArcFeetToHosts(ef edgeFillet, arc geom.Arc3d, tol float64) (math.Point3, math.Point3, bool) {
	p0, p1 := arc.PointAt(0), arc.PointAt(1)
	if onHostSurface(ef.a.Geometry(), p0, tol) && onHostSurface(ef.b.Geometry(), p1, tol) {
		return p0, p1, true
	}
	if onHostSurface(ef.a.Geometry(), p1, tol) && onHostSurface(ef.b.Geometry(), p0, tol) {
		return p1, p0, true
	}
	return math.Point3{}, math.Point3{}, false
}

// farVertexNotVid2 returns the arm edge's endpoint farther from the corner point vPoint (the far cut).
func farVertexNotVid2(e *topo.Edge, vPoint math.Point3, tol float64) math.Point3 {
	if float64(e.StartVertex().Point().DistanceTo(vPoint)) <= tol {
		return e.EndVertex().Point()
	}
	return e.StartVertex().Point()
}

// assembleCurvedMixedBody welds the 2r-torus mixed corner into a watertight solid: it terminates the three
// incident arms (buildMixedArmBundle), emits the three trimmed arm faces + the torus corner patch, retrims
// the three two-arm corner hosts (their rails meet at a triple point on the two walls; the shared top plane
// closes around arc (d)), grows/recedes the far-runout caps, carries every untouched face through, and
// assembles by shared points. Any decline returns the do-no-harm floor (nil + reason).
func assembleCurvedMixedBody(body *topo.Body, roles mixedRoleArms, corner curvedMixedCorner, r float64, res Resolution) (*topo.Body, string) {
	filleted := map[uint64]bool{roles.convex.edge.ID(): true, roles.cove.edge.ID(): true, roles.planar.edge.ID(): true}
	vPoint := cornerVertexPoint(roles, res.Weld()*r)
	cove, reason := buildMixedArmBundle(roles.cove, roles.cove.armSurface, corner.arcCove, vPoint, filleted, r, res)
	if reason != "" {
		return nil, "cove arm: " + reason
	}
	convex, reason := buildMixedArmBundle(roles.convex, roles.convex.armSurface, corner.arcInner, vPoint, filleted, r, res)
	if reason != "" {
		return nil, "convex arm: " + reason
	}
	planar, reason := buildMixedArmBundle(roles.planar, roles.planar.cyl, corner.arcBandB, vPoint, filleted, r, res)
	if reason != "" {
		return nil, "planar arm: " + reason
	}
	faces := []filletFace{
		mixedArmFace(cove), mixedArmFace(convex), mixedArmFace(planar),
		mixedTorusPatchFace(corner),
	}
	hostFaces, reason := mixedHostFaces(body, roles, corner, cove, convex, planar, vPoint, res)
	if reason != "" {
		return nil, reason
	}
	return assembleBody(append(faces, hostFaces...)), ""
}

// mixedArmFace emits one trimmed arm face: the analytic arm surface bounded by [railA (far→near), corner
// arc (nearA→nearB), railB reversed (nearB→far), far trim reversed (far→far)] — the corner arc is the same
// curve object the torus patch reads; the two host rails are the same objects the host retrims read.
func mixedArmFace(mb mixedArmBundle) filletFace {
	trim := mb.far.trim
	loop := []endSeg{mb.railA, mb.cornerArc}
	loop = append(loop, reverseEndSegs([]endSeg{mb.railB})...)
	loop = append(loop, reverseEndSegs([]endSeg{trim})...)
	return filletFace{surface: mb.arm, loops: []filletLoop{loopFromSegs(loop)}, parent: filletEdgeProvenance(mb.ef.edge)}
}

// mixedTorusPatchFace emits the 2r-torus corner patch bounded by the four weld arcs (a cove, b inner, c
// bandB, d top) chained into a closed ring — each arc byte-identical to the neighbour arm/host it welds to.
func mixedTorusPatchFace(c curvedMixedCorner) filletFace {
	segs := []endSeg{
		{from: c.arcCove.PointAt(0), to: c.arcCove.PointAt(1), curve: c.arcCove, mid: c.arcCove.PointAt(0.5), arc: true},
		{from: c.arcInner.PointAt(0), to: c.arcInner.PointAt(1), curve: c.arcInner, mid: c.arcInner.PointAt(0.5), arc: true},
		{from: c.arcBandB.PointAt(0), to: c.arcBandB.PointAt(1), curve: c.arcBandB, mid: c.arcBandB.PointAt(0.5), arc: true},
		{from: c.arcTop.PointAt(0), to: c.arcTop.PointAt(1), curve: c.arcTop, mid: c.arcTop.PointAt(0.5), arc: true},
	}
	ring, ok := chainEndSegs(segs, railGreatCircleTol*c.major)
	if !ok {
		ring = segs // fall back to the emitted order; assembleBody welds by point regardless
	}
	return filletFace{surface: c.torus, loops: []filletLoop{loopFromSegs(ring)}}
}

// mixedHostFaces retrims the five bitten host faces (three two-arm corner hosts + two far-runout caps) and
// carries every other body face through unchanged. A retrim decline floors honestly (do-no-harm).
func mixedHostFaces(body *topo.Body, roles mixedRoleArms, corner curvedMixedCorner, cove, convex, planar mixedArmBundle, vPoint math.Point3, res Resolution) ([]filletFace, string) {
	tol := res.Weld() * corner.r
	retrims, reason := mixedCornerHosts(roles, corner, cove, convex, planar, vPoint, tol)
	if reason != "" {
		return nil, reason
	}
	caps, reason := mixedCapFaces(roles, cove, convex, planar, vPoint, tol)
	if reason != "" {
		return nil, reason
	}
	maps.Copy(retrims, caps)
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

// mixedCornerHosts retrims the three two-arm corner hosts: the boss wall (cove.b == convex.a) and the
// radial plane (convex.b == planar.b), whose two arm rails meet at a single triple point, and the top plane
// (cove.a == planar.a), whose two rails are joined by the corner patch's top-contact arc (d). Declines
// (reason) on any host retrim obstruction — the do-no-harm floor.
func mixedCornerHosts(roles mixedRoleArms, corner curvedMixedCorner, cove, convex, planar mixedArmBundle, vPoint math.Point3, tol float64) (map[*topo.Face]filletFace, string) {
	arcD := endSeg{from: corner.arcTop.PointAt(0), to: corner.arcTop.PointAt(1), curve: corner.arcTop, mid: corner.arcTop.PointAt(0.5), arc: true}
	retrims := map[*topo.Face]filletFace{}
	var ok bool
	retrims[coveBossHost(roles)], ok = mixedCornerHostRetrim(coveBossHost(roles), cornerBite{cove.railB, roles.cove.edge}, cornerBite{convex.railA, roles.convex.edge}, nil, vPoint, tol)
	if !ok {
		return nil, "mixed corner: boss-wall host retrim declined"
	}
	retrims[roles.convex.b], ok = mixedCornerHostRetrim(roles.convex.b, cornerBite{convex.railB, roles.convex.edge}, cornerBite{planar.railB, roles.planar.edge}, nil, vPoint, tol)
	if !ok {
		return nil, "mixed corner: radial-plane host retrim declined"
	}
	retrims[roles.cove.a], ok = mixedCornerHostRetrim(roles.cove.a, cornerBite{cove.railA, roles.cove.edge}, cornerBite{planar.railA, roles.planar.edge}, []endSeg{arcD}, vPoint, tol)
	if !ok {
		return nil, "mixed corner: top-plane host retrim declined"
	}
	return retrims, ""
}

// coveBossHost returns the boss wall face — the cylinder host shared by the cove arm and the convex arm.
func coveBossHost(roles mixedRoleArms) *topo.Face {
	if _, ok := roles.cove.a.Geometry().(geom.Cylinder); ok {
		return roles.cove.a
	}
	return roles.cove.b
}

// mixedCapFaces retrims the two far-runout caps: the convex cap RECEDES (spliceCornerBite), while the two
// concave caps (cove + planar, which may share one plane) GROW around their cross-section arcs
// (growCapArc). A shared cap accumulates both concave bites in one pass.
func mixedCapFaces(roles mixedRoleArms, cove, convex, planar mixedArmBundle, vPoint math.Point3, tol float64) (map[*topo.Face]filletFace, string) {
	out := map[*topo.Face]filletFace{}
	// convex cap: recede the small far corner.
	segs := segsFromLoop(outerHostLoop(convex.far.capping))
	spliced, ok := spliceCornerBite(segs, convex.far.trim, tol)
	if !ok {
		return nil, "mixed corner: convex far-cap recede declined"
	}
	out[convex.far.capping] = capFaceFromSegs(convex.far.capping, spliced)
	// concave caps: grow around each cross-section arc, accumulating a shared cap's two bites.
	concaves := []struct {
		mb  mixedArmBundle
		far math.Point3
	}{
		{cove, farVertexNotVid2(roles.cove.edge, vPoint, tol)},
		{planar, farVertexNotVid2(roles.planar.edge, vPoint, tol)},
	}
	work := map[*topo.Face][]endSeg{}
	for _, c := range concaves {
		cap := c.mb.far.capping
		if _, seen := work[cap]; !seen {
			work[cap] = segsFromLoop(outerHostLoop(cap))
		}
		grown, ok := growCapArc(work[cap], c.mb.far.trim, c.far, tol)
		if !ok {
			return nil, "mixed corner: concave far-cap grow declined"
		}
		work[cap] = grown
	}
	for cap, segs := range work {
		out[cap] = capFaceFromSegs(cap, segs)
	}
	return out, ""
}

// capFaceFromSegs wraps a retrimmed cap's outer-loop segs back into a filletFace, carrying every inner loop
// through unchanged and preserving the cap's provenance.
func capFaceFromSegs(cap *topo.Face, segs []endSeg) filletFace {
	loops := append([]filletLoop{loopFromSegs(segs)}, innerHostLoops(cap)...)
	return filletFace{surface: cap.Geometry(), loops: loops, parent: cap.Lineage()}
}

// cornerVertexPoint returns the shared trihedral vertex point — the endpoint the three arm edges share.
func cornerVertexPoint(roles mixedRoleArms, tol float64) math.Point3 {
	cv := roles.convex.edge
	for _, p := range [2]math.Point3{cv.StartVertex().Point(), cv.EndVertex().Point()} {
		if edgeHasEndpoint(roles.cove.edge, p, tol) && edgeHasEndpoint(roles.planar.edge, p, tol) {
			return p
		}
	}
	return cv.StartVertex().Point()
}

// edgeHasEndpoint reports whether edge e has an endpoint at p within tol.
func edgeHasEndpoint(e *topo.Edge, p math.Point3, tol float64) bool {
	return float64(e.StartVertex().Point().DistanceTo(p)) <= tol || float64(e.EndVertex().Point().DistanceTo(p)) <= tol
}
