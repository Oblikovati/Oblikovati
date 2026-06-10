// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// These are the surface-tangent work-plane definitions — Inventor's WorkPlanes
// constructors that build on a B-rep face (a cylinder/cone/sphere/torus) rather than on
// another work feature. The face is named by a face [WorkRef] (work_surface_ref.go),
// re-bound to the running body each recompute. The analytic surfaces expose their axis/
// centre/radius (kernel/geom), so the tangent geometry is closed-form; sub-cases that
// have no closed-form single answer (e.g. a plane tangent to a torus parallel to a
// reference plane) report a Sick reason rather than guessing, consistent with the rest
// of the work-feature engine (degenerate input → sick, never garbage).

// torusMidPlaneDef is the mid-plane of a torus: through its centre, normal to its axis
// (Inventor's AddByTorusMidPlane).
type torusMidPlaneDef struct{ face WorkRef }

func (d *torusMidPlaneDef) kindName() string { return "torus-midplane" }
func (d *torusMidPlaneDef) refs() []WorkRef  { return []WorkRef{d.face} }
func (d *torusMidPlaneDef) eval(r workResolver) (sketch.Plane, error) {
	s, err := r.surface(d.face)
	if err != nil {
		return sketch.Plane{}, err
	}
	t, ok := s.(geom.Torus)
	if !ok {
		return sketch.Plane{}, fmt.Errorf("torus mid-plane needs a torus face, got %T", s)
	}
	return planeFromOriginNormal(t.Center, t.AxisDir)
}

// pointAndTangentPlaneDef is the surface's tangent plane at a point on it (Inventor's
// AddByPointAndTangent): through the point, normal = the surface normal there.
type pointAndTangentPlaneDef struct{ point, face WorkRef }

func (d *pointAndTangentPlaneDef) kindName() string { return "point-tangent" }
func (d *pointAndTangentPlaneDef) refs() []WorkRef  { return []WorkRef{d.point, d.face} }
func (d *pointAndTangentPlaneDef) eval(r workResolver) (sketch.Plane, error) {
	p, err := r.point(d.point)
	if err != nil {
		return sketch.Plane{}, err
	}
	s, err := r.surface(d.face)
	if err != nil {
		return sketch.Plane{}, err
	}
	n, err := surfaceNormalAtPoint(s, p)
	if err != nil {
		return sketch.Plane{}, err
	}
	return planeFromOriginNormal(p, n)
}

// planeAndTangentPlaneDef is parallel to a reference plane and tangent to the surface
// (Inventor's AddByPlaneAndTangent). Closed-form for cylinders and spheres; other
// surfaces report a Sick reason.
type planeAndTangentPlaneDef struct{ base, face WorkRef }

func (d *planeAndTangentPlaneDef) kindName() string { return "plane-tangent" }
func (d *planeAndTangentPlaneDef) refs() []WorkRef  { return []WorkRef{d.base, d.face} }
func (d *planeAndTangentPlaneDef) eval(r workResolver) (sketch.Plane, error) {
	base, err := r.plane(d.base)
	if err != nil {
		return sketch.Plane{}, err
	}
	s, err := r.surface(d.face)
	if err != nil {
		return sketch.Plane{}, err
	}
	return planeParallelTangent(base, s)
}

// lineAndTangentPlaneDef holds a line and is tangent to the surface (Inventor's
// AddByLineAndTangent). Closed-form for a cylindrical face whose axis is parallel to the
// line (the canonical case — a datum through an edge, tangent to a round); other
// configurations report a Sick reason.
type lineAndTangentPlaneDef struct{ line, face WorkRef }

func (d *lineAndTangentPlaneDef) kindName() string { return "line-tangent" }
func (d *lineAndTangentPlaneDef) refs() []WorkRef  { return []WorkRef{d.line, d.face} }
func (d *lineAndTangentPlaneDef) eval(r workResolver) (sketch.Plane, error) {
	line, err := r.axis(d.line)
	if err != nil {
		return sketch.Plane{}, err
	}
	s, err := r.surface(d.face)
	if err != nil {
		return sketch.Plane{}, err
	}
	return planeThroughLineTangent(line, s)
}

// AddByTorusMidPlane creates the mid-plane of the torus face.
//
//	wp := planes.AddByTorusMidPlane(feature.FaceRef(torusFace.ReferenceKey()))
func (c *WorkPlanes) AddByTorusMidPlane(face WorkRef) *WorkPlane {
	return c.addUser(&torusMidPlaneDef{face: face})
}

// AddByPointAndTangent creates the tangent plane of a surface at a point on it.
//
//	wp := planes.AddByPointAndTangent(p.Key(), feature.FaceRef(cylFace.ReferenceKey()))
func (c *WorkPlanes) AddByPointAndTangent(point, face WorkRef) *WorkPlane {
	return c.addUser(&pointAndTangentPlaneDef{point: point, face: face})
}

// AddByPlaneAndTangent creates a plane parallel to base and tangent to the surface
// (cylinder or sphere).
//
//	wp := planes.AddByPlaneAndTangent(feature.OriginXZPlane, feature.FaceRef(cylKey))
func (c *WorkPlanes) AddByPlaneAndTangent(base, face WorkRef) *WorkPlane {
	return c.addUser(&planeAndTangentPlaneDef{base: base, face: face})
}

// AddByLineAndTangent creates a plane through line and tangent to the surface (a
// cylinder whose axis is parallel to the line).
//
//	wp := planes.AddByLineAndTangent(edgeAxis.Key(), feature.FaceRef(cylKey))
func (c *WorkPlanes) AddByLineAndTangent(line, face WorkRef) *WorkPlane {
	return c.addUser(&lineAndTangentPlaneDef{line: line, face: face})
}

// FaceRef encodes a B-rep face's persistent reference key as a WorkRef, for the
// surface-tangent constructors above. It is the public spelling of the internal
// face-reference encoding (work_surface_ref.go).
func FaceRef(key []byte) WorkRef { return faceRef(key) }

// surfaceNormalAtPoint returns the outward unit normal of an analytic surface at a point
// lying on it. Each case derives the normal from the surface's axis/centre, so the point
// need not be parameter-inverted.
func surfaceNormalAtPoint(s geom.Surface, p math.Point3) (math.UnitVector3, error) {
	switch g := s.(type) {
	case geom.Cylinder:
		return outwardRadial(g.Origin, g.AxisDir, p)
	case geom.Sphere:
		return math.UnitVector3FromVector(g.Center.VectorTo(p))
	case geom.Cone:
		return coneNormalAtPoint(g, p)
	case geom.Torus:
		return torusNormalAtPoint(g, p)
	default:
		return math.UnitVector3{}, fmt.Errorf("tangent work plane: unsupported surface %T", s)
	}
}

// outwardRadial returns the unit vector from the axis (origin, dir) to p, perpendicular
// to the axis — the surface normal of a cylinder at p.
func outwardRadial(origin math.Point3, dir math.UnitVector3, p math.Point3) (math.UnitVector3, error) {
	rel := origin.VectorTo(p)
	foot := origin.TranslateBy(dir.AsVector().Scale(rel.Dot(dir.AsVector())))
	return math.UnitVector3FromVector(foot.VectorTo(p))
}

// coneNormalAtPoint returns the cone's outward normal at p: cos(half)·radial − sin(half)·axis.
func coneNormalAtPoint(c geom.Cone, p math.Point3) (math.UnitVector3, error) {
	radial, err := outwardRadial(c.Apex, c.AxisDir, p)
	if err != nil {
		return math.UnitVector3{}, err
	}
	cosH, sinH := stdmath.Cos(c.HalfAngle), stdmath.Sin(c.HalfAngle)
	n := radial.AsVector().Scale(cosH).Sub(c.AxisDir.AsVector().Scale(sinH))
	return math.UnitVector3FromVector(n)
}

// torusNormalAtPoint returns the torus normal at p: the unit vector from the nearest
// tube-centreline point (centre + Major·radial) to p.
func torusNormalAtPoint(t geom.Torus, p math.Point3) (math.UnitVector3, error) {
	rel := t.Center.VectorTo(p)
	planar := rel.Sub(t.AxisDir.AsVector().Scale(rel.Dot(t.AxisDir.AsVector())))
	radial, err := math.UnitVector3FromVector(planar)
	if err != nil {
		return math.UnitVector3{}, fmt.Errorf("torus tangent: point is on the torus axis")
	}
	tubeCenter := t.Center.TranslateBy(radial.AsVector().Scale(t.MajorRadius))
	return math.UnitVector3FromVector(tubeCenter.VectorTo(p))
}

// planeParallelTangent returns the plane parallel to base (same normal) that is tangent
// to the surface. For a cylinder the reference normal must be perpendicular to the axis
// (otherwise no parallel plane is tangent along a line); the tangent point is on the +N
// side of the axis/centre (the opposite tangent is the −N plane).
func planeParallelTangent(base sketch.Plane, s geom.Surface) (sketch.Plane, error) {
	n := base.Normal().AsVector()
	switch g := s.(type) {
	case geom.Cylinder:
		if !math.IsNearZero(n.Dot(g.AxisDir.AsVector()), math.DefaultTolerance) {
			return sketch.Plane{}, fmt.Errorf("reference plane is not parallel to the cylinder axis")
		}
		origin := g.Origin.TranslateBy(n.Scale(g.Radius))
		return sketch.NewPlane(origin, base.XAxis(), base.YAxis())
	case geom.Sphere:
		origin := g.Center.TranslateBy(n.Scale(g.Radius))
		return sketch.NewPlane(origin, base.XAxis(), base.YAxis())
	default:
		return sketch.Plane{}, fmt.Errorf("plane tangent to a %T is supported for cylinders and spheres only", s)
	}
}

// planeThroughLineTangent returns the plane that holds the line and is tangent to the
// surface. Supported for a cylinder whose axis is parallel to the line; the tangent is
// solved in the cross-section (a tangent line from the line's projection to the radius
// circle), so the line must lie outside the cylinder.
func planeThroughLineTangent(line *WorkAxis, s geom.Surface) (sketch.Plane, error) {
	g, ok := s.(geom.Cylinder)
	if !ok {
		return sketch.Plane{}, fmt.Errorf("tangent-to-line work plane supports cylindrical faces only, got %T", s)
	}
	if !line.Direction().IsParallelTo(g.AxisDir, math.DefaultTolerance) {
		return sketch.Plane{}, fmt.Errorf("the line must be parallel to the cylinder axis")
	}
	a := line.Origin()
	normal, err := cylinderTangentNormal(a, g)
	if err != nil {
		return sketch.Plane{}, err
	}
	return planeFromOriginNormal(a, normal)
}

// cylinderTangentNormal returns the normal of the plane that holds the axis-parallel line
// through a and is tangent to the cylinder — solved in the cross-section as a tangent line
// from a's projection to the radius circle (the line must lie outside the cylinder).
func cylinderTangentNormal(a math.Point3, g geom.Cylinder) (math.UnitVector3, error) {
	rel := g.Origin.VectorTo(a)
	perp := rel.Sub(g.AxisDir.AsVector().Scale(rel.Dot(g.AxisDir.AsVector())))
	d := perp.Length()
	if d < g.Radius-math.DefaultTolerance {
		return math.UnitVector3{}, fmt.Errorf("the line is inside the cylinder (offset %g < radius %g); no tangent plane", d, g.Radius)
	}
	u, err := math.UnitVector3FromVector(perp)
	if err != nil {
		return math.UnitVector3{}, fmt.Errorf("the line is on the cylinder axis; the tangent plane is undefined")
	}
	w := g.AxisDir.Cross(u)
	cosA := g.Radius / d
	sinA := stdmath.Sqrt(stdmath.Max(0, 1-cosA*cosA))
	return math.UnitVector3FromVector(u.AsVector().Scale(cosA).Add(w.Scale(sinA)))
}
