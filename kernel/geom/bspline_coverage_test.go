// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

func TestBSplineCurve2dValidationAndCopies(t *testing.T) {
	ctrl := []math.Point2{math.P2(0, 0), math.P2(1, 0)}
	weights := []float64{1, 2}
	knots := []float64{0, 0, 1, 1}
	curve, err := NewBSplineCurve2d(1, ctrl, weights, knots)
	if err != nil {
		t.Fatalf("NewBSplineCurve2d: %v", err)
	}
	ctrl[0] = math.P2(99, 99)
	weights[0] = 99
	knots[0] = 99
	if curve.Ctrl[0] != math.P2(0, 0) || curve.Weights[0] != 1 || curve.Knots[0] != 0 {
		t.Fatalf("B-spline constructor aliased input slices: %#v", curve)
	}
	if _, err := NewBSplineCurve2d(1, []math.Point2{math.P2(0, 0)}, []float64{1}, []float64{0, 0, 1}); err == nil {
		t.Fatal("NewBSplineCurve2d accepted too few control points")
	}
	if _, err := NewBSplineCurve2d(1, []math.Point2{math.P2(0, 0), math.P2(1, 0)}, []float64{1, 0}, knots); err == nil {
		t.Fatal("NewBSplineCurve2d accepted a non-positive weight")
	}
}

func TestFittedBSplineValidationAndAveragedKnots(t *testing.T) {
	if _, err := NewFittedBSplineCurve(nil); err == nil {
		t.Fatal("NewFittedBSplineCurve accepted no points")
	}
	if _, err := NewFittedBSplineCurve2d([]math.Point2{math.P2(0, 0)}); err == nil {
		t.Fatal("NewFittedBSplineCurve2d accepted one point")
	}
	knots := averagedKnots([]float64{0, 0.25, 0.5, 0.75, 1}, 3)
	if len(knots) != 9 || knots[0] != 0 || knots[3] != 0 || knots[4] != 0.5 || knots[8] != 1 {
		t.Fatalf("averagedKnots = %v", knots)
	}
}

func TestBSplineSurfaceNetValidationBranches(t *testing.T) {
	ctrl := [][]math.Point3{{math.P3(0, 0, 0), math.P3(1, 0, 0)}, {math.P3(0, 1, 0), math.P3(1, 1, 0)}}
	weights := [][]float64{{1, 1}, {1, 1}}
	s, err := NewBSplineSurface(1, 1, ctrl, weights, []float64{0, 0, 1, 1}, []float64{0, 0, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	ctrl[0][0] = math.P3(99, 99, 99)
	weights[0][0] = 99
	if s.Ctrl[0][0] != math.P3(0, 0, 0) || s.Weights[0][0] != 1 {
		t.Fatalf("B-spline surface constructor aliased input nets: %#v", s)
	}
	if _, err := NewBSplineSurface(1, 1, nil, nil, nil, nil); err == nil {
		t.Fatal("NewBSplineSurface accepted an empty net")
	}
	if _, err := NewBSplineSurface(1, 1, [][]math.Point3{{math.P3(0, 0, 0)}, {math.P3(1, 0, 0), math.P3(2, 0, 0)}}, [][]float64{{1}, {1}}, nil, nil); err == nil {
		t.Fatal("NewBSplineSurface accepted a non-rectangular control net")
	}
	if _, err := NewBSplineSurface(1, 1, ctrl, [][]float64{{1}}, []float64{0, 0, 1, 1}, []float64{0, 0, 1, 1}); err == nil {
		t.Fatal("NewBSplineSurface accepted a mismatched weight net")
	}
	if _, err := NewBSplineSurface(1, 1, ctrl, [][]float64{{1, 1}, {1, 0}}, []float64{0, 0, 1, 1}, []float64{0, 0, 1, 1}); err == nil {
		t.Fatal("NewBSplineSurface accepted a non-positive surface weight")
	}
}
