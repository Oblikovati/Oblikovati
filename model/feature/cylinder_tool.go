// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// holeFacets is the number of sides used to approximate a hole/boss cylinder as a prism
// (an exact analytic cylinder is a NURBS-phase refinement).
const holeFacets = 32

// cutterOverhang extends a cut tool a small distance past the surface it enters/exits, so the
// boolean's entry/exit faces sit clear of the target's faces rather than coincident with them
// (a coincident pair would leave a zero-thickness sliver). Database units; shared by the drill
// tool and the chamfer wedge.
const cutterOverhang = 1e-2

// drillTool builds a faceted cylinder solid: a regular polygon of the given radius,
// centered at center on the plane perpendicular to axisInto, extruded `depth` along
// axisInto. A small entry overhang past the start keeps the boolean's entry face clean.
// It is the cut tool for holes and (joined) the boss body.
func drillTool(center math.Point3, axisInto math.UnitVector3, radius, depth float64, feat string) *topo.Body {
	return drillToolFrom(center, axisInto, radius, depth, cutterOverhang, feat)
}

// drillToolFrom is drillTool with the entry overhang named. The overhang exists to keep the tool's
// entry face off a coincident target face, which only happens when the bore starts AT a surface; a
// bore that starts inside material (a from-to hole, #1863) must begin exactly where it was asked
// to, so it passes 0 rather than eating an overhang's worth of real material.
func drillToolFrom(center math.Point3, axisInto math.UnitVector3, radius, depth, entry float64, feat string) *topo.Body {
	plane := planePerp(center, axisInto)
	return buildPrism(regularPolygon(radius, holeFacets), plane, span{near: -entry, far: depth}, 0, feat)
}

// cylinderTool is [drillToolFrom]'s ANALYTIC twin: the same swept volume as a real
// geom.Cylinder rather than a holeFacets-gon prism.
//
// It exists for the tool a hole RECORDS for replay, not for the tool it cuts with — the cut goes
// through the exact brep.Cut*Hole builders. A pattern re-applies the recorded tool at each
// occurrence, so a faceted one made the pattern remove a different volume from the one the
// original hole removed, and shattered the target's analytic faces on the way: a 3-up pattern of
// one Ø2 bore turned a 7-face body into 5234 planar faces (#3463). The boolean handles the
// analytic cylinder directly.
//
// Falls back to the prism only if the cylinder cannot be built (a degenerate radius or height),
// so a caller always gets a tool.
func cylinderTool(center math.Point3, axisInto math.UnitVector3, radius, depth, entry float64, feat string) *topo.Body {
	base := center.TranslateBy(axisInto.AsVector().Scale(math.Scalar(-entry)))
	cyl, err := brep.SolidCylinder(base, axisInto.AsVector(), radius, depth+entry)
	if err != nil || cyl == nil {
		return drillToolFrom(center, axisInto, radius, depth, entry, feat)
	}
	return cyl
}

// regularPolygon returns an n-gon of the given radius centered at the origin, wound
// counter-clockwise.
func regularPolygon(radius float64, n int) []math.Point2 {
	out := make([]math.Point2, n)
	for i := range n {
		a := 2 * stdmath.Pi * float64(i) / float64(n)
		out[i] = math.P2(radius*stdmath.Cos(a), radius*stdmath.Sin(a))
	}
	return out
}

// drillToolPrismArea is the cross-section area of the [drillTool] PRISM: the holeFacets-gon it is
// built from, not a circle. It is NOT what a drilled hole removes any more — HoleFeature.cutCylinder
// prefers the exact brep drills, so an ordinary bore takes out a true geom.Cylinder of area πr², and
// mass properties now integrate that analytic B-rep (M48/C3 #3453). This area is what the paths that
// still cut with the faceted prism remove or add: the boss ([BossFeature]), the assembly hole, the
// faceted counterbore fallback, and any bore the exact drill declines (one starting inside material,
// or tangent to a face it would clip). Volume assertions use it ONLY for those, so they measure the
// geometry the kernel built rather than the one it approximates.
//
// Example: want := blockVolume + drillToolPrismArea(bossR)*bossHeight // a boss's faceted stud
func drillToolPrismArea(radius float64) float64 {
	n := float64(holeFacets)
	return 0.5 * n * radius * radius * stdmath.Sin(2*stdmath.Pi/n)
}

// planePerp returns the sketch plane through origin whose normal is axisInto, so a prism
// extruded along the plane normal drills in the axisInto direction.
func planePerp(origin math.Point3, axisInto math.UnitVector3) sketch.Plane {
	return planeFromFrame(origin, axisInto, perpAxis(axisInto))
}

// planeFromFrame returns the sketch plane through origin with an EXPLICIT in-plane X axis
// (ref) and normal (axisInto); Y = axisInto × ref, so plane.Normal() == axisInto. Unlike
// planePerp (which picks an arbitrary ref), this preserves a caller-supplied phase — used to
// re-facet an analytic cylinder back into a prism in its original generating frame so facet
// identity stays stable (#129). ref must be perpendicular to axisInto.
func planeFromFrame(origin math.Point3, axisInto, ref math.UnitVector3) sketch.Plane {
	y, _ := math.UnitVector3FromVector(axisInto.Cross(ref)) // axisInto × ref ⇒ ref × y = axisInto
	p, _ := sketch.NewPlane(origin, ref, y)
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

// throughDepth returns a drill depth that fully exits the body along the inward axis: the
// farthest extent of the body's range box from center, plus an overhang for a clean exit face.
// Used as the faceted fallback when an exact through-hole isn't possible (see HoleFeature.drill).
func throughDepth(body *topo.Body, center math.Point3, into math.UnitVector3) float64 {
	axis := into.AsVector()
	far := 0.0
	for _, c := range boxCorners(body.RangeBox()) {
		if d := float64(center.VectorTo(c).Dot(axis)); d > far {
			far = d
		}
	}
	return far + cutterOverhang
}

// boxCorners returns the eight corners of an axis-aligned box.
func boxCorners(b math.Box) []math.Point3 {
	out := make([]math.Point3, 0, 8)
	for _, x := range [2]math.Scalar{b.Min.X, b.Max.X} {
		for _, y := range [2]math.Scalar{b.Min.Y, b.Max.Y} {
			for _, z := range [2]math.Scalar{b.Min.Z, b.Max.Z} {
				out = append(out, math.Point3{X: x, Y: y, Z: z})
			}
		}
	}
	return out
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
