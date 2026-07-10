// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// revSlopeTol is the DIMENSIONLESS meridian slope threshold classifying an edge as axis-parallel
// (cylinder: |Δr| ≤ revSlopeTol·|Δz|) or axis-perpendicular (plane: |Δz| ≤ revSlopeTol·|Δr|).
// Surface type is a question about the edge's DIRECTION — an angle — so the threshold is an
// angular tolerance, scale-free by construction (#1603, audit A7). 1e-9 matches geom's epsRel
// relative-precision precedent: it sits a decade below the smallest taper classification must
// preserve (1e-8 rad) and ~2 decades above the slope noise of a double-precision model→(r,z)
// projection on proportionate geometry (ε·r/L ≈ 1e-13 at r/L ~ 1e3). OCCT can run its analogous
// cone-vs-cylinder recognition at Precision::Angular()=1e-12 only because it reads exact curve
// directions; we difference meridian endpoints, so Δr carries cancellation noise ~ε·r.
const revSlopeTol = 1e-9 // tol:angular

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
// signal to fall back to the faceted revolve — only for fewer than 3 vertices. For a CURVED meridian
// edge (a fillet arc → torus) use SolidOfRevolutionMeridian with an arc vertex.
func SolidOfRevolution(axisOrigin math.Point3, axisDir math.Vector3, meridian []math.Point2, feat string) (*topo.Body, error) {
	if len(meridian) < 3 {
		return nil, nil
	}
	mer := ccwMeridian(meridian)
	verts := make([]RevolveVertex, len(mer))
	for i := range mer {
		verts[i] = RevolveVertex{P: mer[i]}
	}
	return SolidOfRevolutionMeridian(axisOrigin, axisDir, verts, feat)
}

// RevolveVertex is one meridian vertex at (radius, height). When ArcCenter is non-nil the edge from
// the PREVIOUS vertex to this one is a circular ARC about that (radius, height) centre — revolving
// to a geom.Torus face (a fillet) — otherwise a straight line (cylinder/cone/plane).
type RevolveVertex struct {
	P         math.Point2
	ArcCenter *math.Point2
}

// SolidOfRevolutionMeridian is the general builder: it revolves a meridian whose edges may be lines
// OR arcs 360° about the axis. An arc edge whose centre is OFF the axis revolves to a geom.Torus
// (a fillet); an arc whose centre is ON the axis and that runs from a rim circle to the pole (one
// endpoint on the axis) revolves to a spherical CAP (a domed end). The meridian is auto-oriented to
// counter-clockwise in (r,z) so each face's right-hand normal points out of the solid. Returns
// (nil,nil) for fewer than 3 vertices OR when an arc revolves to a surface not yet built analytically
// (a sphere ZONE with both endpoints off-axis, or a pole-to-pole meridian) — the caller then keeps the
// faceted revolve.
func SolidOfRevolutionMeridian(axisOrigin math.Point3, axisDir math.Vector3, verts []RevolveVertex, feat string) (*topo.Body, error) {
	if len(verts) < 3 {
		return nil, nil
	}
	a, err := math.UnitVector3FromVector(axisDir)
	if err != nil {
		return nil, err
	}
	refCircle, err := geom.NewCircle(axisOrigin, a.AsVector(), 1) // borrow geom's angle-0 frame
	if err != nil {
		return nil, err
	}
	verts = ccwMeridianVerts(verts) // right-hand face normals must point out of the solid
	if !revolveArcsAnalytic(verts, revolveResolution(verts).Plane()) {
		return nil, nil // an arc revolves to a surface we don't build analytically: fall back to faceted
	}
	return buildRevolution(axisOrigin, a, refCircle.RefDir, verts, feat), nil
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

// ccwMeridianVerts returns the meridian vertices wound counter-clockwise in (r,z) so a face's
// right-hand normal points OUT of the solid. Arc info lives on an edge's END vertex, so reversing the
// ring must re-key every ArcCenter: the reversed edge k is old edge (n−1−k) traversed backwards, and
// its new end is that old edge's START vertex — the vertex ONE slot back around the original ring.
func ccwMeridianVerts(verts []RevolveVertex) []RevolveVertex {
	pts := make([]math.Point2, len(verts))
	for i, v := range verts {
		pts[i] = v.P
	}
	if signedArea2(pts) >= 0 {
		return verts
	}
	n := len(verts)
	out := make([]RevolveVertex, n)
	for k := 0; k < n; k++ {
		oldEdge := n - 1 - k
		out[k] = RevolveVertex{P: verts[(oldEdge-1+n)%n].P, ArcCenter: verts[oldEdge].ArcCenter}
	}
	return out
}

// arcCenterOnAxis reports whether an arc centre (in (r,z)) sits on the axis of revolution — its radial
// coordinate is below the meridian's own coincidence resolution. An on-axis centre revolves the arc to
// a SPHERE (the arc traces a meridian of that sphere); an off-axis centre gives a torus.
func arcCenterOnAxis(center math.Point2, axisTol float64) bool {
	return stdmath.Abs(float64(center.X)) <= axisTol
}

// revolveArcsAnalytic reports whether every ARC edge of the meridian revolves to a surface this
// builder emits analytically: a torus (centre off-axis) or a spherical CAP (centre on-axis with
// EXACTLY one endpoint on the axis, i.e. the arc runs from a rim latitude to the pole). A sphere ZONE
// (centre on-axis, both endpoints off-axis) needs a framed sphere parameterization we do not yet
// build, and a pole-to-pole meridian is degenerate; both return false so the caller keeps the faceted
// revolve (no regression — today every curved-meridian revolve facets).
func revolveArcsAnalytic(verts []RevolveVertex, axisTol float64) bool {
	n := len(verts)
	for i := range verts {
		if verts[i].ArcCenter == nil || !arcCenterOnAxis(*verts[i].ArcCenter, axisTol) {
			continue // a straight edge or an off-axis (torus) arc — both handled
		}
		onPrev := float64(verts[(i-1+n)%n].P.X) <= axisTol
		onCur := float64(verts[i].P.X) <= axisTol
		if onPrev == onCur {
			return false // zone (neither endpoint on-axis) or pole-to-pole (both): not analytic yet
		}
	}
	return true
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
// edge (cylinder for axis-parallel, disk/annulus for axis-perpendicular, cone for oblique, torus for
// an off-axis arc, spherical cap for an on-axis arc closing at the pole; on-axis line edges skipped).
func buildRevolution(axisOrigin math.Point3, a, ref math.UnitVector3, verts []RevolveVertex, feat string) *topo.Body {
	bld := topo.NewBuilder(true, revLin(feat, "body", 0))
	res := revolveResolution(verts)
	nodes := make([]revNode, len(verts))
	for i, vrt := range verts {
		nodes[i] = makeRevNode(bld, axisOrigin, a, ref, float64(vrt.P.X), float64(vrt.P.Y), res.Plane(), feat, i)
	}
	for i := range verts {
		j := (i + 1) % len(verts)
		if verts[j].ArcCenter != nil {
			if arcCenterOnAxis(*verts[j].ArcCenter, res.Plane()) {
				addRevolutionSphereCap(bld, nodes[i], nodes[j], *verts[j].ArcCenter, axisOrigin, a, feat, i)
			} else {
				addRevolutionTorus(bld, nodes[i], nodes[j], *verts[j].ArcCenter, axisOrigin, a, ref, feat, i)
			}
			continue
		}
		addRevolutionFace(bld, nodes[i], nodes[j], a, ref, res.Weld(), feat, i)
	}
	return bld.Build()
}

// revolveResolution derives the meridian's own model-relative coincidence scale (ADR-0042): the
// LENGTH tolerances the build needs — the on-axis vertex weld and the degenerate-edge guard — flow
// from the meridian's extent instead of a cm-anchored absolute constant (#1603, audit A7).
func revolveResolution(verts []RevolveVertex) geom.Resolution {
	pts := make([]math.Point2, len(verts))
	for i, v := range verts {
		pts[i] = v.P
	}
	return geom.ResolutionForPoints2D(pts)
}

// makeRevNode builds the circle (and seam vertex) for one meridian vertex, or just an apex vertex
// when the vertex lies on the axis. axisTol is the meridian-relative on-line classification
// tolerance (res.Plane()): a radius below it is coincident with the axis at this model's
// resolution, so the node welds to an apex/disk-centre vertex (r stored as exactly 0 — downstream
// on-axis tests read circle == nil).
func makeRevNode(bld *topo.Builder, axisOrigin math.Point3, a, ref math.UnitVector3, r, z, axisTol float64, feat string, i int) revNode {
	center := axisOrigin.TranslateBy(a.AsVector().Scale(math.Scalar(z)))
	if r <= axisTol {
		return revNode{center: center, r: 0, v: bld.AddVertex(center, revLin(feat, "apex", i))}
	}
	c := geom.Circle{Center: center, Normal: a, RefDir: ref, Radius: r}
	v := bld.AddVertex(c.PointAt(0), revLin(feat, "seam", i))
	return revNode{center: center, r: r, circle: bld.AddEdge(c, v, v, revLin(feat, "circle", i)), v: v}
}

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
