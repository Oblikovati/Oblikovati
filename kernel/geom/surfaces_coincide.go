// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Two surfaces that are the SAME surface — the degenerate-overlap case (ADR-0045), where the contact
// between two solids is a two-dimensional REGION rather than a curve. Two coaxial cylinders of one
// radius meet like that, and so do two flush planes; the boolean must then cover the shared region
// once, from one side, rather than look for a crossing that does not exist.
//
// It is the sibling of [SurfacesApart] and answers with the same discipline: exact and ONE-SIDED. True
// means PROVEN the same surface; false means "no proof", never "they differ". The pairs it proves are
// the analytic primitives, whose identity is a finite set of parameters; a B-spline or an offset
// surface returns false whatever its control net says.

// SurfacesCoincide reports whether a and b are the same surface to the given resolution. Orientation is
// not part of the question: a surface has no side of its own, and which side each FACE keeps is the
// caller's ON/ON table to decide.
//
// Example:
//
//	if geom.SurfacesCoincide(wallA, wallB, res) { /* a 2-D contact, not a crossing */ }
func SurfacesCoincide(a, b Surface, res Resolution) bool {
	switch x := a.(type) {
	case Plane:
		y, ok := b.(Plane)
		return ok && planesCoincide(x, y, res)
	case Cylinder:
		y, ok := b.(Cylinder)
		return ok && cylindersCoincide(x, y, res)
	case Cone:
		y, ok := b.(Cone)
		return ok && conesCoincide(x, y, res)
	case Sphere:
		y, ok := b.(Sphere)
		return ok && spheresCoincide(x, y, res)
	case Torus:
		y, ok := b.(Torus)
		return ok && toriCoincide(x, y, res)
	}
	return false
}

// planesCoincide: parallel normals and no offset between the origins along one of them.
func planesCoincide(a, b Plane, res Resolution) bool {
	return parallelDirs(a.Normal(), b.Normal()) &&
		stdmath.Abs(float64(a.Origin.VectorTo(b.Origin).Dot(a.Normal()))) <= res.Plane()
}

// cylindersCoincide: one axis LINE and one radius. The origins need not agree — a cylinder's origin is
// only where its v runs from — so the test is that b's origin lies on a's axis line.
func cylindersCoincide(a, b Cylinder, res Resolution) bool {
	return parallelDirs(a.AxisDir.AsVector(), b.AxisDir.AsVector()) &&
		stdmath.Abs(a.Radius-b.Radius) <= res.Weld() &&
		axisLineDistance(a.Origin, a.AxisDir, b.Origin) <= res.Weld()
}

// conesCoincide: one apex, one axis LINE, one half-angle. The axis SENSE matters here where it does not
// for a cylinder: a cone opens one way, and the same apex and half-angle about the opposite direction
// is the other nappe, a different surface.
func conesCoincide(a, b Cone, res Resolution) bool {
	return float64(a.Apex.DistanceTo(b.Apex)) <= res.Weld() &&
		float64(a.AxisDir.AsVector().Dot(b.AxisDir.AsVector())) > 1-coneAngleWeld &&
		stdmath.Abs(a.HalfAngle-b.HalfAngle) <= coneAngleWeld
}

// spheresCoincide: one centre, one radius.
func spheresCoincide(a, b Sphere, res Resolution) bool {
	return float64(a.Center.DistanceTo(b.Center)) <= res.Weld() && stdmath.Abs(a.Radius-b.Radius) <= res.Weld()
}

// toriCoincide: one centre, one axis LINE, and both radii. The axis sense does not matter — a torus is
// symmetric about its own plane — so the axes need only be parallel.
func toriCoincide(a, b Torus, res Resolution) bool {
	return float64(a.Center.DistanceTo(b.Center)) <= res.Weld() &&
		parallelDirs(a.AxisDir.AsVector(), b.AxisDir.AsVector()) &&
		stdmath.Abs(a.MajorRadius-b.MajorRadius) <= res.Weld() &&
		stdmath.Abs(a.MinorRadius-b.MinorRadius) <= res.Weld()
}

// SurfaceNormalAt is the surface's unit normal at the point nearest p — the datum a coincident-face
// tie-break reads to tell a SHARED contact (normals agreeing) from an ANTI-shared one (normals
// opposing), without the caller having to invert p onto the surface itself.
//
// Example: same := geom.SurfaceNormalAt(a, p).Dot(geom.SurfaceNormalAt(b, p)) > 0
func SurfaceNormalAt(s Surface, p math.Point3) math.Vector3 {
	u, v := s.ParamAt(p)
	return s.NormalAt(u, v)
}
