// SPDX-License-Identifier: GPL-2.0-only

package param

import "testing"

func TestQuantitySameUnitAddSub(t *testing.T) {
	a, b := Q(3, Length), Q(2, Length)
	if sum, err := a.Add(b); err != nil || sum != (Quantity{5, Length}) {
		t.Errorf("Add = %v, %v; want {5 length}", sum, err)
	}
	if diff, err := a.Sub(b); err != nil || diff != (Quantity{1, Length}) {
		t.Errorf("Sub = %v, %v; want {1 length}", diff, err)
	}
}

func TestQuantityMixedUnitAddRejected(t *testing.T) {
	_, err := Q(1, Length).Add(Q(1, Angle))
	if err == nil {
		t.Fatal("adding length to angle should error")
	}
	if _, ok := err.(*DimensionError); !ok {
		t.Errorf("error type = %T, want *DimensionError", err)
	}
}

func TestQuantityMulProducesDerivedUnit(t *testing.T) {
	// Length · Length → Area.
	if got, err := Q(2, Length).Mul(Q(3, Length)); err != nil || got != (Quantity{6, Area}) {
		t.Errorf("Length·Length = %v, %v; want {6 area}", got, err)
	}
	// Area · Length → Volume.
	if got, err := Q(6, Area).Mul(Q(2, Length)); err != nil || got != (Quantity{12, Volume}) {
		t.Errorf("Area·Length = %v, %v; want {12 volume}", got, err)
	}
	// Unitless scaling preserves the unit.
	if got, err := Q(5, Length).Mul(Scalar(2)); err != nil || got != (Quantity{10, Length}) {
		t.Errorf("Length·scalar = %v, %v; want {10 length}", got, err)
	}
}

func TestQuantityDiv(t *testing.T) {
	if got, err := Q(12, Volume).Div(Q(3, Area)); err != nil || got != (Quantity{4, Length}) {
		t.Errorf("Volume/Area = %v, %v; want {4 length}", got, err)
	}
	if _, err := Q(1, Length).Div(Q(0, Length)); err == nil {
		t.Error("division by zero should error")
	}
}

func TestQuantityUnnamedDimensionErrors(t *testing.T) {
	// Volume · Volume → L⁶, which has no named unit.
	if _, err := Q(2, Volume).Mul(Q(2, Volume)); err == nil {
		t.Error("L⁶ product should error (no representable unit)")
	}
}

func TestQuantityNonArithmeticUnitErrors(t *testing.T) {
	if _, err := Q(1, Boolean).Mul(Q(2, Length)); err == nil {
		t.Error("multiplying a boolean quantity should error")
	}
	if Boolean.IsArithmetic() || Text.IsArithmetic() {
		t.Error("boolean/text should not be arithmetic")
	}
	if !Length.IsArithmetic() {
		t.Error("length should be arithmetic")
	}
}
