// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CutBlindConicalHole drills a blind hole with a conical drill point (K1b): a cylinder bore of
// `radius` to `depth` (the full-diameter depth) closed by a true CONE tip — the shape a twist
// drill leaves. pointHalfAngle is the half-angle between the axis and the cone surface (a 118°
// included drill point is half-angle 59°). The tip's apex is a geometric pole, not a topology
// vertex, so the cone face's only boundary is the bore-bottom circle (tessellated as an apex
// fan). The whole pocket, apex included, must stay inside the part or this returns an error.
func CutBlindConicalHole(slab *topo.Body, base math.Point3, axisDir math.Vector3, radius, depth, pointHalfAngle float64) (*topo.Body, error) {
	if radius <= 0 || depth <= 0 {
		return nil, fmt.Errorf("brep: conical hole needs radius>0 and depth>0, got r=%g depth=%g", radius, depth)
	}
	if pointHalfAngle <= 0 || pointHalfAngle >= stdmath.Pi/2 {
		return nil, fmt.Errorf("brep: drill point half-angle %g must be in (0, π/2)", pointHalfAngle)
	}
	ua := unit(axisDir)
	if ua.LengthSquared() < 0.5 {
		return nil, fmt.Errorf("brep: drill axis direction is degenerate: %+v", axisDir)
	}
	copied, entry, err := classifyBlindDrill(slab, base, ua, radius)
	if err != nil {
		return nil, err
	}
	bottom := base.TranslateBy(ua.Scale(math.Scalar(depth)))
	apex := bottom.TranslateBy(ua.Scale(math.Scalar(radius / stdmath.Tan(pointHalfAngle))))
	if err := checkBlindFits(slab, entry, bottom, radius); err != nil {
		return nil, err
	}
	if !insideSolid(slab, apex) {
		return nil, fmt.Errorf("brep: conical drill point at %+v exits the part", apex)
	}
	return assembleBlindConical(copied, entry, base, bottom, apex, ua, radius, pointHalfAngle)
}

// assembleBlindConical welds the bore and caps it with a cone tip (apex deep on the axis), whose
// material side faces the axis (a reversed face) like the cylinder wall.
func assembleBlindConical(copied []curvedFace, entry curvedFace, base, bottom, apex math.Point3, ua math.Vector3, radius, halfAngle float64) (*topo.Body, error) {
	bld, holeBot, err := blindBore(copied, entry, base, bottom, ua, radius)
	if err != nil {
		return nil, err
	}
	tip, err := geom.NewCone(apex, ua.Scale(-1), halfAngle) // axis points back up toward the rim
	if err != nil {
		return nil, err
	}
	bld.AddReversedFace(tip, topo.NewLineage(topo.Tok("brep", "drillpoint", 0)), topo.OuterLoop(topo.Fwd(holeBot)))
	return bld.Build(), nil
}
