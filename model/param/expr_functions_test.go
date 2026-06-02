// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	stdmath "math"
	"testing"
)

// These exercise dimensional function behavior that cannot be written with
// expression literals (area/volume units use a '^' the lexer reserves), by
// calling the builtins directly.

func TestSqrtHalvesDimension(t *testing.T) {
	q, err := functions["sqrt"]([]Quantity{Q(4, Area)})
	if err != nil || q != (Quantity{2, Length}) {
		t.Errorf("sqrt(4 area) = %v, %v; want {2 length}", q, err)
	}
	// Volume's exponent (L³) is odd, so it has no square root unit.
	if _, err := functions["sqrt"]([]Quantity{Q(8, Volume)}); err == nil {
		t.Error("sqrt(volume) should error (odd dimension)")
	}
}

func TestAtan2Quadrant(t *testing.T) {
	q, err := functions["atan2"]([]Quantity{Q(1, Length), Q(1, Length)})
	if err != nil || !approxScalar(q.Value, stdmath.Pi/4) || q.Unit != Angle {
		t.Errorf("atan2(1,1) = %v, %v; want π/4 angle", q, err)
	}
	if _, err := functions["atan2"]([]Quantity{Q(1, Length), Q(1, Angle)}); err == nil {
		t.Error("atan2 with mixed units should error")
	}
}

func TestAbsPreservesUnit(t *testing.T) {
	q, err := functions["abs"]([]Quantity{Q(-3, Length)})
	if err != nil || q != (Quantity{3, Length}) {
		t.Errorf("abs(-3 length) = %v, %v; want {3 length}", q, err)
	}
}

func TestArgCountErrors(t *testing.T) {
	if _, err := functions["sin"](nil); err == nil {
		t.Error("sin with no args should error")
	}
	if _, err := functions["min"]([]Quantity{Q(1, Length)}); err == nil {
		t.Error("min with one arg should error")
	}
}
