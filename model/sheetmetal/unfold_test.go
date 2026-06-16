// SPDX-License-Identifier: GPL-2.0-only

package sheetmetal

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
)

const tol = 1e-9

// TestKFactorBendAllowance pins the canonical K-factor relation BA = angle·(r + K·t) for a
// 90° bend, and that bend deduction BD = 2·OSSB − BA with OSSB = (r+t)·tan(45°) = r+t.
func TestKFactorBendAllowance(t *testing.T) {
	m := KFactorMethod(0.44)
	const r, th = 0.2, 0.1 // cm
	angle := math.Pi / 2

	wantBA := angle * (r + 0.44*th)
	if got := m.BendAllowance(angle, r, th); math.Abs(got-wantBA) > tol {
		t.Errorf("BendAllowance(90°) = %.9f, want %.9f", got, wantBA)
	}
	wantBD := 2*(r+th) - wantBA // tan(45°)=1
	if got := m.BendDeduction(angle, r, th); math.Abs(got-wantBD) > tol {
		t.Errorf("BendDeduction(90°) = %.9f, want %.9f", got, wantBD)
	}
}

// TestKFactorChangesFlatExtents is the F01 acceptance criterion: changing K-factor changes
// the developed length. A larger K pushes the neutral axis outward, lengthening the flat.
func TestKFactorChangesFlatExtents(t *testing.T) {
	const r, th = 0.2, 0.1
	angle := math.Pi / 2
	lo := KFactorMethod(0.33).BendAllowance(angle, r, th)
	hi := KFactorMethod(0.50).BendAllowance(angle, r, th)
	if !(hi > lo) {
		t.Errorf("higher K-factor must lengthen the flat: K=0.50 gave %.6f, K=0.33 gave %.6f", hi, lo)
	}
	delta := angle * (0.50 - 0.33) * th
	if math.Abs((hi-lo)-delta) > tol {
		t.Errorf("flat-length delta = %.9f, want %.9f", hi-lo, delta)
	}
}

// TestKFactorMethodDefaults a non-positive K-factor falls back to the standard default.
func TestKFactorMethodDefaults(t *testing.T) {
	if got := KFactorMethod(0).kFactor(); got != defaultKFactor {
		t.Errorf("KFactorMethod(0).kFactor() = %v, want default %v", got, defaultKFactor)
	}
	if got := KFactorMethod(-1).kFactor(); got != defaultKFactor {
		t.Errorf("KFactorMethod(-1).kFactor() = %v, want default %v", got, defaultKFactor)
	}
}

// TestEquationMethod evaluates a custom bend-allowance equation in t/r/a and confirms it is
// used over the K-factor fallback.
func TestEquationMethod(t *testing.T) {
	// A deliberately distinct equation: BA = a · (r + 0.3·t).
	m, err := EquationMethod("a * (r + 0.3 * t)")
	if err != nil {
		t.Fatalf("EquationMethod: %v", err)
	}
	if m.Type != types.EquationUnfold {
		t.Fatalf("Type = %v, want EquationUnfold", m.Type)
	}
	const r, th = 0.2, 0.1
	angle := math.Pi / 2
	want := angle * (r + 0.3*th)
	if got := m.BendAllowance(angle, r, th); math.Abs(got-want) > tol {
		t.Errorf("equation BendAllowance = %.9f, want %.9f", got, want)
	}
}

// TestEquationMethodRejectsUnknownVariable a typo'd variable fails at compile time, naming it.
func TestEquationMethodRejectsUnknownVariable(t *testing.T) {
	_, err := EquationMethod("a * (radius + bogus)")
	if err == nil {
		t.Fatal("EquationMethod accepted an unknown variable, want error")
	}
}

// TestBendTableInterpolation a table characterised at 60° and 90° interpolates linearly at
// 75° and holds the endpoints outside the sampled range.
func TestBendTableInterpolation(t *testing.T) {
	const r, th = 0.2, 0.1
	table := NewBendTable([]BendTableRow{
		{Angle: math.Pi / 3, Radius: r, Thickness: th, Allowance: 0.30}, // 60°
		{Angle: math.Pi / 2, Radius: r, Thickness: th, Allowance: 0.40}, // 90°
	})
	m := BendTableMethod(table)

	mid := (math.Pi/3 + math.Pi/2) / 2 // 75°
	if got := m.BendAllowance(mid, r, th); math.Abs(got-0.35) > 1e-9 {
		t.Errorf("table allowance at 75° = %.9f, want 0.35 (linear midpoint)", got)
	}
	if got := m.BendAllowance(math.Pi/6, r, th); math.Abs(got-0.30) > 1e-9 {
		t.Errorf("table allowance below range = %.9f, want held 0.30", got)
	}
}

// TestBendTableFallsBackWhenNoRowMatches an unmatched radius/thickness reverts to K-factor.
func TestBendTableFallsBackWhenNoRowMatches(t *testing.T) {
	table := NewBendTable([]BendTableRow{{Angle: math.Pi / 2, Radius: 1.0, Thickness: 1.0, Allowance: 9.9}})
	m := BendTableMethod(table)
	angle := math.Pi / 2
	const r, th = 0.2, 0.1
	want := angle * (r + defaultKFactor*th) // K-factor fallback
	if got := m.BendAllowance(angle, r, th); math.Abs(got-want) > tol {
		t.Errorf("unmatched table row = %.9f, want K-factor fallback %.9f", got, want)
	}
}
