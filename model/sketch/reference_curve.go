// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Concrete reference entities (ADR-0055 phase 3). A projected model edge becomes a real sketch
// Line/Circle/Arc/Ellipse/EllipticalArc — grounded (built on fixed refPts + the reference flag, so
// the solver holds it fixed) yet a first-class curve that constraints, dimensions, offset, and
// profiles use natively, with no projected-curve special-casing. This is the SHAPER/Inventor model:
// a projection is a driven reference copy of the source edge's own analytic curve type. A source
// with no single analytic curve (a multi-edge cut/silhouette loop) becomes a grounded reference
// Spline through the sampled points, so every projection still resolves to one real entity.

// addReferenceCurve builds the grounded reference entity for a projected analytic 2D curve, or nil
// for a form with no concrete entity yet (the caller then builds a reference polyline spline).
func (s *Sketch) addReferenceCurve(c geom.Curve2) Entity {
	switch k := c.(type) {
	case geom.LineSegment2d:
		return asReference(s.lines.Add(s.newRefPoint(k.StartPoint), s.newRefPoint(k.EndPoint)))
	case geom.Circle2d:
		return asReference(s.circles.Add(s.newRefPoint(k.Center), math.Scalar(k.Radius)))
	case geom.Arc2d:
		start := arcPointAt(k.Center, k.Radius, k.StartAngle)
		end := arcPointAt(k.Center, k.Radius, k.StartAngle+k.SweepAngle)
		return asReference(s.arcs.Add(s.newRefPoint(k.Center), s.newRefPoint(start), s.newRefPoint(end), k.SweepAngle > 0))
	case geom.EllipseFull2d:
		return asReference(s.ellipses.AddWithCenter(s.newRefPoint(k.Center), k.MajorAxis.AsVector(), math.Scalar(k.MajorRadius), math.Scalar(k.MinorRadius)))
	case geom.EllipticalArc2d:
		return asReference(s.ellArcs.AddWithCenter(s.newRefPoint(k.Center), k.MajorAxis.AsVector(),
			math.Scalar(k.MajorRadius), math.Scalar(k.MinorRadius), math.Scalar(k.StartAngle), math.Scalar(k.StartAngle+k.SweepAngle)))
	default:
		return nil
	}
}

// addReferencePolyline builds a grounded reference Spline through a projected polyline — the concrete
// entity for a projection whose source has no single analytic curve (a multi-edge cut/silhouette
// loop, ADR-0055). Its nodes are fixed refPts, so the solver holds the whole curve in place.
func (s *Sketch) addReferencePolyline(pts []math.Point2) Entity {
	nodes := make([]*Point, len(pts))
	for i, p := range pts {
		nodes[i] = s.newRefPoint(p)
	}
	sp := s.splines.AddWithPoints(nodes, polylineReturnsToStart(pts), true)
	sp.SetReference(true)
	return sp
}

// buildReferenceEntity rebuilds a projection's concrete reference entity on restore from its
// persisted analytic form (shape+params) or, for a non-analytic projection, its polyline coords
// (ADR-0055). ok is false when neither describes a curve.
func (s *Sketch) buildReferenceEntity(shape string, params, coords []float64) (Entity, bool) {
	if c2, ok := analyticCurveFromData(shape, params); ok {
		if e := s.addReferenceCurve(c2); e != nil {
			return e, true
		}
	}
	if len(coords) > 0 {
		return s.addReferencePolyline(unflattenPoints(coords)), true
	}
	return nil, false
}

// setReferenceGeometry drives a concrete reference entity's geometry to match c IN PLACE, keeping its
// id and defining points (so constraints referencing them survive a recompute — the point of phase 3).
// It returns false when c is a different curve type than ent (a projected circle that became an
// ellipse on an oblique recompute), so the caller rebuilds the entity instead.
func setReferenceGeometry(ent Entity, c geom.Curve2) bool {
	switch e := ent.(type) {
	case *Line:
		return setRefLine(e, c)
	case *Circle:
		return setRefCircle(e, c)
	case *Arc:
		return setRefArc(e, c)
	case *Ellipse:
		return setRefEllipse(e, c)
	case *EllipticalArc:
		return setRefEllipticalArc(e, c)
	default:
		return false
	}
}

func setRefLine(e *Line, c geom.Curve2) bool {
	k, ok := c.(geom.LineSegment2d)
	if !ok {
		return false
	}
	e.A.SetPosition(k.StartPoint)
	e.B.SetPosition(k.EndPoint)
	return true
}

func setRefCircle(e *Circle, c geom.Curve2) bool {
	k, ok := c.(geom.Circle2d)
	if !ok {
		return false
	}
	e.Center.SetPosition(k.Center)
	e.Radius = math.Scalar(k.Radius)
	return true
}

func setRefArc(e *Arc, c geom.Curve2) bool {
	k, ok := c.(geom.Arc2d)
	if !ok {
		return false
	}
	e.Center.SetPosition(k.Center)
	e.Start.SetPosition(arcPointAt(k.Center, k.Radius, k.StartAngle))
	e.End.SetPosition(arcPointAt(k.Center, k.Radius, k.StartAngle+k.SweepAngle))
	e.CounterClockwise = k.SweepAngle > 0
	return true
}

func setRefEllipse(e *Ellipse, c geom.Curve2) bool {
	k, ok := c.(geom.EllipseFull2d)
	if !ok {
		return false
	}
	e.Center.SetPosition(k.Center)
	e.MajorAxis = k.MajorAxis.AsVector()
	e.MajorRadius, e.MinorRadius = math.Scalar(k.MajorRadius), math.Scalar(k.MinorRadius)
	e.seedOrientation() // keep orientation consistent with the new axis (reference ellipses skip solve)
	return true
}

func setRefEllipticalArc(e *EllipticalArc, c geom.Curve2) bool {
	k, ok := c.(geom.EllipticalArc2d)
	if !ok {
		return false
	}
	e.Center.SetPosition(k.Center)
	e.MajorAxis = k.MajorAxis.AsVector()
	e.MajorRadius, e.MinorRadius = math.Scalar(k.MajorRadius), math.Scalar(k.MinorRadius)
	e.StartAngle, e.EndAngle = math.Scalar(k.StartAngle), math.Scalar(k.StartAngle+k.SweepAngle)
	e.seedOrientation()
	return true
}

// entityCurve2 is the inverse of addReferenceCurve: it reads a concrete reference entity's exact
// analytic 2D curve for persistence (ADR-0055). ok is false for a reference Spline (a non-analytic
// projection), which persists its polyline instead.
func entityCurve2(ent Entity) (geom.Curve2, bool) {
	switch e := ent.(type) {
	case *Line:
		return geom.NewLineSegment2d(e.A.Position(), e.B.Position()), true
	case *Circle:
		return geom.NewCircle2d(e.Center.Position(), float64(e.Radius)), true
	case *Arc:
		return arcEntityCurve2(e), true
	case *Ellipse:
		c, err := geom.NewEllipseFull2d(e.Center.Position(), e.MajorAxis, float64(e.MajorRadius), float64(e.MinorRadius))
		return c, err == nil
	case *EllipticalArc:
		return ellipticalArcEntityCurve2(e)
	default:
		return nil, false
	}
}

// arcEntityCurve2 reconstructs a reference arc's geom.Arc2d from its centre, start and end points.
func arcEntityCurve2(a *Arc) geom.Arc2d {
	center := a.Center.Position()
	start, end := a.Start.Position(), a.End.Position()
	radius := center.DistanceTo(start)
	startAngle := angleOf(center, start)
	sweep := signedSweep(startAngle, angleOf(center, end), a.CounterClockwise)
	return geom.NewArc2d(center, float64(radius), startAngle, sweep)
}

// ellipticalArcEntityCurve2 reconstructs a reference elliptical arc's geom.EllipticalArc2d.
func ellipticalArcEntityCurve2(e *EllipticalArc) (geom.Curve2, bool) {
	c, err := geom.NewEllipticalArc2d(e.Center.Position(), e.MajorAxis,
		float64(e.MajorRadius), float64(e.MinorRadius), float64(e.StartAngle), float64(e.EndAngle-e.StartAngle))
	return c, err == nil
}

// angleOf is the angle (radians) of p about centre, measured CCW from +X.
func angleOf(center, p math.Point2) float64 {
	v := center.VectorTo(p)
	return stdmath.Atan2(float64(v.Y), float64(v.X))
}

// signedSweep returns the signed sweep (radians) from start to end in the requested winding, in
// (0, 2π] for a non-degenerate arc so a full-circle-adjacent span keeps its direction.
func signedSweep(start, end float64, ccw bool) float64 {
	sweep := end - start
	if ccw {
		for sweep <= 0 {
			sweep += 2 * stdmath.Pi
		}
		return sweep
	}
	for sweep >= 0 {
		sweep -= 2 * stdmath.Pi
	}
	return sweep
}

// referenceable is the entity capability to be flagged grounded reference geometry.
type referenceable interface {
	Entity
	SetReference(bool)
}

// asReference flags a freshly built entity as grounded reference geometry and returns it.
func asReference[E referenceable](e E) Entity {
	e.SetReference(true)
	return e
}
