// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// TestUnitScales pins the database↔file length scaling helpers (#146). The
// database unit is the centimetre (DBUnitMM = 10 mm).
func TestUnitScales(t *testing.T) {
	if DBUnitMM != 10 {
		t.Fatalf("DBUnitMM = %g, want 10 (the kernel works in centimetres)", DBUnitMM)
	}

	cm := TranslationOptions{TargetUnitMM: DBUnitMM}
	// Import: a millimetre file (1 mm per unit) → ×0.1 into cm; an inch file → ×2.54.
	if got := cm.ImportScale(1); !approx(got, 0.1) {
		t.Errorf("ImportScale(mm) = %g, want 0.1", got)
	}
	if got := cm.ImportScale(25.4); !approx(got, 2.54) {
		t.Errorf("ImportScale(in) = %g, want 2.54", got)
	}

	// Export: cm → mm is ×10, cm → cm is ×1, cm → inch is /2.54.
	for unit, want := range map[string]float64{"mm": 10, "cm": 1, "m": 0.01, "in": 10.0 / 25.4} {
		o := TranslationOptions{TargetUnitMM: DBUnitMM, FileUnit: unit}
		if got := o.ExportScale(); !approx(got, want) {
			t.Errorf("ExportScale(%q) = %g, want %g", unit, got, want)
		}
	}

	// FileUnitMM reports the unit's mm size and recognition; unknown is rejected.
	if mm, ok := cm.FileUnitMM(); !ok || mm != 1 { // empty FileUnit ⇒ mm
		t.Errorf("FileUnitMM(empty) = (%g, %v), want (1, true)", mm, ok)
	}
	if mm, ok := (TranslationOptions{FileUnit: "in"}).FileUnitMM(); !ok || mm != 25.4 {
		t.Errorf("FileUnitMM(in) = (%g, %v), want (25.4, true)", mm, ok)
	}
	if _, ok := (TranslationOptions{FileUnit: "furlong"}).FileUnitMM(); ok {
		t.Error("FileUnitMM must reject an unknown unit")
	}

	// A zero TargetUnitMM defaults to the historical 1 mm kernel, so direct callers
	// that pass the zero value are unaffected.
	if got := (TranslationOptions{}).ImportScale(1); !approx(got, 1) {
		t.Errorf("zero-value ImportScale(mm) = %g, want 1", got)
	}
}
