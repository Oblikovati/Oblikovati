// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	gmath "oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// zProfile is an open Z-chain with TWO interior corners, each a right-angle turn:
// (0,0)→(1,0)→(1,1)→(2,1).
func zProfile() *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	p0 := s.Points().Add(gmath.P2(0, 0))
	p1 := s.Points().Add(gmath.P2(1, 0))
	p2 := s.Points().Add(gmath.P2(1, 1))
	p3 := s.Points().Add(gmath.P2(2, 1))
	s.Lines().Add(p0, p1)
	s.Lines().Add(p1, p2)
	s.Lines().Add(p2, p3)
	return s
}

// straightProfile is a single straight run drawn as two collinear segments — no corner to bend:
// (0,0)→(1,0)→(2,0).
func straightProfile() *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	p0 := s.Points().Add(gmath.P2(0, 0))
	p1 := s.Points().Add(gmath.P2(1, 0))
	p2 := s.Points().Add(gmath.P2(2, 0))
	s.Lines().Add(p0, p1)
	s.Lines().Add(p1, p2)
	return s
}

// TestContourFlangeBendSpecsDevelopEveryCorner is the #2076 regression: a contour flange's rounded
// interior corners ARE bends, so BendSpecs must report one per corner — otherwise the flat pattern
// develops them as sharp folds and the blank comes out short by every corner's bend allowance, and
// any DXF cut from it is undersized. Before this the feature implemented no BendLineage and
// contributed zero bends. The swept angle is the corner's own turn; the reported radius follows the
// flange convention (the override, else 0 to defer to the rule's default).
func TestContourFlangeBendSpecsDevelopEveryCorner(t *testing.T) {
	t.Parallel()
	right := stdmath.Pi / 2
	for name, tc := range map[string]struct {
		profile   *sketch.Sketch
		override  float64 // < 0 ⇒ no override closure (defer to the rule)
		wantAngle []float64
		wantR     float64
	}{
		"L corner defers to rule": {lProfile(), -1, []float64{right}, 0},
		"L corner with override":  {lProfile(), 0.4, []float64{right}, 0.4},
		"Z has two bends":         {zProfile(), 0.3, []float64{right, right}, 0.3},
		"straight has no bend":    {straightProfile(), 0.3, nil, 0},
	} {
		t.Run(name, func(t *testing.T) {
			def := &SheetMetalContourFlangeDefinition{Profile: tc.profile}
			if tc.override >= 0 {
				def.Radius = constClosure(tc.override)
			}
			f := &SheetMetalContourFlangeFeature{def: def}
			specs := f.BendSpecs(0.2)
			if len(specs) != len(tc.wantAngle) {
				t.Fatalf("BendSpecs returned %d bends, want %d", len(specs), len(tc.wantAngle))
			}
			for i, s := range specs {
				if stdmath.Abs(s.Angle-tc.wantAngle[i]) > 1e-9 {
					t.Errorf("bend %d swept angle = %g, want %g", i, s.Angle, tc.wantAngle[i])
				}
				if s.Radius != tc.wantR {
					t.Errorf("bend %d radius = %g, want %g (override/defer convention)", i, s.Radius, tc.wantR)
				}
			}
		})
	}
}
