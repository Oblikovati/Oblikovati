// SPDX-License-Identifier: GPL-2.0-only

package geomapi

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
)

// The surface adapters: each wraps a kernel surface and speaks the contract's
// value types.

// surface supplies the umbrella members from the embedded kernel surface.
type surface struct {
	kind  types.SurfaceType
	form  types.SurfaceGeometryForm
	inner geom.Surface
}

func (s surface) SurfaceType() types.SurfaceType          { return s.kind }
func (s surface) GeometryForm() types.SurfaceGeometryForm { return s.form }
func (s surface) Evaluate(u, v float64) types.Point       { return toPoint(s.inner.PointAt(u, v)) }
func (s surface) Normal(u, v float64) types.Vector        { return toVector(s.inner.NormalAt(u, v)) }

func (s surface) Domains() (uLo, uHi, vLo, vHi float64) {
	uLo, uHi = s.inner.UDomain()
	vLo, vHi = s.inner.VDomain()
	return uLo, uHi, vLo, vHi
}

func (s surface) Parameter(p types.Point) (u, v float64) {
	return s.inner.ParamAt(fromPoint(p))
}

func analyticSurface(kind types.SurfaceType, inner geom.Surface) surface {
	return surface{kind: kind, form: types.SurfaceFormNotNURBS, inner: inner}
}

// planeAdapter — contract.Plane over geom.Plane.
type planeAdapter struct {
	surface
	g geom.Plane
}

var _ contract.Plane = planeAdapter{}

func newPlane(g geom.Plane) planeAdapter {
	return planeAdapter{surface: analyticSurface(types.PlaneSurface, g), g: g}
}

func (a planeAdapter) RootPoint() types.Point  { return toPoint(a.g.Origin) }
func (a planeAdapter) UAxis() types.UnitVector { return toUnit(a.g.UAxis) }
func (a planeAdapter) VAxis() types.UnitVector { return toUnit(a.g.VAxis) }

func (a planeAdapter) PlaneNormal() types.UnitVector {
	// The kernel derives the normal from the U/V basis (orthonormal), so the
	// normalized cross product is exactly unit.
	n := toUnit(a.g.UAxis).Cross(toUnit(a.g.VAxis))
	return types.UnitVector(n)
}

// cylinderAdapter — contract.Cylinder over geom.Cylinder.
type cylinderAdapter struct {
	surface
	g geom.Cylinder
}

var _ contract.Cylinder = cylinderAdapter{}

func newCylinder(g geom.Cylinder) cylinderAdapter {
	return cylinderAdapter{surface: analyticSurface(types.CylinderSurface, g), g: g}
}

func (a cylinderAdapter) BasePoint() types.Point { return toPoint(a.g.Origin) }
func (a cylinderAdapter) Axis() types.UnitVector { return toUnit(a.g.AxisDir) }
func (a cylinderAdapter) Radius() float64        { return a.g.Radius }

// coneAdapter — contract.Cone over geom.Cone.
type coneAdapter struct {
	surface
	g geom.Cone
}

var _ contract.Cone = coneAdapter{}

func newCone(g geom.Cone) coneAdapter {
	return coneAdapter{surface: analyticSurface(types.ConeSurface, g), g: g}
}

func (a coneAdapter) ApexPoint() types.Point { return toPoint(a.g.Apex) }
func (a coneAdapter) Axis() types.UnitVector { return toUnit(a.g.AxisDir) }
func (a coneAdapter) HalfAngle() float64     { return a.g.HalfAngle }

// sphereAdapter — contract.Sphere over geom.Sphere.
type sphereAdapter struct {
	surface
	g geom.Sphere
}

var _ contract.Sphere = sphereAdapter{}

func newSphere(g geom.Sphere) sphereAdapter {
	return sphereAdapter{surface: analyticSurface(types.SphereSurface, g), g: g}
}

func (a sphereAdapter) Center() types.Point { return toPoint(a.g.Center) }
func (a sphereAdapter) Radius() float64     { return a.g.Radius }

// torusAdapter — contract.Torus over geom.Torus.
type torusAdapter struct {
	surface
	g geom.Torus
}

var _ contract.Torus = torusAdapter{}

func newTorus(g geom.Torus) torusAdapter {
	return torusAdapter{surface: analyticSurface(types.TorusSurface, g), g: g}
}

func (a torusAdapter) Center() types.Point    { return toPoint(a.g.Center) }
func (a torusAdapter) Axis() types.UnitVector { return toUnit(a.g.AxisDir) }
func (a torusAdapter) MajorRadius() float64   { return a.g.MajorRadius }
func (a torusAdapter) MinorRadius() float64   { return a.g.MinorRadius }

// ellipticalCylinderAdapter — contract.EllipticalCylinder over the kernel type.
type ellipticalCylinderAdapter struct {
	surface
	g geom.EllipticalCylinder
}

var _ contract.EllipticalCylinder = ellipticalCylinderAdapter{}

func newEllipticalCylinder(g geom.EllipticalCylinder) ellipticalCylinderAdapter {
	return ellipticalCylinderAdapter{surface: analyticSurface(types.EllipticalCylinderSurface, g), g: g}
}

func (a ellipticalCylinderAdapter) BasePoint() types.Point      { return toPoint(a.g.Origin) }
func (a ellipticalCylinderAdapter) Axis() types.UnitVector      { return toUnit(a.g.AxisDir) }
func (a ellipticalCylinderAdapter) MajorAxis() types.UnitVector { return toUnit(a.g.Ref) }
func (a ellipticalCylinderAdapter) MajorRadius() float64        { return a.g.MajorRadius }
func (a ellipticalCylinderAdapter) MinorRadius() float64        { return a.g.MinorRadius }

// ellipticalConeAdapter — contract.EllipticalCone over the kernel type.
type ellipticalConeAdapter struct {
	surface
	g geom.EllipticalCone
}

var _ contract.EllipticalCone = ellipticalConeAdapter{}

func newEllipticalCone(g geom.EllipticalCone) ellipticalConeAdapter {
	return ellipticalConeAdapter{surface: analyticSurface(types.EllipticalConeSurface, g), g: g}
}

func (a ellipticalConeAdapter) ApexPoint() types.Point      { return toPoint(a.g.Apex) }
func (a ellipticalConeAdapter) Axis() types.UnitVector      { return toUnit(a.g.AxisDir) }
func (a ellipticalConeAdapter) MajorAxis() types.UnitVector { return toUnit(a.g.Ref) }
func (a ellipticalConeAdapter) MajorHalfAngle() float64     { return a.g.MajorAngle }
func (a ellipticalConeAdapter) MinorHalfAngle() float64     { return a.g.MinorAngle }

// bsplineSurfaceAdapter — contract.BSplineSurface over geom.BSplineSurface.
type bsplineSurfaceAdapter struct {
	surface
	g geom.BSplineSurface
}

var _ contract.BSplineSurface = bsplineSurfaceAdapter{}

func newBSplineSurface(g geom.BSplineSurface) bsplineSurfaceAdapter {
	return bsplineSurfaceAdapter{
		surface: surface{kind: types.BSplineSurfaceKind, form: types.SurfaceFormNURBS, inner: g},
		g:       g,
	}
}

func (a bsplineSurfaceAdapter) Definition() types.BSplineSurfaceDef {
	rows, cols := len(a.g.Ctrl), 0
	if rows > 0 {
		cols = len(a.g.Ctrl[0])
	}
	def := types.BSplineSurfaceDef{
		DegreeU: a.g.UDegree, DegreeV: a.g.VDegree,
		PolesU: rows, PolesV: cols,
		KnotsU: append([]float64(nil), a.g.UKnots...),
		KnotsV: append([]float64(nil), a.g.VKnots...),
	}
	for _, row := range a.g.Ctrl {
		def.Poles = append(def.Poles, toPoints(row)...)
	}
	for _, row := range a.g.Weights {
		def.Weights = append(def.Weights, row...)
	}
	return def
}
