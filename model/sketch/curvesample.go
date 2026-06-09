// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Region detection and the extruded cross-section need each curved sketch entity as
// a polyline that follows the real curve, not a chord between its endpoints. The
// endpoint-only walk used before flattened every arc/spline into a straight line, so
// an extrude of a profile with arcs came out wrong (the loop's polygon enclosed no
// area). These samplers replace that.

// curveSamples is how many straight segments a curved entity is approximated with;
// it matches sampleCircle so a circle and a full-turn arc agree in resolution.
const curveSamples = circleSamples

// traversalPolyline returns entity e's polyline walked in loop order — from its near
// endpoint toward its far endpoint — EXCLUDING the far endpoint. The next entity in
// the loop contributes that point as its own start (and the closing entity's far end
// is the loop's first point), so dropping it keeps the loop polygon free of duplicate
// vertices. A straight line yields a single point; a curved entity yields its start
// plus interior samples so the loop follows the curve, not a chord.
func traversalPolyline(e Entity, reversed bool) []math.Point2 {
	pts := naturalPolyline(e)
	if reversed {
		reverseInPlace2(pts)
	}
	if len(pts) == 0 {
		return pts
	}
	return pts[:len(pts)-1]
}

// naturalPolyline samples entity e from its natural start to its natural end,
// endpoints inclusive. Unknown entities degrade to their endpoint chord.
func naturalPolyline(e Entity) []math.Point2 {
	switch t := e.(type) {
	case *Line:
		return []math.Point2{t.A.Position(), t.B.Position()}
	case *Arc:
		return sampleArcEntity(t)
	case *Spline:
		return sampleSplineEntity(t)
	case *EllipticalArc:
		return sampleEllipticalArcEntity(t)
	case *EquationCurve:
		return t.Sample(curveSamples)
	case *FixedSpline:
		return t.Pts
	case *OffsetSpline:
		return t.Sample()
	default:
		a, b, ok := segmentEnds(e)
		if !ok {
			return nil
		}
		return []math.Point2{a.Position(), b.Position()}
	}
}

// sampleEllipticalArcEntity samples an elliptical arc from StartAngle to EndAngle along
// the ellipse (in the major/minor frame), endpoints inclusive.
func sampleEllipticalArcEntity(e *EllipticalArc) []math.Point2 {
	c := e.Center.Position()
	ux, uy := unitAxis(e.MajorAxis)
	start := float64(e.StartAngle)
	sweep := float64(e.EndAngle) - start
	pts := make([]math.Point2, curveSamples+1)
	for i := range pts {
		a := start + sweep*float64(i)/float64(curveSamples)
		mx, my := e.MajorRadius*stdmath.Cos(a), e.MinorRadius*stdmath.Sin(a)
		pts[i] = math.P2(c.X+mx*ux-my*uy, c.Y+mx*uy+my*ux)
	}
	return pts
}

// sampleArcEntity samples a circular arc from its Start to its End along the true
// arc (respecting CounterClockwise), endpoints inclusive.
func sampleArcEntity(a *Arc) []math.Point2 {
	c := a.Center.Position()
	r := a.Radius()
	start := angleOfPoint(c, a.Start.Position())
	sweep := arcSweepSigned(a, c, start)
	pts := make([]math.Point2, curveSamples+1)
	for i := range pts {
		ang := start + sweep*float64(i)/float64(curveSamples)
		pts[i] = math.P2(c.X+r*stdmath.Cos(ang), c.Y+r*stdmath.Sin(ang))
	}
	return pts
}

// arcSweepSigned returns the signed sweep from start to the arc's end in the arc's
// winding direction (positive counter-clockwise, negative clockwise). A start==end
// arc is a full turn (±2π) rather than a zero sweep. (Distinct from dimension.go's
// arcSweep, which is the unsigned magnitude used for angular dimensions.)
func arcSweepSigned(a *Arc, center math.Point2, start float64) float64 {
	sweep := angleOfPoint(center, a.End.Position()) - start
	if a.CounterClockwise {
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

// sampleEllipseEntity samples a full ellipse's perimeter (counter-clockwise in its
// major/minor frame) for a standalone closed loop.
func sampleEllipseEntity(e *Ellipse) []math.Point2 {
	c := e.Center.Position()
	ux, uy := unitAxis(e.MajorAxis)
	pts := make([]math.Point2, curveSamples)
	for i := range pts {
		a := 2 * stdmath.Pi * float64(i) / float64(curveSamples)
		mx, my := e.MajorRadius*stdmath.Cos(a), e.MinorRadius*stdmath.Sin(a)
		pts[i] = math.P2(c.X+mx*ux-my*uy, c.Y+mx*uy+my*ux)
	}
	return pts
}

// angleOfPoint returns the angle of the vector from center to p, in (−π, π].
func angleOfPoint(center, p math.Point2) float64 {
	v := center.VectorTo(p)
	return stdmath.Atan2(v.Y, v.X)
}

// unitAxis returns the normalized components of v, falling back to +X for a zero
// vector (so a degenerate major axis still yields a usable frame).
func unitAxis(v math.Vector2) (float64, float64) {
	l := v.Length()
	if l < math.DefaultTolerance {
		return 1, 0
	}
	return v.X / l, v.Y / l
}

// reverseInPlace2 reverses a slice of points in place.
func reverseInPlace2(pts []math.Point2) {
	for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
		pts[i], pts[j] = pts[j], pts[i]
	}
}
