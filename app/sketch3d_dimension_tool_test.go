// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestSketch3DDimensionToolInfersKinds drives the command + pick flow for each
// inferred dimension kind: a line sizes its length, a circle its radius, a spline its
// arc length, two points their distance.
func TestSketch3DDimensionToolInfersKinds(t *testing.T) {
	zAxis, err := math.NewUnitVector3(0, 0, 1)
	if err != nil {
		t.Fatalf("axis: %v", err)
	}
	cases := []struct {
		name, wantKind string
		picks          func(sk *sketch.Sketch3D) []sketch.Entity
	}{
		{"line", "lineLength", func(sk *sketch.Sketch3D) []sketch.Entity {
			return []sketch.Entity{sk.AddLine3D(math.P3(0, 0, 0), math.P3(3, 0, 4))}
		}},
		{"circle", "radius", func(sk *sketch.Sketch3D) []sketch.Entity {
			return []sketch.Entity{sk.AddCircle3D(math.P3(0, 0, 0), zAxis, 2)}
		}},
		{"spline", "splineLength", func(sk *sketch.Sketch3D) []sketch.Entity {
			return []sketch.Entity{sk.AddSpline3D([]math.Point3{{X: 0}, {X: 1, Y: 1}, {X: 2}}, false, true)}
		}},
		{"two points", "distance", func(sk *sketch.Sketch3D) []sketch.Entity {
			return []sketch.Entity{sk.AddPoint3D(math.P3(0, 0, 0)), sk.AddPoint3D(math.P3(1, 2, 2))}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, sk := sketch3DSession(t)
			ents := tc.picks(sk)
			if err := s.Execute("Sketch3D.Dimension"); err != nil {
				t.Fatalf("Sketch3D.Dimension: %v", err)
			}
			for _, e := range ents {
				s.feedPick(SketchEntityHandle{Entity: e})
			}
			dims := sk.DimensionConstraints3D()
			if dims.Count() != 1 {
				t.Fatalf("dimensions = %d, want 1", dims.Count())
			}
			if got := dims.Item(0).KindName(); got != tc.wantKind {
				t.Errorf("kind = %q, want %q", got, tc.wantKind)
			}
			if s.ActiveTool() != nil {
				t.Error("tool should deactivate after the dimension applies")
			}
		})
	}
}

// TestSketch3DDimensionSeedsMeasuredValue checks the created dimension's expression
// evaluates to the picked geometry's current size (a 3-4-5 line → 5 cm).
func TestSketch3DDimensionSeedsMeasuredValue(t *testing.T) {
	s, sk := sketch3DSession(t)
	l := sk.AddLine3D(math.P3(0, 0, 0), math.P3(3, 0, 4))
	if err := s.Execute("Sketch3D.Dimension"); err != nil {
		t.Fatalf("Sketch3D.Dimension: %v", err)
	}
	s.feedPick(SketchEntityHandle{Entity: l})
	d := sk.DimensionConstraints3D().Item(0)
	if got := d.Parameter().ModelValue(); stdmath.Abs(got-5) > 1e-9 {
		t.Errorf("seeded value = %v cm, want 5 (the line's current length)", got)
	}
}
