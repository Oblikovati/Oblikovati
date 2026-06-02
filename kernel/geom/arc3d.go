// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/math"
)

// Arc3d is a 3D circular arc (contract: Arc3d) on the circle defined by Center,
// the plane unit Normal, and Radius. RefDir (unit, in-plane) marks angle 0; the
// arc runs from StartAngle sweeping SweepAngle radians (signed about Normal),
// parameterized t∈[0,1].
type Arc3d struct {
	Center     math.Point3
	Normal     math.UnitVector3
	RefDir     math.UnitVector3
	Radius     float64
	StartAngle float64
	SweepAngle float64
}

// NewArc3d builds an arc; refDir is projected onto the plane and normalized so
// the stored RefDir is exactly perpendicular to normal. Errors on a zero normal.
func NewArc3d(center math.Point3, normal, refDir math.Vector3, radius, startAngle, sweepAngle float64) (Arc3d, error) {
	n, err := math.UnitVector3FromVector(normal)
	if err != nil {
		return Arc3d{}, err
	}
	return Arc3d{
		Center: center, Normal: n, RefDir: planarRef(n, refDir),
		Radius: radius, StartAngle: startAngle, SweepAngle: sweepAngle,
	}, nil
}

// Arc3dByThreePoints builds the arc through start, onArc, and end. It errors
// when the three points are collinear (no plane/circle is determined).
func Arc3dByThreePoints(start, onArc, end math.Point3) (Arc3d, error) {
	n, e1, e2, err := planarFrame(start, onArc, end)
	if err != nil {
		return Arc3d{}, err
	}
	start2 := math.P2(0, 0) // start projects to the frame origin by construction
	on2 := projectInto(start, e1, e2, onArc)
	end2 := projectInto(start, e1, e2, end)
	c2, _ := circumcenter2d(start2, on2, end2) // non-collinear: frame guaranteed it
	center := start.TranslateBy(e1.Scale(c2.X)).TranslateBy(e2.Scale(c2.Y))
	ref, _ := math.UnitVector3FromVector(center.VectorTo(start))
	return Arc3d{
		Center: center, Normal: n, RefDir: ref, Radius: center.DistanceTo(start),
		StartAngle: 0, SweepAngle: arcSweep(start2, on2, end2, c2),
	}, nil
}

// arcSweep returns the signed sweep from start2 to end2 about center2, sign
// chosen so the arc winds through on2. The (e1, e2) projection plane and the
// stored (RefDir, binormal) frame are both right-handed about Normal, and the
// constant RefDir offset cancels in this angle difference — so StartAngle stays
// 0 while this gives the correct signed sweep.
func arcSweep(start2, on2, end2, center2 math.Point2) float64 {
	base := angleOf2d(center2, start2)
	sweep := wrapPositive(angleOf2d(center2, end2) - base)
	if signedArea2(start2, on2, end2) < 0 { // clockwise winding ⇒ negative sweep
		sweep -= twoPi
	}
	return sweep
}

// planarRef projects r onto the plane with unit normal n and normalizes it,
// falling back to an arbitrary in-plane direction when r is parallel to n.
func planarRef(n math.UnitVector3, r math.Vector3) math.UnitVector3 {
	proj := r.Sub(n.AsVector().Scale(r.Dot(n.AsVector())))
	u, err := math.UnitVector3FromVector(proj)
	if err != nil {
		return perpendicularUnit(n)
	}
	return u
}

// planarFrame returns the unit normal and an in-plane right-handed basis
// (e1, e2) for the plane through three points, erroring when they are collinear.
func planarFrame(a, b, c math.Point3) (n math.UnitVector3, e1, e2 math.Vector3, err error) {
	v1, v2 := a.VectorTo(b), a.VectorTo(c)
	normal, nerr := math.UnitVector3FromVector(v1.Cross(v2))
	if nerr != nil {
		return math.UnitVector3{}, math.Vector3{}, math.Vector3{}, &CollinearPoints3dError{A: a, B: b, C: c}
	}
	e1u, _ := math.UnitVector3FromVector(v1)
	return normal, e1u.AsVector(), normal.Cross(e1u), nil
}

// projectInto returns the 2D coordinates of q in the frame at origin with basis
// (e1, e2).
func projectInto(origin math.Point3, e1, e2 math.Vector3, q math.Point3) math.Point2 {
	d := origin.VectorTo(q)
	return math.P2(d.Dot(e1), d.Dot(e2))
}

// binormal returns Normal × RefDir, the in-plane unit vector at angle +π/2.
func (a Arc3d) binormal() math.Vector3 {
	return a.Normal.Cross(a.RefDir)
}

// PointAt returns the point at parameter t.
func (a Arc3d) PointAt(t float64) math.Point3 {
	angle := a.StartAngle + t*a.SweepAngle
	return pointOnCircle(a.Center, a.RefDir.AsVector(), a.binormal(), a.Radius, angle)
}

// TangentAt returns the derivative dP/dt (includes the sweep chain factor).
func (a Arc3d) TangentAt(t float64) math.Vector3 {
	angle := a.StartAngle + t*a.SweepAngle
	return circleTangent(a.RefDir.AsVector(), a.binormal(), a.Radius, angle).Scale(a.SweepAngle)
}

// Domain returns [0, 1].
func (a Arc3d) Domain() (lo, hi float64) { return 0, 1 }

// Length returns |Radius · SweepAngle|.
func (a Arc3d) Length() float64 {
	return stdmath.Abs(a.Radius * a.SweepAngle)
}
