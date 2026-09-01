// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Exact curved − convex-planar subtract (M2 Phase 1, Oblikovati/Oblikovati#1334). Cut is not a half-space
// composition (a box is not a half-space complement); cylinder − box keeps the cylinder outside the box
// and the box walls inside the cylinder, flipped. This wires the clean case: a box that tunnels straight
// through the cylinder along its axis, inside the radius — an exact prismatic hole (brep.SubtractAxialPrism),
// the cylinder surfaces preserved. Anything else (a side-breaching or partial tunnel, a tilted box, a
// non-cylinder target) returns ok=false so booleanGeneral keeps the CSG fallback — no regression.

// axisAlignTol bounds how nearly a box face must be parallel or perpendicular to the cylinder axis for the
// axial-prism subtract to apply; a more tilted box tunnels along no clean axis and defers to CSG.
const axisAlignTol = 1e-7

// curvedConvexSubtract returns cylinder − box when the box is an axial through-tunnel inside the cylinder,
// or ok=false to defer. Only Cut maps here, and only with a curved target (the body being cut) and a
// convex all-planar tool whose faces are each parallel or perpendicular to the cylinder axis.
func curvedConvexSubtract(op PartFeatureOperation, target, tool *topo.Body, _ *diag.Recorder) (*topo.Body, bool) {
	if op != Cut || !hasCurvedFace(target) || hasCurvedFace(tool) {
		return nil, false
	}
	if _, convex := convexFacePlanes(tool); !convex {
		return nil, false // a non-convex tool: its hull cross-section would over-remove material
	}
	cyl, base, height, ok := brep.CylinderParams(target)
	if !ok {
		return nil, false
	}
	axis := cyl.AxisDir.AsVector()
	if !boxAxisAligned(tool, axis) || !boxSpansAxially(tool, base, axis, height) {
		return nil, false
	}
	poly := boxAxialCrossSection(tool, cyl, base)
	if len(poly) < 3 {
		return nil, false
	}
	res, err := brep.SubtractAxialPrism(target, poly)
	if err != nil || !Validate(res).ValidSolid() {
		return nil, false
	}
	return res, true
}

// boxAxisAligned reports whether every tool face is parallel or perpendicular to the axis (so the tool is
// a prism whose tube walls run straight along the axis), the shape the axial-prism subtract can build.
func boxAxisAligned(tool *topo.Body, axis math.Vector3) bool {
	for _, f := range tool.Faces() {
		pl, planar := f.Geometry().(geom.Plane)
		if !planar {
			return false
		}
		along := stdmath.Abs(float64(probe.Unit(pl.NormalAt(0, 0)).Dot(axis)))
		if along > axisAlignTol && along < 1-axisAlignTol {
			return false // an oblique wall: no clean axial tube
		}
	}
	return true
}

// boxSpansAxially reports whether the tool fully spans the cylinder along the axis (a through-tunnel, not
// a blind pocket): every cylinder cap level lies between the tool's lowest and highest vertices.
func boxSpansAxially(tool *topo.Body, base math.Point3, axis math.Vector3, height float64) bool {
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, v := range tool.Vertices() {
		s := float64(base.VectorTo(v.Point()).Dot(axis))
		lo, hi = stdmath.Min(lo, s), stdmath.Max(hi, s)
	}
	return lo <= axisAlignTol && hi >= height-axisAlignTol
}

// boxAxialCrossSection projects the tool's vertices onto the base cap plane and convex-hulls them into the
// tunnel's cross-section polygon (ordered CCW about the axis, lifted back to 3D at the base).
func boxAxialCrossSection(tool *topo.Body, cyl geom.Cylinder, base math.Point3) []math.Point3 {
	e1 := cyl.Ref.AsVector()
	e2 := cyl.AxisDir.AsVector().Cross(e1)
	pts := make([]math.Point2, 0, len(tool.Vertices()))
	for _, v := range tool.Vertices() {
		r := base.VectorTo(v.Point())
		pts = append(pts, math.P2(r.Dot(e1), r.Dot(e2)))
	}
	hull := convexHull2D(pts)
	out := make([]math.Point3, len(hull))
	for i, p := range hull {
		out[i] = base.TranslateBy(e1.Scale(p.X)).TranslateBy(e2.Scale(p.Y))
	}
	return out
}
