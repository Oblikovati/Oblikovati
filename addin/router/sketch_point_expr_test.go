// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"testing"

	"oblikovati.org/api/wire"
)

// TestResolvePointsEvaluatesExpressions: PointExprs coordinates are evaluated through the part's
// parameter engine (so generated line/arc/point geometry is parametric at construction), while the
// literal Points path is unchanged when no expressions are given (Oblikovati.API#189).
func TestResolvePointsEvaluatesExpressions(t *testing.T) {
	part := partWith(t, map[string]string{"bore_r": "10 mm", "slot_depth": "5 mm"})

	pts, err := resolvePoints(part, wire.AddSketchEntityArgs{
		PointExprs: [][]string{{"0", "0"}, {"bore_r", "0"}, {"bore_r * 2", "slot_depth"}},
	})
	if err != nil {
		t.Fatalf("resolvePoints(exprs): %v", err)
	}
	if len(pts) != 3 {
		t.Fatalf("got %d points, want 3", len(pts))
	}
	// bore_r = 10 mm = 1.0 cm (database unit); bore_r*2 = 2.0 cm; slot_depth = 0.5 cm.
	if math.Abs(float64(pts[1].X)-1.0) > 1e-9 || math.Abs(float64(pts[2].X)-2.0) > 1e-9 || math.Abs(float64(pts[2].Y)-0.5) > 1e-9 {
		t.Errorf("resolved points = %v, want second X=1.0, third [2.0,0.5] cm", pts)
	}

	// No PointExprs ⇒ the literal path (coordinates already in cm).
	lit, err := resolvePoints(part, wire.AddSketchEntityArgs{Points: [][]float64{{1.5, 2.5}}})
	if err != nil {
		t.Fatalf("resolvePoints(literal): %v", err)
	}
	if math.Abs(float64(lit[0].X)-1.5) > 1e-9 || math.Abs(float64(lit[0].Y)-2.5) > 1e-9 {
		t.Errorf("literal point = %v, want [1.5,2.5]", lit[0])
	}
}

// TestResolvePointsRejectsMalformedExpr: a point expression without exactly two coordinates errors
// with the offending index, per the house exception-message rule.
func TestResolvePointsRejectsMalformedExpr(t *testing.T) {
	part := partWith(t, nil)
	if _, err := resolvePoints(part, wire.AddSketchEntityArgs{PointExprs: [][]string{{"1"}}}); err == nil {
		t.Error("a one-coordinate pointExprs entry should error")
	}
}
