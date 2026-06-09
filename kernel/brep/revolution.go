// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati/kernel/geom"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// revTol is the absolute slope tolerance classifying a meridian edge as axis-parallel
// (cylinder) or axis-perpendicular (plane). Database units are small (cm), so 1e-7 is well
// below any real feature size yet above floating noise from the model→(r,z) projection.
const revTol = 1e-7

// SolidOfRevolution builds a closed ANALYTIC B-rep by revolving a meridian a full 360° about
// the axis line through axisOrigin along axisDir. meridian is the cross-section in the (radius,
// height) half-plane: X = perpendicular distance from the axis (≥0), Y = signed distance along
// axisDir. It must be a simple closed polygon (the last vertex implicitly joins the first).
//
// Each meridian edge becomes a true surface of revolution: an axis-parallel edge → an analytic
// geom.Cylinder, an axis-perpendicular edge → a planar disk (inner r=0) or annulus, sharing
// closed-circle edges and per-cylinder seam rulings so the body is a watertight manifold (the
// same periodic-face idiom as SolidCylinder). This is what lets thread/chamfer/fillet attach to a
// revolved cylindrical face (Oblikovati/Oblikovati#129).
//
// An OBLIQUE meridian edge becomes a true analytic geom.Cone frustum. It returns (nil, nil) — a
// signal to fall back to the faceted revolve — only for fewer than 3 vertices (a CURVED meridian
// edge, e.g. a fillet arc → torus, is a follow-up; this constructor takes a polygonal meridian).
func SolidOfRevolution(axisOrigin math.Point3, axisDir math.Vector3, meridian []math.Point2, feat string) (*topo.Body, error) {
	if len(meridian) < 3 {
		return nil, nil
	}
	a, err := math.UnitVector3FromVector(axisDir)
	if err != nil {
		return nil, err
	}
	mer := ccwMeridian(meridian)
	refCircle, err := geom.NewCircle(axisOrigin, a.AsVector(), 1) // borrow geom's angle-0 frame
	if err != nil {
		return nil, err
	}
	return buildRevolution(axisOrigin, a, refCircle.RefDir, mer, feat), nil
}

// ccwMeridian returns the meridian wound counter-clockwise in (r,z) (positive signed area), so a
// face's right-hand normal points OUT of the solid — the orientation the build rules below assume.
func ccwMeridian(mer []math.Point2) []math.Point2 {
	if signedArea2(mer) >= 0 {
		return mer
	}
	out := make([]math.Point2, len(mer))
	for i, p := range mer {
		out[len(mer)-1-i] = p
	}
	return out
}

// signedArea2 is the shoelace signed area of a closed 2D polygon (positive = CCW).
func signedArea2(p []math.Point2) float64 {
	sum := 0.0
	for i := range p {
		j := (i + 1) % len(p)
		sum += float64(p[i].X*p[j].Y - p[j].X*p[i].Y)
	}
	return sum / 2
}

// revNode is one meridian vertex realized in 3D: the circle of revolution it traces (nil when the
// vertex sits on the axis, r=0) and the seam/apex vertex at angle 0.
type revNode struct {
	center math.Point3
	r      float64
	circle *topo.Edge // nil when r==0 (an axis point: an apex / disk centre)
	v      *topo.Vertex
}

// buildRevolution assembles the body: one circle per off-axis vertex, then one face per meridian
// edge (cylinder for axis-parallel, disk/annulus for axis-perpendicular; on-axis edges skipped).
func buildRevolution(axisOrigin math.Point3, a, ref math.UnitVector3, mer []math.Point2, feat string) *topo.Body {
	bld := topo.NewBuilder(true, revLin(feat, "body", 0))
	nodes := make([]revNode, len(mer))
	for i, m := range mer {
		nodes[i] = makeRevNode(bld, axisOrigin, a, ref, float64(m.X), float64(m.Y), feat, i)
	}
	for i := range mer {
		j := (i + 1) % len(mer)
		addRevolutionFace(bld, nodes[i], nodes[j], a, ref, feat, i)
	}
	return bld.Build()
}

// makeRevNode builds the circle (and seam vertex) for one meridian vertex, or just an apex vertex
// when the vertex lies on the axis (r≈0).
func makeRevNode(bld *topo.Builder, axisOrigin math.Point3, a, ref math.UnitVector3, r, z float64, feat string, i int) revNode {
	center := axisOrigin.TranslateBy(a.AsVector().Scale(math.Scalar(z)))
	if r <= revTol {
		return revNode{center: center, r: 0, v: bld.AddVertex(center, revLin(feat, "apex", i))}
	}
	c := geom.Circle{Center: center, Normal: a, RefDir: ref, Radius: r}
	v := bld.AddVertex(c.PointAt(0), revLin(feat, "seam", i))
	return revNode{center: center, r: r, circle: bld.AddEdge(c, v, v, revLin(feat, "circle", i)), v: v}
}

// addRevolutionFace adds the surface-of-revolution face for one meridian edge: a cylinder (axis-
// parallel), a planar disk/annulus (axis-perpendicular), or a cone frustum (oblique). An edge that
// lies on the axis traces no surface and is skipped.
func addRevolutionFace(bld *topo.Builder, ni, nj revNode, a, ref math.UnitVector3, feat string, i int) {
	if ni.r <= revTol && nj.r <= revTol {
		return // edge on the axis: no face
	}
	switch {
	case stdmath.Abs(ni.r-nj.r) <= revTol:
		addRevolutionCylinder(bld, ni, nj, a, ref, feat, i)
	case stdmath.Abs(axialGap(ni, nj, a)) <= revTol:
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
	if ni.r <= revTol {
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

// coneApex returns the point where the meridian edge's slant line meets the axis (r=0). An endpoint
// already on the axis IS the apex; otherwise it is the linear extrapolation to zero radius.
func coneApex(ni, nj revNode) math.Point3 {
	if ni.r <= revTol {
		return ni.v.Point()
	}
	if nj.r <= revTol {
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
	if inner.r > revTol {
		loops = append(loops, topo.InnerLoop(innerUse(inner.circle)))
	}
	return loops
}

// revLin is the lineage for a generated surface-of-revolution entity.
func revLin(feat, role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok(feat, role, i)) }
