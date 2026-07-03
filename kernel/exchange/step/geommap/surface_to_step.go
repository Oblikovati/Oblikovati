// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// stepSurfaceWriter emits a STEP entity for one analytic surface kind, returning its id.
type stepSurfaceWriter func(e *Emitter, s geom.Surface) (int, error)

// stepSurfaceWriters is the table-driven routing from surface kind to STEP emitter (audit
// I6): coverage is a map-keys check against geom.SurfaceKinds (TestStepSurfaceWriterCoverage),
// not a switch audit, so a new geom kind that lacks a writer AND is not declared unsupported
// fails CI instead of falling into a silent default.
var stepSurfaceWriters = map[geom.SurfaceKind]stepSurfaceWriter{
	geom.SurfacePlane:    func(e *Emitter, s geom.Surface) (int, error) { return e.planeToStep(s.(geom.Plane)), nil },
	geom.SurfaceCylinder: func(e *Emitter, s geom.Surface) (int, error) { return e.cylinderToStep(s.(geom.Cylinder)), nil },
	geom.SurfaceCone:     func(e *Emitter, s geom.Surface) (int, error) { return e.coneToStep(s.(geom.Cone)), nil },
	geom.SurfaceSphere:   func(e *Emitter, s geom.Surface) (int, error) { return e.sphereToStep(s.(geom.Sphere)), nil },
	geom.SurfaceTorus:    func(e *Emitter, s geom.Surface) (int, error) { return e.torusToStep(s.(geom.Torus)), nil },
}

// SurfaceToStep emits a STEP surface entity for s, returning its id. The analytic surfaces
// with a STEP entity map exactly; a surface kind without a writer errors, naming the kind
// and consumer (the caller may then fall back to a tessellated face — deferred to PBI-E).
func (e *Emitter) SurfaceToStep(s geom.Surface) (int, error) {
	ks, ok := s.(geom.KindedSurface)
	if !ok {
		return 0, fmt.Errorf("geommap.SurfaceToStep: surface type %T is not a geom.KindedSurface", s)
	}
	write, ok := stepSurfaceWriters[ks.Kind()]
	if !ok {
		return 0, fmt.Errorf("geommap.SurfaceToStep: no STEP writer for surface kind %v (%T) — analytic export is table-driven; add a writer or fall back to tessellation", ks.Kind(), s)
	}
	return write(e, s)
}

// planeToStep emits PLANE with a placement at the plane origin, Z=normal, X=UAxis.
func (e *Emitter) planeToStep(p geom.Plane) int {
	place := e.Placement(p.Origin, p.Normal(), p.UAxis.AsVector())
	return e.w.Add("PLANE", part21.QuoteString(""), part21.Ref(place))
}

// cylinderToStep emits CYLINDRICAL_SURFACE(placement, radius).
func (e *Emitter) cylinderToStep(c geom.Cylinder) int {
	place := e.Placement(c.Origin, c.AxisDir.AsVector(), c.Ref.AsVector())
	return e.w.Add("CYLINDRICAL_SURFACE", part21.QuoteString(""), part21.Ref(place), e.LengthValue(c.Radius))
}

// coneToStep emits CONICAL_SURFACE(placement, base_radius, half_angle). STEP's
// placement is at the cone's base, so the origin is the apex walked forward along
// the axis by base_radius/tan(half_angle); base_radius is chosen as 1 unit of slant
// to give a well-defined placement (the surface is apex-anchored, so any base works;
// 1 mm keeps numbers small).
func (e *Emitter) coneToStep(c geom.Cone) int {
	const baseDist = 1.0 // mm along the axis from apex to the reference base circle
	baseRadius := baseDist * stdmath.Tan(c.HalfAngle)
	baseOrigin := c.Apex.TranslateBy(c.AxisDir.AsVector().Scale(baseDist))
	place := e.Placement(baseOrigin, c.AxisDir.AsVector(), c.Ref.AsVector())
	return e.w.Add("CONICAL_SURFACE", part21.QuoteString(""), part21.Ref(place),
		e.LengthValue(baseRadius), part21.FormatReal(c.HalfAngle))
}

// sphereToStep emits SPHERICAL_SURFACE(placement, radius).
func (e *Emitter) sphereToStep(s geom.Sphere) int {
	place := e.Placement(s.Center, math.V3(0, 0, 1), math.V3(1, 0, 0))
	return e.w.Add("SPHERICAL_SURFACE", part21.QuoteString(""), part21.Ref(place), e.LengthValue(s.Radius))
}

// torusToStep emits TOROIDAL_SURFACE(placement, major_radius, minor_radius).
func (e *Emitter) torusToStep(t geom.Torus) int {
	place := e.Placement(t.Center, t.AxisDir.AsVector(), t.Ref.AsVector())
	return e.w.Add("TOROIDAL_SURFACE", part21.QuoteString(""), part21.Ref(place),
		e.LengthValue(t.MajorRadius), e.LengthValue(t.MinorRadius))
}
