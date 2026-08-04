// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// A centre point is a hole-centre marker. Nothing consumes it yet — the assembly hole takes an
// explicit 3D centre and there is no part Hole feature — but it renders distinctly and must
// survive a round trip so a future consumer finds it.
func TestCenterPointFlag(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	p := s.Points().Add(math.P2(1, 2))
	if p.IsCenterPoint() {
		t.Fatal("a plain point must not start as a centre point")
	}
	p.SetCenterPoint(true)
	if !p.IsCenterPoint() {
		t.Error("SetCenterPoint(true) must take effect")
	}
	p.SetCenterPoint(false)
	if p.IsCenterPoint() {
		t.Error("SetCenterPoint(false) must clear it")
	}
}

func TestRecipePointCanBeACenterPoint(t *testing.T) {
	r := Recipe{
		Points:   []math.Point2{math.P2(3, 4)},
		Entities: []RecipeEntity{{Kind: RecipePoint, Points: []int{0}, CenterPoint: true}},
	}
	s := NewSketches().Add(XYPlane())
	ents, _, err := s.Apply(r, types.OverConstrainedApplyDriven)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	p, ok := ents[0].(*Point)
	if !ok {
		t.Fatalf("entity = %T, want *Point", ents[0])
	}
	if !p.IsCenterPoint() {
		t.Error("a RecipePoint with CenterPoint set must produce a centre point")
	}
}

// Curve endpoints are never centre points — only a point placed in its own right can be one.
func TestCurveEndpointsAreNotCenterPoints(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	if l.A.IsCenterPoint() || l.B.IsCenterPoint() {
		t.Error("a line's endpoints must not be centre points")
	}
}
