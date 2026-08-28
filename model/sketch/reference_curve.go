// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Concrete reference entities (ADR-0055 phase 3). A projected model edge becomes a real sketch
// Line/Circle/Arc/Ellipse/EllipticalArc — grounded (built on fixed refPts + the reference flag, so
// the solver holds it fixed) yet a first-class curve that constraints, dimensions, offset, and
// profiles use natively, with no projected-curve special-casing. This is the SHAPER/Inventor model:
// a projection is a driven reference copy of the source edge's own analytic curve type.

// addReferenceCurve builds the grounded reference entity for a projected analytic 2D curve, or nil
// for a form with no concrete entity yet (a multi-edge/free-form projection keeps a polyline).
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
