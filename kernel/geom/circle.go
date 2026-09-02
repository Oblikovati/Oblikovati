// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// Circle is a full 3D circle (contract: Circle) lying in the plane through
// Center with the given unit Normal. RefDir is a unit vector in that plane
// marking parameter angle 0; the circle is parameterized t∈[0,1] mapping to
// angle 2πt, counter-clockwise about Normal.
type Circle struct {
	Center math.Point3
	Normal math.UnitVector3
	RefDir math.UnitVector3
	Radius float64
}

// NewCircle builds a circle from a center, plane normal, and radius, choosing an
// arbitrary in-plane reference direction. It errors when normal is zero length.
func NewCircle(center math.Point3, normal math.Vector3, radius float64) (Circle, error) {
	n, err := math.UnitVector3FromVector(normal)
	if err != nil {
		return Circle{}, err
	}
	return Circle{Center: center, Normal: n, RefDir: perpendicularUnit(n), Radius: radius}, nil
}

// binormal returns the in-plane unit vector perpendicular to RefDir (Normal ×
// RefDir), completing the right-handed evaluation basis.
func (c Circle) binormal() math.Vector3 {
	return c.Normal.Cross(c.RefDir)
}

// PointAt returns the point at angle 2πt.
func (c Circle) PointAt(t float64) math.Point3 {
	return pointOnCircle(c.Center, c.RefDir.AsVector(), c.binormal(), c.Radius, twoPi*t)
}

// TangentAt returns the derivative dP/dt at parameter t.
func (c Circle) TangentAt(t float64) math.Vector3 {
	return circleTangent(c.RefDir.AsVector(), c.binormal(), c.Radius, twoPi*t).Scale(twoPi)
}

// Domain returns [0, 1].
func (c Circle) Domain() (lo, hi float64) { return 0, 1 }

// Circumference returns 2πr.
func (c Circle) Circumference() float64 { return twoPi * c.Radius }

// CircleByThreePoints builds the unique 3D circle through three points (contract:
// CreateCircleByThreePoints): it lies in the plane the points span and is centered at
// their circumcenter, with RefDir pointing from the center to the first point. It errors
// when the points are collinear (no finite circle passes through them).
func CircleByThreePoints(a, b, c math.Point3) (Circle, error) {
	n, e1, e2, err := planarFrame(a, b, c)
	if err != nil {
		return Circle{}, err
	}
	// Project into the points' plane (a at the origin), fit the 2D circumcenter there,
	// then lift the center back to 3D — the same construction as Arc3dByThreePoints.
	center2, _ := circumcenter2d(math.P2(0, 0), projectInto(a, e1, e2, b), projectInto(a, e1, e2, c))
	center := a.TranslateBy(e1.Scale(center2.X)).TranslateBy(e2.Scale(center2.Y))
	ref, _ := math.UnitVector3FromVector(center.VectorTo(a))
	return Circle{Center: center, Normal: n, RefDir: ref, Radius: center.DistanceTo(a)}, nil
}

// AsCircle returns the full circle a curve runs on, when it is one.
//
// It exists so a consumer does not have to type-assert a geometry kind to ask a geometric
// question. The kernel ground rules put geometry-kind switches in this package only —
// "behaviour many operations need is a method on geom.Surface/geom.Curve" — and several
// operations need exactly this: the chamfer and fillet ask it to decide whether an edge is a
// rim they can sweep rotationally, and the drill recognizers ask it of an imprint.
//
// A circular ARC is not a circle: it carries its own curve type and its own sweep, and an
// operation that treats a 90° arc as a full rim would revolve a tool all the way round.
//
// Example:
//
//	if c, ok := geom.AsCircle(edge.Geometry()); ok { axis := c.Normal }
func AsCircle(c Curve3) (Circle, bool) {
	circle, ok := c.(Circle)
	return circle, ok
}
