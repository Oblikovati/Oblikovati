// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// holeFacets is the number of sides used to approximate a hole/boss cylinder as a prism
// (an exact analytic cylinder is a NURBS-phase refinement).
const holeFacets = 32

// drillTool builds a faceted cylinder solid: a regular polygon of the given radius,
// centered at center on the plane perpendicular to axisInto, extruded `depth` along
// axisInto. A small entry overhang past the start keeps the boolean's entry face clean.
// It is the cut tool for holes and (joined) the boss body.
func drillTool(center math.Point3, axisInto math.UnitVector3, radius, depth float64, feat string) *topo.Body {
	const overhang = 1e-2
	plane := planePerp(center, axisInto)
	return buildPrism(regularPolygon(radius, holeFacets), plane, span{near: -overhang, far: depth}, 0, feat)
}

// regularPolygon returns an n-gon of the given radius centered at the origin, wound
// counter-clockwise.
func regularPolygon(radius float64, n int) []math.Point2 {
	out := make([]math.Point2, n)
	for i := 0; i < n; i++ {
		a := 2 * stdmath.Pi * float64(i) / float64(n)
		out[i] = math.P2(radius*stdmath.Cos(a), radius*stdmath.Sin(a))
	}
	return out
}

// regularPolygonArea returns the area of an n-gon of the given radius (for tests/asserts).
func regularPolygonArea(radius float64, n int) float64 {
	return 0.5 * float64(n) * radius * radius * stdmath.Sin(2*stdmath.Pi/float64(n))
}

// planePerp returns the sketch plane through origin whose normal is axisInto, so a prism
// extruded along the plane normal drills in the axisInto direction.
func planePerp(origin math.Point3, axisInto math.UnitVector3) sketch.Plane {
	x := perpAxis(axisInto)
	y, _ := math.UnitVector3FromVector(axisInto.Cross(x)) // axisInto × x ⇒ x × y = axisInto
	p, _ := sketch.NewPlane(origin, x, y)
	return p
}

// perpAxis returns some unit vector perpendicular to n, crossing it with whichever world
// axis is least aligned so the cross product is well-conditioned.
func perpAxis(n math.UnitVector3) math.UnitVector3 {
	ref := math.V3(1, 0, 0).AsUnit()
	if stdmath.Abs(n.AsVector().X) > 0.9 {
		ref = math.V3(0, 1, 0).AsUnit()
	}
	u, _ := math.UnitVector3FromVector(n.Cross(ref))
	return u
}

// faceVertexPoints returns the positions of a face's bounding vertices.
func faceVertexPoints(f *topo.Face) []math.Point3 {
	vs := f.Vertices()
	out := make([]math.Point3, len(vs))
	for i, v := range vs {
		out[i] = v.Point()
	}
	return out
}

// replaceBody returns bodies with old replaced by res (res dropped when empty).
func replaceBody(bodies []*topo.Body, old, res *topo.Body) []*topo.Body {
	out := make([]*topo.Body, 0, len(bodies))
	for _, b := range bodies {
		if b == old {
			if res != nil && len(res.Faces()) > 0 {
				out = append(out, res)
			}
			continue
		}
		out = append(out, b)
	}
	return out
}
