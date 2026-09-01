// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// extractObstacle turns a mid-span ObstacleFeature into the 4-sided RailLoop the general coons4 tier
// (corner_provider_coons4.go) fills. The Side order is s0=wall, s1=wingR, s2=rim, s3=wingL — the same
// mapping bsplineObstacleProvider already certifies (obstacleSides) — so resolveBlend's general path
// reproduces the (sign-corrected, Task 2) obstacle patch: the wall and both wings are G1 to their
// analytic neighbour surfaces, the base rim is a G0 crease. ok=false on a bad rail or an
// unreconstructable adjacent surface → the caller honest-rejects (ADR-3).
func extractObstacle(of *ObstacleFeature) (RailLoop, bool) {
	c0, d1, rim, wingL, ok := obstacleLoopCurves(of)
	if !ok {
		return RailLoop{}, false
	}
	wall, wingLSurf, wingRSurf, rimSurf, ok := obstacleAdjacents(of)
	if !ok {
		return RailLoop{}, false
	}
	sides := []Side{
		{Curve: c0, Adjacent: wall, Cont: G1},         // s0 wall  A->D
		{Curve: d1, Adjacent: wingRSurf, Cont: G1},    // s1 wingR D->P+
		{Curve: rim, Adjacent: rimSurf, Cont: G0},     // s2 rim   P+->P-
		{Curve: wingL, Adjacent: wingLSurf, Cont: G1}, // s3 wingL P-->A
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}

// obstacleLoopCurves returns the four rails in LOOP order A->D->P+->P-->A, reusing obstacleRails'
// already-pinned geometry. c0 (wall, A->D) and its own d1 (wingR, D->P+) are already forward; the rim
// (obstacleRails' c1, P-->P+) and wingL (obstacleRails' d0, A->P-) are REVERSED here so the chain's
// end/start points meet consecutively (RailLoop.Closed) — the exact walk obstaclePatchLoops documents
// ("A->D (c0) -> P+ (d1) -> P- (c1 reversed) -> A (d0 reversed)", corner_blend_obstacle.go). This only
// fixes the LOOP's own chain: coons4Fill re-derives each rail's feed direction from the loop's own
// corners (loopRails/pinnedRail), so the eventual fill geometry is unaffected by this reversal.
func obstacleLoopCurves(of *ObstacleFeature) (c0, d1, rim, wingL geom.BSplineCurve, ok bool) {
	rawC0, rawC1, rawD0, rawD1, ok := obstacleRails(of)
	if !ok {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, false
	}
	rim, ok1 := reverseBSplineCurve(rawC1)
	wingL, ok2 := reverseBSplineCurve(rawD0)
	if !ok1 || !ok2 {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, false
	}
	return rawC0, rawD1, rim, wingL, true
}

// obstacleAdjacents returns the four analytic neighbour surfaces the RailLoop's G1 ribbons sample
// NormalAt on: the wall plane (WallLine+WallInto), the two wing cylinders (reconstructed EXACTLY from
// WingStart/WingEnd's own cross-section arcs — wingCylinder), and the rim's host plane (of.HostPlane,
// read only for the G0 side, never sampled). ok=false when the wall plane or a wing cylinder cannot be
// reconstructed.
func obstacleAdjacents(of *ObstacleFeature) (wall, wingL, wingR, rim geom.Surface, ok bool) {
	wallPl, ok1 := obstacleWallPlane(of)
	l, ok2 := wingCylinder(of.WingStart, of.BlendAxis)
	r, ok3 := wingCylinder(of.WingEnd, of.BlendAxis)
	if !ok1 || !ok2 || !ok3 {
		return nil, nil, nil, nil, false
	}
	return wallPl, l, r, of.HostPlane, true
}

// obstacleWallPlane rebuilds the wall's plane from WallLine (its own direction is one in-plane axis)
// and WallInto (the other — already in-plane and perpendicular to the bottom rail by the detector's
// contract, see ObstacleFeature.WallInto's doc) via NewPlaneFromAxes, so no cross-product/sign
// convention is invented here. ok=false when the two axes are parallel (a degenerate WallLine/WallInto
// pair) or WallLine is nil.
func obstacleWallPlane(of *ObstacleFeature) (geom.Plane, bool) {
	if of == nil || of.WallLine == nil {
		return geom.Plane{}, false
	}
	lo, _ := of.WallLine.Domain()
	origin := of.WallLine.PointAt(lo)
	along := of.WallLine.TangentAt(lo)
	pl, err := geom.NewPlaneFromAxes(origin, along, of.WallInto)
	return pl, err == nil
}

// wingCylinder reconstructs the EXACT fillet cylinder a wing section arc lives on. section
// (WingStart/WingEnd) is always the geom.Arc3d cross-section wingSection builds off the TRUE wing
// cylinder (fillet_obstacle_rebuild.go), so its Center sits on the cylinder axis and its Radius IS the
// cylinder radius — an exact reconstruction, with no dependency on ObstacleFeature.Radius (which is
// only the G1-ribbon LENGTH, corner_blend_obstacle.go:ribbonLength, not the true cylinder radius).
// axis is BlendAxis, the detector's true fillet-cylinder axis; the arc's own RefDir pins the
// reconstructed cylinder's angle-zero to the arc's. ok=false when section is not a geom.Arc3d (an
// unexpected wing shape this task cannot handle) or axis/RefDir collapse (NewCylinderWithRef).
func wingCylinder(section geom.Curve3, axis math.Vector3) (geom.Cylinder, bool) {
	arc, isArc := section.(geom.Arc3d)
	if !isArc {
		return geom.Cylinder{}, false
	}
	cyl, err := geom.NewCylinderWithRef(arc.Center, axis, arc.RefDir.AsVector(), arc.Radius)
	return cyl, err == nil
}
