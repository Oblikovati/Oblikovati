// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// This file holds the surface-derived 3D-sketch curves (M22-F11): curves computed from
// referenced part surfaces/curves rather than from constrainable points. Each wraps the
// F10 surface-intersection kernel (kernel/geom) and produces a polyline (or polylines) on
// Evaluate. They carry no solver DOFs (their geometry is derived); recompute re-evaluates
// them against the current geometry. The face-reference binding that supplies the
// geom.Surface inputs from part faces (reference keys + recompute) is the integration
// layer; this is the computational core, exercised against analytic surfaces.

// defaultCurveSamples is the sampling resolution for projecting/offsetting a source curve.
const defaultCurveSamples = 64

// SurfaceSource is a model entity that yields a surface to derive a curve from (e.g. a
// part face), with a stable SourceID for associative re-derivation across recompute. The
// kernel's faces adapt to this via a self-resolving source (keyed by reference), so a
// surface-derived curve re-evaluates against the freshly rebuilt B-rep.
type SurfaceSource interface {
	SourceID() string
	Surface() geom.Surface
}

// staticSurface wraps a fixed geom.Surface as a non-associative SurfaceSource (direct use
// and tests; SourceID "" ⇒ no re-derivation).
type staticSurface struct{ s geom.Surface }

func (s staticSurface) SourceID() string      { return "" }
func (s staticSurface) Surface() geom.Surface { return s.s }

// StaticSurface adapts a geom.Surface to a SurfaceSource with no associativity.
func StaticSurface(s geom.Surface) SurfaceSource { return staticSurface{s: s} }

// isDerivedCurve3D reports whether an entity is a surface-derived curve. These are
// recompute-derived from referenced geometry, so — like the realized B-rep — they are not
// persisted as geometry; their references rebind on recompute (the integration layer), and
// serialization skips them.
func isDerivedCurve3D(e Entity) bool {
	switch e.(type) {
	case *IntersectionCurve3D, *SilhouetteCurve3D, *ProjectToSurfaceCurve3D, *OnFaceCurve3D, *OffsetCurve3:
		return true
	case *IncludedPoint3D, *IncludedCurve3D:
		return true // included reference geometry rebinds from its source on recompute
	default:
		return false
	}
}

// IntersectionCurve3D is the intersection of two referenced surfaces (e.g. two part
// faces), as one or more polylines on the first surface.
type IntersectionCurve3D struct {
	entityBase
	A, B SurfaceSource
	Grid geom.SurfaceGrid
}

// Evaluate returns the intersection polylines via the F10 surface↔surface tracer,
// re-resolving both surfaces from their sources (so it tracks recompute). A lost reference
// (nil surface) yields no geometry.
func (c *IntersectionCurve3D) Evaluate() [][]math.Point3 {
	a, b := c.A.Surface(), c.B.Surface()
	if a == nil || b == nil {
		return nil
	}
	return geom.IntersectSurfaceSurface(a, b, c.Grid)
}

// SilhouetteCurve3D is the contour generator of a referenced surface for a view direction.
type SilhouetteCurve3D struct {
	entityBase
	Surface SurfaceSource
	ViewDir math.Vector3
	Grid    geom.SurfaceGrid
}

// Evaluate returns the silhouette polylines via the F10 silhouette tracer (nil surface ⇒
// no geometry).
func (c *SilhouetteCurve3D) Evaluate() [][]math.Point3 {
	surface := c.Surface.Surface()
	if surface == nil {
		return nil
	}
	return geom.Silhouette(surface, c.ViewDir, c.Grid)
}

// ProjectToSurfaceCurve3D is a source curve projected onto a referenced surface (the
// perpendicular foot of each sampled point).
type ProjectToSurfaceCurve3D struct {
	entityBase
	Source  geom.Curve3
	Surface SurfaceSource
	Samples int
}

// Evaluate samples the source curve and projects each point onto the surface.
func (c *ProjectToSurfaceCurve3D) Evaluate() []math.Point3 {
	surface := c.Surface.Surface()
	n := samplesOr(c.Samples)
	lo, hi := c.Source.Domain()
	out := make([]math.Point3, n+1)
	for i := range out {
		t := lo + (hi-lo)*float64(i)/float64(n)
		_, _, foot := geom.ClosestPointOnSurface(surface, c.Source.PointAt(t))
		out[i] = foot
	}
	return out
}

// OnFaceCurve3D is a curve drawn in a referenced surface's parameter space, mapped to 3D
// through the surface. UV are the (u, v) parameter samples.
type OnFaceCurve3D struct {
	entityBase
	Surface SurfaceSource
	UV      []math.Point2
}

// Evaluate maps the parameter samples onto the surface.
func (c *OnFaceCurve3D) Evaluate() []math.Point3 {
	surface := c.Surface.Surface()
	out := make([]math.Point3, len(c.UV))
	for i, uv := range c.UV {
		out[i] = surface.PointAt(float64(uv.X), float64(uv.Y))
	}
	return out
}

// OffsetCurve3 is a 3D source curve offset by a signed distance in the plane with the
// given normal (offset direction = normal × tangent).
type OffsetCurve3 struct {
	entityBase
	Source   geom.Curve3
	Distance float64
	Normal   math.Vector3
	Samples  int
}

// Evaluate samples the source curve and offsets each point perpendicular to its tangent
// in the offset plane.
func (c *OffsetCurve3) Evaluate() []math.Point3 {
	n := samplesOr(c.Samples)
	lo, hi := c.Source.Domain()
	out := make([]math.Point3, n+1)
	for i := range out {
		t := lo + (hi-lo)*float64(i)/float64(n)
		out[i] = offsetPoint3(c.Source, t, c.Distance, c.Normal)
	}
	return out
}

// offsetPoint3 offsets the curve point at t by distance along normal × tangent (unit).
func offsetPoint3(curve geom.Curve3, t, distance float64, normal math.Vector3) math.Point3 {
	p := curve.PointAt(t)
	dir := normal.Cross(curve.TangentAt(t))
	length := float64(dir.Length())
	if length < math.DefaultTolerance {
		return p // tangent parallel to the normal: offset undefined, leave the point
	}
	return p.TranslateBy(dir.Scale(math.Scalar(distance / length)))
}

// samplesOr returns n when positive, else the default sampling resolution.
func samplesOr(n int) int {
	if n > 0 {
		return n
	}
	return defaultCurveSamples
}

// AddIntersectionCurve3D adds the intersection of fixed surfaces a and b (non-associative;
// grid windows a's parameter domain for an unbounded base — see kernel/geom.SurfaceGrid).
func (s *Sketch3D) AddIntersectionCurve3D(a, b geom.Surface, grid geom.SurfaceGrid) *IntersectionCurve3D {
	return s.AddIntersectionCurve3DRef(StaticSurface(a), StaticSurface(b), grid)
}

// AddIntersectionCurve3DRef adds the intersection of two surface sources (associative:
// the sources re-resolve their faces by reference key on recompute).
func (s *Sketch3D) AddIntersectionCurve3DRef(a, b SurfaceSource, grid geom.SurfaceGrid) *IntersectionCurve3D {
	c := &IntersectionCurve3D{entityBase: newEntity(), A: a, B: b, Grid: grid}
	s.addEntity3D(c)
	return c
}

// AddSilhouetteCurve3D adds the silhouette of a fixed surface for the given view direction.
func (s *Sketch3D) AddSilhouetteCurve3D(surface geom.Surface, viewDir math.Vector3, grid geom.SurfaceGrid) *SilhouetteCurve3D {
	return s.AddSilhouetteCurve3DRef(StaticSurface(surface), viewDir, grid)
}

// AddSilhouetteCurve3DRef adds the silhouette of a surface source (associative).
func (s *Sketch3D) AddSilhouetteCurve3DRef(surface SurfaceSource, viewDir math.Vector3, grid geom.SurfaceGrid) *SilhouetteCurve3D {
	c := &SilhouetteCurve3D{entityBase: newEntity(), Surface: surface, ViewDir: viewDir, Grid: grid}
	s.addEntity3D(c)
	return c
}

// AddProjectToSurfaceCurve3D adds the projection of source onto a fixed surface.
func (s *Sketch3D) AddProjectToSurfaceCurve3D(source geom.Curve3, surface geom.Surface) *ProjectToSurfaceCurve3D {
	return s.AddProjectToSurfaceCurve3DRef(source, StaticSurface(surface))
}

// AddProjectToSurfaceCurve3DRef projects a source curve onto a surface source (the surface
// is associative and rebinds on recompute; the source curve is a fixed snapshot).
func (s *Sketch3D) AddProjectToSurfaceCurve3DRef(source geom.Curve3, surface SurfaceSource) *ProjectToSurfaceCurve3D {
	c := &ProjectToSurfaceCurve3D{entityBase: newEntity(), Source: source, Surface: surface}
	s.addEntity3D(c)
	return c
}

// SourceCurve3 resolves a curve entity in this sketch (by id) to its kernel curve — the
// source for a project/offset surface curve. It errors if the id is unknown or the entity
// is not a curve.
func (s *Sketch3D) SourceCurve3(id ID) (geom.Curve3, error) {
	e, ok := s.EntityByID(id)
	if !ok {
		return nil, fmt.Errorf("sketch3d: no entity with id %d", id)
	}
	return curve3OfEntity(e)
}

// curve3OfEntity maps a 3D-sketch entity to its kernel curve.
func curve3OfEntity(e Entity) (geom.Curve3, error) {
	switch v := e.(type) {
	case *Line3D:
		return geom.NewLineSegment(v.A.Position(), v.B.Position()), nil
	case *Circle3D:
		return v.Curve()
	case *Arc3D:
		return v.Curve()
	case *Ellipse3D:
		return v.Curve()
	case *EllipticalArc3D:
		return v.Curve()
	case *HelicalCurve3D:
		return v.Curve()
	default:
		return nil, fmt.Errorf("sketch3d: entity %T is not a source curve", e)
	}
}

// AddOnFaceCurve3D adds a curve drawn in a fixed surface's parameter space (the uv samples).
func (s *Sketch3D) AddOnFaceCurve3D(surface geom.Surface, uv []math.Point2) *OnFaceCurve3D {
	return s.AddOnFaceCurve3DRef(StaticSurface(surface), uv)
}

// AddOnFaceCurve3DRef adds an on-face curve over a surface source (associative: it rebinds
// to the referenced face's surface on recompute).
func (s *Sketch3D) AddOnFaceCurve3DRef(surface SurfaceSource, uv []math.Point2) *OnFaceCurve3D {
	c := &OnFaceCurve3D{entityBase: newEntity(), Surface: surface, UV: append([]math.Point2(nil), uv...)}
	s.addEntity3D(c)
	return c
}

// AddOffsetCurve3 adds the offset of source by distance in the plane with the given normal.
func (s *Sketch3D) AddOffsetCurve3(source geom.Curve3, distance float64, normal math.Vector3) *OffsetCurve3 {
	c := &OffsetCurve3{entityBase: newEntity(), Source: source, Distance: distance, Normal: normal}
	s.addEntity3D(c)
	return c
}
