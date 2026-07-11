// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	stdmath "math"
	"testing"
)

// TestISOFitBandCanonicalValues validates the ISO 286 band resolution against published
// ISO 286-2 fit-table values across sizes and classes (#1848). These are the authoritative
// numbers a mechanical engineer reads from the standard; if the IT table or a
// fundamental-deviation formula is wrong, one of these fails. Bands are in millimetres.
func TestISOFitBandCanonicalValues(t *testing.T) {
	cases := []struct {
		nominal            float64
		class              string
		wantUpper, wantLow float64 // mm
	}{
		{50, "H7", 0.025, 0},       // basic hole
		{50, "g6", -0.009, -0.025}, // sliding-fit shaft
		{50, "f7", -0.025, -0.050}, // running-fit shaft
		{50, "h6", 0, -0.016},      // location shaft
		{50, "d9", -0.080, -0.142}, // loose running shaft
		{25, "H7", 0.021, 0},
		{25, "f7", -0.020, -0.041},
		{10, "H7", 0.015, 0},
		{10, "g6", -0.005, -0.014},
		{5, "f7", -0.010, -0.022},
		{50, "F8", 0.064, 0.025}, // clearance hole
		{30, "h7", 0, -0.021},
		{100, "g6", -0.012, -0.034},
	}
	for _, c := range cases {
		up, lo, err := ISOFitBand(c.nominal, c.class)
		if err != nil {
			t.Errorf("ISOFitBand(%g, %q) error: %v", c.nominal, c.class, err)
			continue
		}
		if !closeMM(up, c.wantUpper) || !closeMM(lo, c.wantLow) {
			t.Errorf("ISOFitBand(%g, %q) = %+.4f/%+.4f mm, want %+.4f/%+.4f", c.nominal, c.class, up, lo, c.wantUpper, c.wantLow)
		}
	}
}

// TestISOFitBandSymmetricJS: a js class is a symmetric ±IT/2 band (#1848). 50js7 → ±IT7/2
// = ±25/2 = ±12.5 µm.
func TestISOFitBandSymmetricJS(t *testing.T) {
	up, lo, err := ISOFitBand(50, "js7")
	if err != nil {
		t.Fatalf("ISOFitBand js7: %v", err)
	}
	if !closeMM(up, 0.0125) || !closeMM(lo, -0.0125) {
		t.Errorf("50js7 = %+.4f/%+.4f mm, want ±0.0125", up, lo)
	}
}

// TestISOFitBandRejectsUnsupported: out-of-range sizes, grades, and letters error with a
// clear message rather than returning a dubious band (CLAUDE.md correctness bar, #1848).
func TestISOFitBandRejectsUnsupported(t *testing.T) {
	for _, tc := range []struct {
		nominal     float64
		class, want string
	}{
		{2, "H7", "out of range"},                // ≤3 mm unsupported
		{600, "H7", "out of range"},              // >500 mm unsupported
		{50, "H2", "IT grade"},                   // grade below IT5
		{50, "H14", "IT grade"},                  // grade above IT11
		{50, "p6", "unsupported ISO fit letter"}, // interference letter not modelled
		{50, "H", "malformed"},                   // no grade
		{50, "7", "malformed"},                   // no letter
	} {
		if _, _, err := ISOFitBand(tc.nominal, tc.class); err == nil {
			t.Errorf("ISOFitBand(%g, %q) = nil error, want one mentioning %q", tc.nominal, tc.class, tc.want)
		}
	}
}

// closeMM compares two millimetre deviations within half a micrometre (the tables are whole µm).
func closeMM(got, want float64) bool { return stdmath.Abs(got-want) < 0.0005 }
