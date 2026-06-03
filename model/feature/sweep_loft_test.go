// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// centeredSquareOn returns a sketch on plane with a centered square of the given half
// width (corners ±half), wound counter-clockwise.
func centeredSquareOn(plane sketch.Plane, half float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(plane)
	c0 := s.Points().Add(math.P2(-half, -half))
	c1 := s.Points().Add(math.P2(half, -half))
	c2 := s.Points().Add(math.P2(half, half))
	c3 := s.Points().Add(math.P2(-half, half))
	s.Lines().Add(c0, c1)
	s.Lines().Add(c1, c2)
	s.Lines().Add(c2, c3)
	s.Lines().Add(c3, c0)
	return s
}

// planeAtZ returns the XY-parallel sketch plane at height z.
func planeAtZ(z float64) sketch.Plane {
	p, _ := sketch.NewPlane(math.P3(0, 0, z), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
	return p
}

// sketchList is a SketchIndexer over an ordered set of sketches (loft uses several).
type sketchList struct{ sks []*sketch.Sketch }

func (l sketchList) IndexOf(s *sketch.Sketch) (int, bool) {
	for i, x := range l.sks {
		if x == s {
			return i, true
		}
	}
	return 0, false
}

func (l sketchList) At(i int) (*sketch.Sketch, bool) {
	if i < 0 || i >= len(l.sks) {
		return nil, false
	}
	return l.sks[i], true
}

func TestSweepAlongPathMakesValidSolid(t *testing.T) {
	// A 2×2 square swept along an L-path (up Z, then over X) → a valid elbow solid.
	fs := NewPartFeatures(nil, nil)
	path := sketch.NewPath3D([]*sketch.Point3D{
		sketch.NewPoint3D(math.P3(0, 0, 0)),
		sketch.NewPoint3D(math.P3(0, 0, 5)),
		sketch.NewPoint3D(math.P3(5, 0, 5)),
	}, false)
	pf := NewSweepFeatures(fs).Add(centeredSquareOn(sketch.XYPlane(), 1), 0, path, nil, ops.NewBody)
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("sweep went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("swept body is not a valid solid: %+v", r)
	}
	// Cross-section area 4 along a path of length 10 → volume on the order of 40.
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; v < 20 || v > 60 {
		t.Errorf("swept volume = %g, want roughly 40 (area 4 × path 10)", v)
	}
}

func TestLoftBetweenSquaresIsFrustum(t *testing.T) {
	// A 4×4 square at z=0 lofted to a 2×2 square at z=5 → a square frustum:
	// V = h/3·(A1 + A2 + √(A1·A2)) = 5/3·(16 + 4 + 8) = 140/3 ≈ 46.667.
	fs := NewPartFeatures(nil, nil)
	bottom := centeredSquareOn(sketch.XYPlane(), 2)
	top := centeredSquareOn(planeAtZ(5), 1)
	pf := NewLoftFeatures(fs).Add([]LoftSection{{bottom, 0}, {top, 0}}, false, ops.NewBody)
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("loft went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("lofted body is not a valid solid: %+v", r)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(v, 140.0/3) > 0.02 {
		t.Errorf("frustum volume = %g, want ≈46.667", v)
	}
}

func TestSweepAndLoftRoundTrip(t *testing.T) {
	prof := centeredSquareOn(sketch.XYPlane(), 1)
	bottom := centeredSquareOn(sketch.XYPlane(), 2)
	top := centeredSquareOn(planeAtZ(5), 1)
	idx := sketchList{sks: []*sketch.Sketch{prof, bottom, top}}

	fs := NewPartFeatures(nil, nil)
	path := sketch.NewPath3D([]*sketch.Point3D{
		sketch.NewPoint3D(math.P3(0, 0, 0)), sketch.NewPoint3D(math.P3(0, 0, 5)),
	}, false)
	NewSweepFeatures(fs).Add(prof, 0, path, func() float64 { return 0.3 }, ops.Join)
	NewLoftFeatures(fs).Add([]LoftSection{{bottom, 0}, {top, 0}}, false, ops.NewBody)

	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, idx, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	sweep := fresh.Item(0).Definition().(*SweepFeature).Definition()
	if sweep.Path.Count() != 2 || sweep.Twist() != 0.3 || sweep.Operation != ops.Join {
		t.Errorf("sweep round-trip lost data: pts=%d twist=%g op=%v", sweep.Path.Count(), sweep.Twist(), sweep.Operation)
	}
	loft := fresh.Item(1).Definition().(*LoftFeature).Definition()
	if len(loft.Sections) != 2 || loft.Sections[1].Sketch != top {
		t.Errorf("loft round-trip lost sections: %+v", loft.Sections)
	}
}
