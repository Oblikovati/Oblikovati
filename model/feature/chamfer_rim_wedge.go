// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A chamfer tool is one section swept along the edge it bevels. Which sweep depends on the edge's
// CURVE, and that is the whole difference between this file and the prism in chamfer.go:
//
//	a straight edge  → sweep the section LINEARLY      → a triangular prism (wedgePrism)
//	a circular rim   → sweep the section ROTATIONALLY  → a revolved ring   (revolvedChamferWedge)
//
// The prism form cannot express a closed rim at all: edgeCornerFrame builds its frame from the
// edge's two endpoints, and a closed circle's endpoints are the SAME point, so it reported
// "chamfer: degenerate edge" and the feature went sick (#1689). That was hidden for as long as
// every curved body was faceted before the chamfer ran — a faceted rim is a chain of short
// straight edges, each with two distinct ends — so removing the faceting (#3459) is what exposed
// it. The section itself is identical in both forms; only the sweep differs.

// closedCircularRim reports the circle an edge runs on when that edge is a CLOSED full circle: a
// geom.Circle whose two endpoints coincide. An arc carries a different curve type, and an open
// circular edge still has two distinct ends, so both keep the prism path.
func closedCircularRim(edge *topo.Edge) (geom.Circle, bool) {
	c, ok := geom.AsCircle(edge.Geometry())
	if !ok {
		return geom.Circle{}, false
	}
	v0, v1 := edge.StartVertex(), edge.EndVertex()
	if v0 == nil || v1 == nil {
		return geom.Circle{}, false
	}
	// Compare POSITIONS, not vertex identity: a seam may be stitched as one vertex used twice or
	// as two coincident vertices, and both are the same closed rim.
	weld := geom.ResolutionForSize(c.Radius).Weld()
	return c, float64(v0.Point().DistanceTo(v1.Point())) <= weld
}

// revolvedChamferWedge builds the chamfer cut tool for a closed circular rim: the section triangle
// {rim, s1 along the first face's interior, s2 along the second's} revolved about the rim's own
// axis. Cutting that ring bevels the whole rim in one operation, and — because it is a revolve —
// the bevel comes out as an exact analytic face rather than a fan of facets.
//
// No overhang is needed. The prism form overhangs its ends so the boolean meets the neighbouring
// faces flush; a revolved ring is closed, so it has no ends to fall short of.
//
// ok=false when the rim's geometry cannot be read (a face whose normal is undefined at the rim, or
// a section that would cross the axis), leaving the caller on its existing path.
func revolvedChamferWedge(edge *topo.Edge, c geom.Circle, s1, s2 float64, feat string) (*topo.Body, bool) {
	mer, ok := rimSectionMeridian(edge, c, s1, s2)
	if !ok {
		return nil, false
	}
	out, err := brep.SolidOfRevolution(c.Center, c.Normal.AsVector(), mer, feat)
	if err != nil || out == nil {
		return nil, false
	}
	return out, true
}

// rimSectionMeridian is the chamfer section at the rim, in the (radius, axial) coordinates
// brep.SolidOfRevolution revolves: the rim itself, plus a leg of s1 into the first adjacent face
// and s2 into the second. ok=false when a face's normal cannot be read at the rim, or a leg would
// carry the section across the axis (which is not a rim chamfer).
func rimSectionMeridian(edge *topo.Edge, c geom.Circle, s1, s2 float64) ([]math.Point2, bool) {
	faces := edge.Faces()
	if len(faces) != 2 || s1 <= 0 || s2 <= 0 {
		return nil, false
	}
	radial := c.RefDir.AsVector()
	axis := c.Normal.AsVector()
	p := c.Center.TranslateBy(radial.Scale(math.Scalar(c.Radius)))
	tangent, err := math.UnitVector3FromVector(axis.Cross(radial))
	if err != nil {
		return nil, false
	}
	probe := 0.25 * stdmath.Min(s1, s2)
	d1, ok1 := rimSetbackDir(faces[0], p, tangent, probe)
	d2, ok2 := rimSetbackDir(faces[1], p, tangent, probe)
	if !ok1 || !ok2 {
		return nil, false
	}
	b := meridianOffset(c.Radius, d1.Scale(math.Scalar(s1)), radial, axis)
	d := meridianOffset(c.Radius, d2.Scale(math.Scalar(s2)), radial, axis)
	if b.X <= 0 || d.X <= 0 {
		return nil, false
	}
	return []math.Point2{math.P2(c.Radius, 0), b, d}, true
}

// rimSetbackDir is the in-surface direction at p, perpendicular to the rim's tangent, pointing
// INTO face f — what interiorDir computes for a straight edge, but read from the SURFACE rather
// than from the face's vertex centroid.
//
// The centroid heuristic is wrong here and quietly so: a full cylinder wall's vertices are its two
// seam ends, so their centroid sits on the axis and the direction comes out tilted radially inward
// when the true setback runs straight down the wall. n×t is exact for any surface, and the probe
// picks the side that is actually inside the face's trim.
func rimSetbackDir(f *topo.Face, p math.Point3, tangent math.UnitVector3, probe float64) (math.Vector3, bool) {
	u, v := f.Geometry().ParamAt(p)
	n, err := math.UnitVector3FromVector(f.Geometry().NormalAt(u, v))
	if err != nil {
		return math.Vector3{}, false
	}
	d, err := math.UnitVector3FromVector(n.AsVector().Cross(tangent.AsVector()))
	if err != nil {
		return math.Vector3{}, false
	}
	for _, cand := range []math.Vector3{d.AsVector(), d.AsVector().Scale(-1)} {
		if brep.PointInFaceTrim(f, p.TranslateBy(cand.Scale(math.Scalar(probe)))) {
			return cand, true
		}
	}
	return math.Vector3{}, false
}

// meridianOffset expresses a 3D offset from the rim as a meridian point (radius, axial) about the
// rim's axis — the coordinates brep.SolidOfRevolution revolves.
func meridianOffset(radius float64, delta, radial, axis math.Vector3) math.Point2 {
	return math.P2(radius+float64(delta.Dot(radial)), float64(delta.Dot(axis)))
}
