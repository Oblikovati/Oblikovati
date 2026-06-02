// SPDX-License-Identifier: GPL-2.0-only

package geom

import "testing"

// Compile-time proof that every curve value type satisfies the evaluation
// interface for its dimension. A type that drops PointAt/TangentAt/Domain fails
// the build here, not at some distant call site.
var (
	_ Curve3 = Line{}
	_ Curve3 = LineSegment{}
	_ Curve3 = Circle{}
	_ Curve3 = Arc3d{}
	_ Curve3 = EllipseFull{}
	_ Curve3 = EllipticalArc{}
	_ Curve3 = Polyline{}
	_ Curve3 = BSplineCurve{}

	_ Surface = Plane{}
	_ Surface = Cylinder{}
	_ Surface = Cone{}
	_ Surface = Sphere{}
	_ Surface = Torus{}
	_ Surface = BSplineSurface{}

	_ Curve2 = Line2d{}
	_ Curve2 = LineSegment2d{}
	_ Curve2 = Circle2d{}
	_ Curve2 = Arc2d{}
	_ Curve2 = EllipseFull2d{}
	_ Curve2 = EllipticalArc2d{}
	_ Curve2 = Polyline2d{}
)

// eqScalar is the per-test numeric tolerance for evaluated geometry.
const eqScalar = 1e-12

func approxScalar(t *testing.T, got, want float64, what string) {
	t.Helper()
	if d := got - want; d > eqScalar || d < -eqScalar {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}
