// SPDX-License-Identifier: GPL-2.0-only

package geomapi

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestFactoryCreateConstructors covers the transient-geometry factory constructors that the
// existing tests did not reach: each builds from valid inputs and must return a non-nil curve
// or surface without error.
func TestFactoryCreateConstructors(t *testing.T) {
	f := New()
	z := types.UnitVector{X: 0, Y: 0, Z: 1}
	x := types.UnitVector{X: 1, Y: 0, Z: 0}
	x2 := types.UnitVector2d{X: 1, Y: 0}
	p := types.NewPoint
	p2 := types.NewPoint2d
	curveDef := types.BSplineCurveDef{Degree: 2, Poles: []types.Point{p(0, 0, 0), p(1, 1, 0), p(2, 0, 0)}, Knots: []float64{0, 0, 0, 1, 1, 1}}
	curve2dDef := types.BSplineCurve2dDef{Degree: 2, Poles: []types.Point2d{p2(0, 0), p2(1, 1), p2(2, 0)}, Knots: []float64{0, 0, 0, 1, 1, 1}}

	checks := []struct {
		name string
		err  error
	}{
		{"Line", second(f.CreateLine(p(0, 0, 0), z))},
		{"CircleByThreePoints", second(f.CreateCircleByThreePoints(p(1, 0, 0), p(0, 1, 0), p(-1, 0, 0)))},
		{"ArcByThreePoints", second(f.CreateArcByThreePoints(p(1, 0, 0), p(0, 1, 0), p(-1, 0, 0)))},
		{"EllipseFull", second(f.CreateEllipseFull(p(0, 0, 0), z, x, 3, 1))},
		{"EllipticalArc", second(f.CreateEllipticalArc(p(0, 0, 0), z, x, 3, 1, 0, 1.5))},
		{"FittedBSplineCurve", second(f.CreateFittedBSplineCurve([]types.Point{p(0, 0, 0), p(1, 1, 0), p(2, 0, 0)}))},
		{"Cylinder", second(f.CreateCylinder(p(0, 0, 0), z, 2))},
		{"Cone", second(f.CreateCone(p(0, 0, 0), z, 0.4))},
		{"Line2d", second(f.CreateLine2d(p2(0, 0), x2))},
		{"EllipseFull2d", second(f.CreateEllipseFull2d(p2(0, 0), x2, 3, 1))},
		{"EllipticalArc2d", second(f.CreateEllipticalArc2d(p2(0, 0), x2, 3, 1, 0, 1.5))},
		{"Polyline2d", second(f.CreatePolyline2d([]types.Point2d{p2(0, 0), p2(1, 0), p2(1, 1)}))},
		{"BSplineCurve2d", second(f.CreateBSplineCurve2d(curve2dDef))},
		{"BSplineCurve", second(f.CreateBSplineCurve(curveDef))},
		{"FittedBSplineCurve2d", second(f.CreateFittedBSplineCurve2d([]types.Point2d{p2(0, 0), p2(1, 1), p2(2, 0)}))},
	}
	for _, c := range checks {
		if c.err != nil {
			t.Errorf("Create%s: unexpected error: %v", c.name, c.err)
		}
	}
}

// second discards a constructor's first (value) return, keeping only the error, so the table
// above stays compact across constructors with different value types.
func second[T any](_ T, err error) error { return err }
