// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Building ONE revolved face per meridian segment (split out of revolution.go for #2213).
//
// Which analytic surface a segment sweeps is a CLASSIFICATION, not a guess: a segment's two
// endpoints and, for an arc, whether its centre sits on the axis, decide between cylinder, cone,
// torus, sphere cap, sphere zone and plane. One builder per class, each emitting the exact surface
// and its loop; nothing here falls back to a general ruled patch.

// revEdgeClass is the surface type a straight meridian edge revolves to.
type revEdgeClass int

const (
	revEdgeOnAxis   revEdgeClass = iota // both endpoints on the axis: traces no surface
	revEdgeCylinder                     // axis-parallel (or a sub-resolution noise edge)
	revEdgePlane                        // axis-perpendicular: disk / annulus
	revEdgeCone                         // oblique: cone frustum
)

// classifyRevolveEdge dispatches a straight meridian edge on its DIRECTION: the dimensionless
// slope ratio against revSlopeTol decides cylinder vs plane vs cone, so the classification is
// invariant under uniform model scaling (#1603, audit A7). weld (res.Weld()) guards the one
// length-scale degeneracy: an edge shorter than the meridian's coincidence resolution is float
// noise, kept as a (degenerate) cylinder — the old behavior — rather than extrapolating a cone
// apex at r/Δr ≈ 1/noise away.
func classifyRevolveEdge(ni, nj revNode, a math.UnitVector3, weld float64) revEdgeClass {
	if ni.circle == nil && nj.circle == nil {
		return revEdgeOnAxis
	}
	dr := stdmath.Abs(ni.r - nj.r)
	dz := stdmath.Abs(axialGap(ni, nj, a))
	switch {
	case stdmath.Hypot(dr, dz) <= weld:
		return revEdgeCylinder
	case dr <= revSlopeTol*dz:
		return revEdgeCylinder
	case dz <= revSlopeTol*dr:
		return revEdgePlane
	default:
		return revEdgeCone
	}
}

// addRevolutionFace adds the surface-of-revolution face for one meridian edge: a cylinder (axis-
// parallel), a planar disk/annulus (axis-perpendicular), or a cone frustum (oblique). An edge that
// lies on the axis traces no surface and is skipped.
func addRevolutionFace(bld *topo.Builder, ni, nj revNode, a, ref math.UnitVector3, weld float64, feat string, i int) {
	switch classifyRevolveEdge(ni, nj, a, weld) {
	case revEdgeOnAxis:
		return // edge on the axis: no face
	case revEdgeCylinder:
		addRevolutionCylinder(bld, ni, nj, a, ref, feat, i)
	case revEdgePlane:
		addRevolutionPlane(bld, ni, nj, a, feat, i)
	default:
		addRevolutionCone(bld, ni, nj, a, ref, feat, i)
	}
}

// axialGap is the signed distance from node i to node j along the axis (z_j − z_i).
func axialGap(ni, nj revNode, a math.UnitVector3) float64 {
	return float64(ni.center.VectorTo(nj.center).Dot(a.AsVector()))
}

// addRevolutionCylinder adds the periodic cylindrical wall for an axis-parallel edge: a seam
// ruling traversed twice between the two shared circles. The loop is identical regardless of edge
// direction; only the material side flips — an upward edge (dz>0) faces +radial (outer wall,
// AddFace), a downward edge faces −radial (a bore, AddReversedFace).
func addRevolutionCylinder(bld *topo.Builder, ni, nj revNode, a, ref math.UnitVector3, feat string, i int) {
	cyl, err := geom.NewCylinderWithRef(ni.center, a.AsVector(), ref.AsVector(), ni.r)
	if err != nil {
		return
	}
	seam := bld.AddEdge(geom.NewLineSegment(ni.v.Point(), nj.v.Point()), ni.v, nj.v, revLin(feat, "seam-edge", i))
	loop := topo.OuterLoop(topo.Fwd(ni.circle), topo.Fwd(seam), topo.Rev(nj.circle), topo.Rev(seam))
	if nj.center.VectorTo(ni.center).Dot(a.AsVector()) < 0 { // dz>0: i below j
		bld.AddFace(cyl, revLin(feat, "wall", i), loop)
		return
	}
	bld.AddReversedFace(cyl, revLin(feat, "wall", i), loop)
}

// addRevolutionCone adds the periodic conical wall for an oblique edge: a slant ruling (the seam)
// traversed twice between the shared circles, exactly like the cylinder but on a geom.Cone. The
// cone shares the meridian frame (NewConeWithRef) so its seam lines up with the neighbouring faces.
// An endpoint on the axis (r=0) is the cone apex — that end contributes no circle, just the seam.
func addRevolutionCone(bld *topo.Builder, ni, nj revNode, a, ref math.UnitVector3, feat string, i int) {
	apex := coneApex(ni, nj)
	base := ni
	if ni.circle == nil {
		base = nj
	}
	axisDir := apex.VectorTo(base.center) // along the axis, apex → base (increasing radius)
	half := stdmath.Atan2(base.r, stdmath.Abs(float64(axisDir.Dot(a.AsVector()))))
	cone, err := geom.NewConeWithRef(apex, axisDir, ref.AsVector(), half)
	if err != nil {
		return
	}
	seam := bld.AddEdge(geom.NewLineSegment(ni.v.Point(), nj.v.Point()), ni.v, nj.v, revLin(feat, "cone-seam", i))
	loop := coneLoop(ni, nj, seam)
	if axialGap(ni, nj, a) > 0 { // dz>0: outward-facing cone (AddFace), like the cylinder rule
		bld.AddFace(cone, revLin(feat, "cone", i), loop)
		return
	}
	bld.AddReversedFace(cone, revLin(feat, "cone", i), loop)
}

// addRevolutionTorus adds the periodic toroidal wall for an ARC meridian edge (a fillet): the arc
// (radius = minor) revolves to a geom.Torus whose tube centre circle is the arc centre revolved
// (major = arc-centre radius). The seam is the arc itself at angle 0, traversed twice between the two
// shared circles — the cylinder idiom with a curved seam. Both endpoints are off-axis (a fillet).
func addRevolutionTorus(bld *topo.Builder, ni, nj revNode, arcCenter math.Point2, axisOrigin math.Point3, a, ref math.UnitVector3, feat string, i int) {
	rc, zc := float64(arcCenter.X), float64(arcCenter.Y)
	zi := float64(axisOrigin.VectorTo(ni.center).Dot(a.AsVector()))
	zj := float64(axisOrigin.VectorTo(nj.center).Dot(a.AsVector()))
	minor := stdmath.Hypot(ni.r-rc, zi-zc)
	torusCenter := axisOrigin.TranslateBy(a.AsVector().Scale(math.Scalar(zc)))
	torus, err := geom.NewTorusWithRef(torusCenter, a.AsVector(), ref.AsVector(), rc, minor)
	if err != nil {
		return
	}
	mid := arcMidpoint(axisOrigin, a, ref, arcCenter, minor, ni.r, zi, nj.r, zj)
	seamArc, err := geom.Arc3dByThreePoints(ni.v.Point(), mid, nj.v.Point())
	if err != nil {
		return
	}
	seam := bld.AddEdge(seamArc, ni.v, nj.v, revLin(feat, "fillet-seam", i))
	loop := topo.OuterLoop(topo.Fwd(ni.circle), topo.Fwd(seam), topo.Rev(nj.circle), topo.Rev(seam))
	if axialGap(ni, nj, a) > 0 { // dz>0: convex fillet faces outward (AddFace), like cylinder/cone
		bld.AddFace(torus, revLin(feat, "fillet", i), loop)
		return
	}
	bld.AddReversedFace(torus, revLin(feat, "fillet", i), loop)
}

// addRevolutionSphereCap adds the analytic spherical-cap face for an ARC meridian edge whose centre
// lies ON the axis: the arc runs from a rim latitude circle to the pole on the axis, so it revolves to
// the polar zone of a sphere centred at the revolved arc centre (radius = the arc radius). Following
// the cone-apex idiom (coneLoop) and the full disk (addRevolutionPlane), the face is bounded by the
// SINGLE rim circle with the pole left IMPLICIT — the sphere's geometric tip, not a topology vertex —
// so the trimmed-face tessellator routes it to sphereCapFan (true spherical bulge). A seamed pole loop
// would carry the off-plane pole in its boundary and fall to the flat best-fit-plane CDT instead
// (a flat lid, zero cap volume — the #1334 bug). One endpoint is the pole (circle==nil).
func addRevolutionSphereCap(bld *topo.Builder, ni, nj revNode, arcCenter math.Point2, axisOrigin math.Point3, a math.UnitVector3, feat string, i int) {
	center := axisOrigin.TranslateBy(a.AsVector().Scale(math.Scalar(arcCenter.Y)))
	rim, use := ni, topo.Fwd(ni.circle)
	if ni.circle == nil { // pole at i, rim at j — mirror coneLoop's apex-at-i collapse (Rev of the far circle)
		rim, use = nj, topo.Rev(nj.circle)
	}
	radius := float64(center.VectorTo(rim.v.Point()).Length()) // sphere radius = |centre → rim|
	sph, err := geom.NewSphere(center, radius)
	if err != nil {
		return
	}
	loop := topo.OuterLoop(use)
	if axialGap(ni, nj, a) > 0 { // dz>0: convex cap faces outward (AddFace), like cylinder/cone/torus
		bld.AddFace(sph, revLin(feat, "cap", i), loop)
		return
	}
	bld.AddReversedFace(sph, revLin(feat, "cap", i), loop)
}

// addRevolutionSphereZone adds the analytic spherical ZONE face for an ARC meridian edge whose centre
// lies ON the axis but whose BOTH endpoints are OFF the axis: the arc is a latitude band of the sphere
// centred at the revolved arc centre (radius = the arc radius). Topologically it is the torus case — two
// shared rim circles bridged by the arc seam traversed twice — but on a geom.Sphere, so the sphere-zone
// tessellator sweeps latitude rings between the two rims with the sphere's true bulge. This is the sphere
// analogue of addRevolutionTorus; caller guarantees both rims are off the equator via sphereZoneAnalytic
// (a band MAY cross it since #2061 — sphereZoneBandFan meshes either kind).
// (self-aligning thrust seat #54; #129 curved-meridian follow-up.)
func addRevolutionSphereZone(bld *topo.Builder, ni, nj revNode, arcCenter math.Point2, axisOrigin math.Point3, a, ref math.UnitVector3, feat string, i int) {
	rc, zc := float64(arcCenter.X), float64(arcCenter.Y)
	zi := float64(axisOrigin.VectorTo(ni.center).Dot(a.AsVector()))
	zj := float64(axisOrigin.VectorTo(nj.center).Dot(a.AsVector()))
	radius := stdmath.Hypot(ni.r-rc, zi-zc) // sphere radius = arc radius (centre → rim, in (r,z))
	center := axisOrigin.TranslateBy(a.AsVector().Scale(math.Scalar(zc)))
	sph, err := geom.NewSphere(center, radius)
	if err != nil {
		return
	}
	mid := arcMidpoint(axisOrigin, a, ref, arcCenter, radius, ni.r, zi, nj.r, zj)
	seamArc, err := geom.Arc3dByThreePoints(ni.v.Point(), mid, nj.v.Point())
	if err != nil {
		return
	}
	seam := bld.AddEdge(seamArc, ni.v, nj.v, revLin(feat, "zone-seam", i))
	loop := topo.OuterLoop(topo.Fwd(ni.circle), topo.Fwd(seam), topo.Rev(nj.circle), topo.Rev(seam))
	if axialGap(ni, nj, a) > 0 { // dz>0: convex band faces outward (AddFace), like cylinder/cone/torus
		bld.AddFace(sph, revLin(feat, "zone", i), loop)
		return
	}
	bld.AddReversedFace(sph, revLin(feat, "zone", i), loop)
}

// arcMidpoint returns the 3D point at the middle of the meridian arc (centre (rc,zc) in (r,z),
// radius minor, from (ri,zi) to (rj,zj)), placed at angle 0 in the (ref, axis) half-plane — the
// third point that pins the seam arc's plane and sweep.
func arcMidpoint(axisOrigin math.Point3, a, ref math.UnitVector3, c math.Point2, minor, ri, zi, rj, zj float64) math.Point3 {
	rc, zc := float64(c.X), float64(c.Y)
	sx, sy := ri-rc, zi-zc // start direction from centre
	ex, ey := rj-rc, zj-zc // end direction from centre
	bx, by := sx+ex, sy+ey // bisector (mid direction for a ≤180° arc)
	bl := stdmath.Hypot(bx, by)
	if bl == 0 { // 180° arc: bisector degenerate — use the perpendicular of the chord
		bx, by = -sy, sx
		bl = stdmath.Hypot(bx, by)
	}
	midR := rc + minor*bx/bl
	midZ := zc + minor*by/bl
	return axisOrigin.TranslateBy(a.AsVector().Scale(math.Scalar(midZ))).TranslateBy(ref.AsVector().Scale(math.Scalar(midR)))
}

// coneApex returns the point where the meridian edge's slant line meets the axis (r=0). An endpoint
// already on the axis (circle == nil, r stored as exactly 0) IS the apex; otherwise it is the
// linear extrapolation to zero radius.
func coneApex(ni, nj revNode) math.Point3 {
	if ni.circle == nil {
		return ni.v.Point()
	}
	if nj.circle == nil {
		return nj.v.Point()
	}
	t := ni.r / (ni.r - nj.r) // r(t)=ni.r+t(nj.r-ni.r)=0
	return ni.v.Point().TranslateBy(ni.v.Point().VectorTo(nj.v.Point()).Scale(math.Scalar(t)))
}

// coneLoop builds the cone face loop: the same seam-doubled periodic loop as a cylinder for a
// frustum (both circles present), collapsing the apex end to just the seam when one radius is 0.
func coneLoop(ni, nj revNode, seam *topo.Edge) topo.LoopSpec {
	switch {
	case ni.circle == nil: // apex at i
		return topo.OuterLoop(topo.Fwd(seam), topo.Rev(nj.circle), topo.Rev(seam))
	case nj.circle == nil: // apex at j
		return topo.OuterLoop(topo.Fwd(ni.circle), topo.Fwd(seam), topo.Rev(seam))
	default: // frustum
		return topo.OuterLoop(topo.Fwd(ni.circle), topo.Fwd(seam), topo.Rev(nj.circle), topo.Rev(seam))
	}
}

// addRevolutionPlane adds the planar disk/annulus for an axis-perpendicular edge. The outward
// normal is −axis when the edge runs outward (dr>0) and +axis when inward (dr<0). The larger-r
// circle is the outer boundary; the smaller is an inner hole (omitted when it sits on the axis,
// making a full disk).
func addRevolutionPlane(bld *topo.Builder, ni, nj revNode, a math.UnitVector3, feat string, i int) {
	outer, inner := ni, nj
	if nj.r > ni.r {
		outer, inner = nj, ni
	}
	outward := a.AsVector()
	if nj.r > ni.r { // edge i→j runs inward→outward (dr>0) ⇒ face points −axis
		outward = a.AsVector().Scale(-1)
	}
	plane, err := geom.NewPlane(ni.center, outward)
	if err != nil {
		return
	}
	loops := planeLoops(outer, inner, outward.Dot(a.AsVector()) < 0)
	bld.AddFace(plane, revLin(feat, "disk", i), loops...)
}

// planeLoops returns the loop specs for a disk/annulus: the outer circle wound CCW about the face
// normal, plus the inner hole wound oppositely (omitted for a full disk). down reports whether the
// face normal is −axis (so the geom.Circle's natural +axis winding must be reversed).
func planeLoops(outer, inner revNode, down bool) []topo.LoopSpec {
	outerUse, innerUse := topo.Fwd, topo.Rev
	if down {
		outerUse, innerUse = topo.Rev, topo.Fwd
	}
	loops := []topo.LoopSpec{topo.OuterLoop(outerUse(outer.circle))}
	if inner.circle != nil {
		loops = append(loops, topo.InnerLoop(innerUse(inner.circle)))
	}
	return loops
}

// revLin is the lineage for a generated surface-of-revolution entity.
func revLin(feat, role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok(feat, role, i)) }
