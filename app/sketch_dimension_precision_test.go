// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/model/param"
)

// TestDimensionSeedRespectsPrecision: a dimension placed on geometry measured with float noise
// seeds a CLEAN, precision-rounded expression ("10 mm"), not the raw "9.999999998 mm" — so the
// stored parameter carries the on-screen number (Oblikovati/Oblikovati#146 follow-up).
func TestDimensionSeedRespectsPrecision(t *testing.T) {
	units := param.DefaultUnitsOfMeasure() // mm, 3 length decimals; deg, 2 angle decimals
	if got := lengthExpr(units, 0.9999999998); got != "10 mm" {
		t.Errorf("length seed = %q, want \"10 mm\"", got)
	}
	// 30° with float noise → clean "30 deg".
	if got := angleExpr(units, stdmath.Pi/6+1e-12); got != "30 deg" {
		t.Errorf("angle seed = %q, want \"30 deg\"", got)
	}
}

// TestDimensionLabelRespectsPrecision: the on-screen label is formatted DOWN from the live
// measured float to the document's display precision, never printed losslessly.
func TestDimensionLabelRespectsPrecision(t *testing.T) {
	units := param.DefaultUnitsOfMeasure()
	if got := lengthLabel(units, 0.9999999998); got != "10.000 mm" {
		t.Errorf("length label = %q, want \"10.000 mm\"", got)
	}
	if got := angleLabel(units, stdmath.Pi/6+1e-12); got != "30.00 deg" {
		t.Errorf("angle label = %q, want \"30.00 deg\"", got)
	}
}
