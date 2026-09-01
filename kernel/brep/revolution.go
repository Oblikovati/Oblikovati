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
// (a pole-to-pole meridian — a whole sphere, which SolidSphere builds instead) — the caller then keeps
// the faceted revolve.
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
	for k := range n {
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
// builder emits analytically: a torus (centre off-axis), a spherical CAP (centre on-axis, EXACTLY one
// endpoint on the axis — a rim latitude to the pole), or a spherical ZONE (centre on-axis, BOTH
// endpoints off-axis — a latitude band, addRevolutionSphereZone; since #2061 the band may cross the
// equator, because sphereZoneBandFan meshes either kind). The one on-axis case it still declines is a
// pole-to-pole meridian (both endpoints on the axis — a whole sphere; SolidSphere is that route), which
// returns false so the caller keeps the faceted revolve.
func revolveArcsAnalytic(verts []RevolveVertex, axisTol float64) bool {
	n := len(verts)
	for i := range verts {
		if verts[i].ArcCenter == nil || !arcCenterOnAxis(*verts[i].ArcCenter, axisTol) {
			continue // a straight edge or an off-axis (torus) arc — both handled
		}
		prev := verts[(i-1+n)%n].P
		cur := verts[i].P
		onPrev := float64(prev.X) <= axisTol
		onCur := float64(cur.X) <= axisTol
		if onPrev != onCur {
			continue // exactly one endpoint on the axis: a spherical cap — handled
		}
		if onPrev { // both endpoints on the axis: a pole-to-pole meridian — a whole sphere, declined here
			return false
		}
		if !sphereZoneAnalytic(prev, cur, *verts[i].ArcCenter, axisTol) {
			return false // an equator-straddling zone: not a single monotone-colatitude band yet
		}
	}
	return true
}

// sphereZoneAnalytic reports whether an on-axis-centre arc between two OFF-axis endpoints revolves to a
// single analytic spherical ZONE the builder emits (addRevolutionSphereZone). Each endpoint must be a
// proper latitude circle, which fails only when a rim sits ON the equator (|Δz| ≤ axisTol): its radius
// is then the sphere's own, so the zone is tangent to whatever cylinder adjoins it there and the shared
// rim is degenerate. axisTol is the meridian-relative coincidence scale (res.Plane()).
//
// It used to ALSO require both rims on the same side of the equator, because an equator-crossing band
// had no analytic mesh — kernel/ops had no zone mesher and the gnomonic patch chart cannot cover a belt
// spanning both hemispheres, so such a face came out ~75% short in area. sphereZoneBandFan now sweeps
// latitude rings about the rims' own axis and meshes either kind exactly, so the restriction is gone
// and a barrel/bead meridian revolves analytically instead of facetting (#2061).
func sphereZoneAnalytic(prev, cur, center math.Point2, axisTol float64) bool {
	dzPrev := float64(prev.Y - center.Y)
	dzCur := float64(cur.Y - center.Y)
	return stdmath.Abs(dzPrev) > axisTol && stdmath.Abs(dzCur) > axisTol
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
			switch {
			case !arcCenterOnAxis(*verts[j].ArcCenter, res.Plane()):
				addRevolutionTorus(bld, nodes[i], nodes[j], *verts[j].ArcCenter, axisOrigin, a, ref, feat, i)
			case nodes[i].circle == nil || nodes[j].circle == nil:
				addRevolutionSphereCap(bld, nodes[i], nodes[j], *verts[j].ArcCenter, axisOrigin, a, feat, i)
			default:
				addRevolutionSphereZone(bld, nodes[i], nodes[j], *verts[j].ArcCenter, axisOrigin, a, ref, feat, i)
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
