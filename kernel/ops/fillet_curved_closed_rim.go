// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Convex CLOSED-rim curved arm (Miter-A1 / OCCT blend/simple J1). A single CONVEX closed circular edge
// where a curved host of revolution (cone, cylinder, or sphere) meets a perpendicular cap plane rounds into a FULL torus
// band — no corner, no runout, no capped ends. computeCorners' 2-pick miter path is demuxed away for such
// a rim (closedRimPick: one edge counted twice, StartVertex==EndVertex), and assembleCurvedArmBody
// dispatches it here instead of flooring at "curved arms do not meet at one shared trihedral vertex".
//
// The band IS the arm's own exact geom.Torus (ef.armSurface), bounded by the two CLOSED contact circles
// (torusContactCircle) on the two receded hosts. It is assembled through the host-agnostic rim rebuild
// (rebuildWithRimFillet) — the SAME proven watertight closed band the lone-cylinder-rim path ships
// (FilletCylinderRim), so its closed_band_loft tessellation and winding are reused verbatim, not re-derived.
// Concave rims (S2/S5) are excluded by ClassifyEdgeConvexity: their physical ball is void-side (external
// tangency), so this convex arm's contact feet would land on faces that do not exist — a separate slice.

// isConvexClosedRimArm is assembleCurvedArmBody's dispatch classifier for the closed-band arm: exactly ONE
// pick carrying ONE exact TORUS arm on a CONVEX closed circular rim (StartVertex==EndVertex). A concave rim
// (S2/S5) is EdgeConcave → false (its convex arm builder is wrong-side); a cylinder/canal arm or an OPEN
// (runout) rim → false; so every other closed-rim configuration keeps flooring unchanged (do-no-harm).
func isConvexClosedRimArm(fils []edgeFillet) bool {
	curved := curvedArmsOf(fils)
	if len(fils) != 1 || len(curved) != 1 || curved[0].edge == nil {
		return false
	}
	ef := curved[0]
	if _, ok := ef.armSurface.(geom.Torus); !ok {
		return false // only an exact torus arm forms a closed torus band
	}
	if ef.edge.StartVertex().ID() != ef.edge.EndVertex().ID() {
		return false // an OPEN rim is a runout/corner, not a closed band
	}
	return ClassifyEdgeConvexity(ef.edge) == EdgeConvex // concave rims (S2/S5) keep flooring
}

// convexClosedRimBandBody welds the convex closed rim into a full torus band: the arm's exact torus, bounded
// by its two closed contact circles on the receded hosts, assembled through the host-agnostic rim rebuild
// (the proven watertight closed band). An EMPTY reason means the returned body is the weld; a non-empty
// reason names the obstruction (with the offending value) and the body is nil, so the caller keeps the clean
// do-no-harm floor (never a partial body).
func convexClosedRimBandBody(body *topo.Body, ef edgeFillet, res Resolution) (*topo.Body, string) {
	rf, reason := solveClosedRimBand(ef, res)
	if reason != "" {
		return nil, reason
	}
	b, err := rebuildWithRimFillet(body, rf)
	if err != nil {
		return nil, fmt.Sprintf("closed-rim band rebuild declined: %v", err)
	}
	return b, ""
}

// solveClosedRimBand resolves the closed-band rim fillet: the curved host-of-revolution + cap faces, the arm torus
// re-framed so its seam sits at the rim-vertex azimuth (so the receded host seam stays a straight ruling on
// the host), and the receded-host seam. Declines (naming the offending value) when the hosts are not one
// curved host + one cap plane, the rim vertex lies on the torus axis, or the torus reframe declines.
func solveClosedRimBand(ef edgeFillet, res Resolution) (*rimFillet, string) {
	arm := ef.armSurface.(geom.Torus)
	e := ef.edge
	devF, capF, ok := rimBandHosts(e)
	if !ok {
		return nil, fmt.Sprintf("closed-rim band: edge %d must border one curved host-of-revolution and one cap plane", e.ID())
	}
	rimV := e.StartVertex()
	ref, err := math.UnitVector3FromVector(perpComponent(arm.Center.VectorTo(rimV.Point()), arm.AxisDir))
	if err != nil {
		return nil, fmt.Sprintf("closed-rim band: rim vertex %v is on the torus axis — degenerate seam frame", rimV.Point())
	}
	tor, err := geom.NewTorusWithRef(arm.Center, arm.AxisDir.AsVector(), ref.AsVector(), arm.MajorRadius, arm.MinorRadius)
	if err != nil {
		return nil, fmt.Sprintf("closed-rim band: torus reframe declined: %v", err)
	}
	return assembleRimBand(ef, devF, capF, tor, ref, rimV, res)
}

// rimBandHosts splits a closed rim edge's two faces into the curved host of revolution (cone, cylinder, or sphere) and the cap
// PLANE. ok=false unless exactly one of each — the only pairing a closed circular rim can carry (a plane∧
// plane edge is a straight line, never a circle).
func rimBandHosts(e *topo.Edge) (devF, capF *topo.Face, ok bool) {
	faces := e.Faces()
	if len(faces) != 2 {
		return nil, nil, false
	}
	for i := 0; i < 2; i++ {
		if _, isPlane := faces[i].Geometry().(geom.Plane); !isPlane {
			continue
		}
		if _, bothPlanes := faces[1-i].Geometry().(geom.Plane); bothPlanes {
			return nil, nil, false // two planes cannot bound a circular rim
		}
		return faces[1-i], faces[i], true
	}
	return nil, nil, false
}

// assembleRimBand builds the rimFillet the closed-band rebuild consumes: the two closed contact circles
// (torusContactCircle, framed on the shared rim-vertex azimuth ref so their PointAt(0) sit on one meridian),
// the receded-host seam (wallSeam), and the seam arc's on-arc midpoint. Declines (naming the offending
// value) when a contact circle does not resolve or the host has no seam edge to recede.
func assembleRimBand(ef edgeFillet, devF, capF *topo.Face, tor geom.Torus, ref math.UnitVector3, rimV *topo.Vertex, res Resolution) (*rimFillet, string) {
	capCenter, capR, ok1 := torusContactCircle(capF.Geometry(), tor, res)
	devCenter, devR, ok2 := torusContactCircle(devF.Geometry(), tor, res)
	if !ok1 || !ok2 {
		return nil, fmt.Sprintf("closed-rim band: contact circle unresolved (cap ok=%v, host %T ok=%v)", ok1, devF.Geometry(), ok2)
	}
	capTan := geom.Circle{Center: capCenter, Normal: tor.AxisDir, RefDir: ref, Radius: capR}
	devTan := geom.Circle{Center: devCenter, Normal: tor.AxisDir, RefDir: ref, Radius: devR}
	seamEdge, bottomV := wallSeam(devF, ef.edge, rimV)
	if seamEdge == nil {
		return nil, fmt.Sprintf("closed-rim band: host %T has no seam edge at the rim vertex to recede", devF.Geometry())
	}
	return &rimFillet{
		cyl: devF, cap: capF, rimEdge: ef.edge, seamEdge: seamEdge, rimV: rimV, bottomV: bottomV,
		cylTan: devTan, capTan: capTan, band: tor, r: tor.MinorRadius, seamMid: rimBandSeamMid(tor, devTan, capTan),
	}, ""
}

// rimBandSeamMid is the on-arc midpoint of the tube seam joining the host-tangent and cap-tangent contacts:
// the torus point at the seam azimuth (u=0, both contacts share it via ref), halfway along the SHORT tube
// arc between the two contacts' tube angles. wrapPi picks the short arc so a cone's non-perpendicular
// contacts (unlike a cylinder's clean v=0/v=π/2) do not bow the seam the long way around the tube.
func rimBandSeamMid(tor geom.Torus, devTan, capTan geom.Circle) math.Point3 {
	_, vDev := tor.ParamAt(devTan.PointAt(0))
	_, vCap := tor.ParamAt(capTan.PointAt(0))
	return tor.PointAt(0, vDev+wrapPi(vCap-vDev)/2)
}
