// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"fmt"
	stdmath "math"

	"oblikovati/kernel/exchange/step/part21"
	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// ErrUnsupportedSurface signals a STEP surface type with no kernel analogue. The
// reader records it as a warning and falls back rather than aborting the import.
type ErrUnsupportedSurface struct {
	Keyword string
	ID      int
}

func (e ErrUnsupportedSurface) Error() string {
	return fmt.Sprintf("geommap: unsupported surface %s (#%d)", e.Keyword, e.ID)
}

// Surface maps a STEP surface entity to a geom.Surface. The analytic surfaces
// (PLANE, *_SURFACE) map exactly; B_SPLINE_SURFACE_WITH_KNOTS expands its knots;
// anything else returns ErrUnsupportedSurface so the caller can fall back.
func Surface(g *part21.EntityGraph, id int, scale float64) (geom.Surface, error) {
	ent, err := g.Lookup(id)
	if err != nil {
		return nil, err
	}
	if len(ent.Components) > 0 { // complex instance, e.g. a rational (weighted) B-spline
		return rationalBSplineSurface(g, ent, scale)
	}
	switch ent.Keyword {
	case "PLANE":
		return planeFromStep(g, ent, scale)
	case "CYLINDRICAL_SURFACE":
		return cylinderFromStep(g, ent, scale)
	case "CONICAL_SURFACE":
		return coneFromStep(g, ent, scale)
	case "SPHERICAL_SURFACE":
		return sphereFromStep(g, ent, scale)
	case "TOROIDAL_SURFACE":
		return torusFromStep(g, ent, scale)
	case "B_SPLINE_SURFACE_WITH_KNOTS":
		return bsplineSurfaceFromStep(g, ent, scale)
	default:
		return nil, ErrUnsupportedSurface{Keyword: ent.Keyword, ID: id}
	}
}

// surfaceFrame reads the AXIS2_PLACEMENT_3D at parameter 1 (shared by every
// analytic surface), returning its frame.
func surfaceFrame(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (Frame, error) {
	ref, err := refParam(ent.Params, 1)
	if err != nil {
		return Frame{}, fmt.Errorf("geommap: %s placement: %w", ent.Keyword, err)
	}
	return Placement(g, ref, scale)
}

// planeFromStep maps PLANE(name, placement) → geom.Plane.
func planeFromStep(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (geom.Surface, error) {
	f, err := surfaceFrame(g, ent, scale)
	if err != nil {
		return nil, err
	}
	p, err := geom.NewPlaneFromAxes(f.Origin, f.AxisX, f.AxisZ.Cross(f.AxisX))
	return p, err
}

// cylinderFromStep maps CYLINDRICAL_SURFACE(name, placement, radius). The radius
// is a length, so it scales to mm.
func cylinderFromStep(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (geom.Surface, error) {
	f, err := surfaceFrame(g, ent, scale)
	if err != nil {
		return nil, err
	}
	radius, err := floatParam(ent.Params, 2)
	if err != nil {
		return nil, fmt.Errorf("geommap: CYLINDRICAL_SURFACE radius: %w", err)
	}
	return geom.NewCylinder(f.Origin, f.AxisZ, radius*scale)
}

// coneFromStep maps CONICAL_SURFACE(name, placement, radius, half_angle). STEP's
// placement origin lies at the cone's base; the kernel cone is apex-based, so the
// apex is derived by walking back along the axis by radius/tan(half_angle).
func coneFromStep(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (geom.Surface, error) {
	f, err := surfaceFrame(g, ent, scale)
	if err != nil {
		return nil, err
	}
	radius, halfAngle, err := coneParams(ent, scale)
	if err != nil {
		return nil, err
	}
	apex := apexFromBase(f, radius, halfAngle)
	return geom.NewCone(apex, f.AxisZ, halfAngle)
}

// coneParams reads a cone's base radius (scaled) and half angle (radians).
func coneParams(ent *part21.RawEntity, scale float64) (radius, halfAngle float64, err error) {
	if radius, err = floatParam(ent.Params, 2); err != nil {
		return 0, 0, fmt.Errorf("geommap: CONICAL_SURFACE radius: %w", err)
	}
	if halfAngle, err = floatParam(ent.Params, 3); err != nil {
		return 0, 0, fmt.Errorf("geommap: CONICAL_SURFACE half_angle: %w", err)
	}
	return radius * scale, halfAngle, nil
}

// sphereFromStep maps SPHERICAL_SURFACE(name, placement, radius).
func sphereFromStep(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (geom.Surface, error) {
	f, err := surfaceFrame(g, ent, scale)
	if err != nil {
		return nil, err
	}
	radius, err := floatParam(ent.Params, 2)
	if err != nil {
		return nil, fmt.Errorf("geommap: SPHERICAL_SURFACE radius: %w", err)
	}
	return geom.NewSphere(f.Origin, radius*scale)
}

// torusFromStep maps TOROIDAL_SURFACE(name, placement, major_radius, minor_radius).
func torusFromStep(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (geom.Surface, error) {
	f, err := surfaceFrame(g, ent, scale)
	if err != nil {
		return nil, err
	}
	majorR, err := floatParam(ent.Params, 2)
	if err != nil {
		return nil, fmt.Errorf("geommap: TOROIDAL_SURFACE major_radius: %w", err)
	}
	minorR, err := floatParam(ent.Params, 3)
	if err != nil {
		return nil, fmt.Errorf("geommap: TOROIDAL_SURFACE minor_radius: %w", err)
	}
	return geom.NewTorus(f.Origin, f.AxisZ, majorR*scale, minorR*scale)
}

// apexFromBase returns the cone apex: the base origin walked back along the axis by
// the base radius over tan(half_angle) (the slant height's axial projection).
func apexFromBase(f Frame, baseRadius, halfAngle float64) math.Point3 {
	axis := f.AxisZ.Scale(1 / f.AxisZ.Length())
	backoff := baseRadius / stdmath.Tan(halfAngle)
	return f.Origin.TranslateBy(axis.Scale(-backoff))
}
