// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// ObstacleFeature carries the geometry of a mid-span obstacle patch (ADR-4, spec §3): the obstacle's
// rim curve, the two Nodes P± where it crosses the receded fillet boundary, and the four neighbour
// pieces the 4-sided FillSurface must weld to. WingStart/WingEnd are the section arcs of the abutting
// cylinder wings AT the Nodes — reused BY VALUE from the wing faces so the patch is G1 to them by
// identity and no T-junction crack appears. WallLine is the fillet's wall-tangent seam; HostPlane and
// the wing/rim are the neighbour surfaces the certify measures G1/G0 against.
type ObstacleFeature struct {
	RimCurve           geom.Curve3    // obstacle base rim (T6: the base ellipse), full curve
	RimArcPts          []math.Point3  // ordered dip-side rim samples P- -> P+ INCLUSIVE (task 6); source for the c1 rail fit (obstacleRimArc)
	Nodes              [2]math.Point3 // P-, P+ : rim ∩ receded boundary
	WingStart, WingEnd geom.Curve3    // cylinder-wing section arcs at P-, P+ (the shared end rails)
	WallLine           geom.Curve3    // wall-tangent seam between the Nodes' wall points
	HostPlane          geom.Plane     // the notched host face's plane (for the rim-side G0 side)
}

// bsplineObstacleProvider is the obstacle-variant tier of the corner-blend engine: a single Coons
// FillSurface over the four rails, certified. It Fits only obstacle requests; junction requests fall
// through to the junction providers untouched.
type bsplineObstacleProvider struct{}

// Name reports the provider's telemetry kind (never read by assembly; ADR-2 lineage invariance).
func (bsplineObstacleProvider) Name() CornerBlendKind { return BlendKindBSpline }
