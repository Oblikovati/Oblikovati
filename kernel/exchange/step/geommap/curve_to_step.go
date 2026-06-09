// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"fmt"

	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// CurveToStep emits the STEP curve for a kernel edge curve and returns its id plus
// the EDGE_CURVE.same_sense flag relating the STEP curve's natural direction to the
// kernel edge's start→end direction. A LINE/CIRCLE is always emitted same_sense
// agreeing with start→end, except an Arc3d swept clockwise emits same_sense=.F.
// (the CIRCLE's natural direction is CCW about its normal).
func (e *Emitter) CurveToStep(c geom.Curve3) (id int, sameSense bool, err error) {
	switch v := c.(type) {
	case geom.LineSegment:
		return e.lineToStep(v.StartPoint, v.StartPoint.VectorTo(v.EndPoint)), true, nil
	case geom.Line:
		return e.lineToStep(v.Origin, v.Dir.AsVector()), true, nil
	case geom.Circle:
		return e.circleToStep(v.Center, v.Normal.AsVector(), v.RefDir.AsVector(), v.Radius), true, nil
	case geom.Arc3d:
		return e.arcToStep(v)
	default:
		return 0, false, fmt.Errorf("geommap: cannot export curve type %T to STEP", c)
	}
}

// lineToStep emits LINE(point, VECTOR(direction, 1.0)).
func (e *Emitter) lineToStep(origin math.Point3, dir math.Vector3) int {
	pt := e.Point(origin)
	d := e.Direction(dir)
	vec := e.w.Add("VECTOR", part21.QuoteString(""), part21.Ref(d), part21.FormatReal(1))
	return e.w.Add("LINE", part21.QuoteString(""), part21.Ref(pt), part21.Ref(vec))
}

// circleToStep emits CIRCLE(placement, radius) with the given frame.
func (e *Emitter) circleToStep(center math.Point3, normal, refDir math.Vector3, radius float64) int {
	place := e.Placement(center, normal, refDir)
	return e.w.Add("CIRCLE", part21.QuoteString(""), part21.Ref(place), e.LengthValue(radius))
}

// arcToStep emits the arc's underlying CIRCLE and the same_sense flag. The STEP
// CIRCLE always runs CCW about its normal; an arc with a positive sweep runs CCW
// too (same_sense=.T.), a negative sweep runs CW (same_sense=.F.).
func (e *Emitter) arcToStep(a geom.Arc3d) (int, bool, error) {
	id := e.circleToStep(a.Center, a.Normal.AsVector(), a.RefDir.AsVector(), a.Radius)
	return id, a.SweepAngle >= 0, nil
}
