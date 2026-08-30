// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Exact curved ∩ convex-planar boolean (M2 Phase 1, Oblikovati/Oblikovati#1334). A convex planar solid
// (a box) is the intersection of its faces' inner half-spaces, so intersecting a curved analytic solid
// with it is just composing brep.HalfSpaceCut over those planes — an EXACT curved B-rep (cylinder/sphere
// surfaces preserved), not the triangle-soup CSG. booleanGeneral tries this before falling back to CSG;
// any operand or cut it cannot handle (a non-convex tool, an unhandled face) returns ok=false, so there
// is no regression — the CSG path still runs.

// curvedConvexIntersect returns the exact intersection of a curved solid with a convex planar tool, or
// ok=false to defer to CSG. Only the Intersect operation maps to a half-space composition; Join and Cut
// (a box is not a half-space complement) stay on the existing path.
func curvedConvexIntersect(op PartFeatureOperation, target, tool *topo.Body, _ *diag.Recorder) (*topo.Body, bool) {
	if op != Intersect {
		return nil, false
	}
	curved, planar := orderCurvedPlanar(target, tool)
	if curved == nil {
		return nil, false
	}
	planes, ok := convexFacePlanes(planar)
	if !ok {
		return nil, false
	}
	body := curved
	for _, pl := range planes {
		res, err := brep.HalfSpaceCut(body, pl)
		if err != nil {
			return nil, false // an unhandled cut: defer the whole boolean to CSG
		}
		if len(res.Faces()) == 0 {
			return res, true // a plane removed everything: the intersection is empty
		}
		body = res
	}
	if !Validate(body).ValidSolid() {
		return nil, false
	}
	return body, true
}

// orderCurvedPlanar returns the operand that carries a curved (non-planar) face and the all-planar one,
// or (nil, nil) when they are not one of each (both planar, or both curved).
func orderCurvedPlanar(a, b *topo.Body) (curved, planar *topo.Body) {
	switch {
	case hasCurvedFace(a) && !hasCurvedFace(b):
		return a, b
	case hasCurvedFace(b) && !hasCurvedFace(a):
		return b, a
	default:
		return nil, nil
	}
}

// hasCurvedFace reports whether a body has any non-planar face.
func hasCurvedFace(b *topo.Body) bool {
	for _, f := range b.Faces() {
		if _, planar := f.Geometry().(geom.Plane); !planar {
			return true
		}
	}
	return false
}

// convexFacePlanes returns each face's plane oriented along its OUTWARD normal, provided the body is an
// all-planar CONVEX solid (every vertex on the inner side of every face plane). HalfSpaceCut keeps each
// plane's negative side, which is the convex solid's interior, so composing them carves the tool.
func convexFacePlanes(b *topo.Body) ([]geom.Plane, bool) {
	planes, normals, origins, ok := outwardFacePlanes(b)
	if !ok {
		return nil, false
	}
	tol := ResolutionForBodies(b).Plane()
	for _, v := range b.Vertices() {
		for i := range normals {
			if float64(origins[i].VectorTo(v.Point()).Dot(normals[i])) > tol {
				return nil, false // a vertex lies outside a face plane → the solid is not convex
			}
		}
	}
	return planes, true
}

// outwardFacePlanes collects each planar face's outward-oriented plane (and its normal/origin for the
// convexity test). ok=false if any face is non-planar or degenerate.
func outwardFacePlanes(b *topo.Body) (planes []geom.Plane, normals []math.Vector3, origins []math.Point3, ok bool) {
	for _, f := range b.Faces() {
		pl, planar := f.Geometry().(geom.Plane)
		if !planar {
			return nil, nil, nil, false
		}
		n := pl.NormalAt(0, 0)
		if f.Reversed() {
			n = n.Scale(-1)
		}
		out, err := geom.NewPlane(pl.Origin, n)
		if err != nil {
			return nil, nil, nil, false
		}
		planes = append(planes, out)
		normals = append(normals, n)
		origins = append(origins, pl.Origin)
	}
	return planes, normals, origins, true
}
