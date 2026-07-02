// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

func TestConstraints3DResiduals(t *testing.T) {
	a := NewPoint3D(math.P3(1, 2, 3))
	b := NewPoint3D(math.P3(1, 2, 9))

	coin := NewCoincident3D(a, b)
	if satisfied(coin) {
		t.Error("coincident3D satisfied for different Z")
	}
	b.SetPosition(math.P3(1, 2, 3))
	if !satisfied(coin) || len(coin.Variables()) != 6 {
		t.Error("coincident3D wrong after aligning points")
	}

	// Collinear along the X axis.
	p0 := NewPoint3D(math.P3(0, 0, 0))
	p1 := NewPoint3D(math.P3(1, 0, 0))
	p2 := NewPoint3D(math.P3(5, 0, 0))
	if p0.EntityID() == 0 {
		t.Error("Point3D id should be nonzero")
	}
	col := NewCollinear3D(p0, p1, p2)
	if !satisfied(col) || len(col.Variables()) != 9 {
		t.Errorf("collinear3D wrong: vars=%d", len(col.Variables()))
	}
	conc := NewConcentric3D(NewPoint3D(math.P3(1, 1, 1)), NewPoint3D(math.P3(1, 1, 1)))
	if !satisfied(conc) || len(conc.Variables()) != 6 {
		t.Errorf("concentric3D wrong: vars=%d", len(conc.Variables()))
	}
	p2.SetPosition(math.P3(5, 1, 0))
	if satisfied(NewCollinear3D(p0, p1, p2)) {
		t.Error("collinear3D satisfied for a bent chain")
	}

	// Concentric centers.
	if !satisfied(NewConcentric3D(NewPoint3D(math.P3(2, 2, 2)), NewPoint3D(math.P3(2, 2, 2)))) {
		t.Error("concentric3D not satisfied for equal centers")
	}

	// Equal radius DOFs, resolved from the operand curves (#1625).
	s3 := NewSketches3D().Add()
	zUp, _ := math.NewUnitVector3(0, 0, 1)
	circA := s3.AddCircle3D(math.P3(0, 0, 0), zUp, 4)
	circB := s3.AddCircle3D(math.P3(9, 0, 0), zUp, 4)
	eq := mustEqual3D(t, circA, circB)
	if !satisfied(eq) {
		t.Error("equal3D not satisfied for equal radii")
	}
	circB.Radius = 5
	if satisfied(eq) {
		t.Error("equal3D satisfied after changing one radius")
	}
	if _, err := NewEqual3D(s3.AddPoint3D(math.P3(0, 0, 0)), circA); err == nil {
		t.Error("NewEqual3D should refuse a non-radius-bearing operand")
	}
}

func TestCustomConstraint3D(t *testing.T) {
	x := math.Scalar(3)
	// Custom: x must equal 10.
	c := NewCustomConstraint3D(func() []float64 { return []float64{x - 10} }, []*math.Scalar{&x})
	if satisfied(c) {
		t.Error("custom constraint satisfied before x reaches 10")
	}
	x = 10
	if !satisfied(c) || c.EntityID() == 0 || len(c.Variables()) != 1 {
		t.Error("custom constraint wrong after x = 10")
	}
}
