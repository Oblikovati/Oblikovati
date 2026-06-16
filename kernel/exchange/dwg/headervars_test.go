// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "testing"

// TestParseHeaderVarsINSUNITS validates the header-variable parser against the oracle's
// $INSUNITS for the whole corpus, across both header generations (R2000 single-stream and
// R2018 three-stream). INSUNITS sits ~330 fields into the variable list, so a correct
// value here means every preceding field was consumed with the right type and version
// gating — the property the unit conversion depends on.
func TestParseHeaderVarsINSUNITS(t *testing.T) {
	cases := []struct {
		file string
		want int // oracle $INSUNITS (4 = mm, 0 = unitless)
	}{
		{"testfile-1.dwg", 4}, {"testfile-2.dwg", 4}, {"testfile-3.dwg", 4},
		{"testfile-4.dwg", 4}, {"testfile-5.dwg", 0}, {"testfile-6.dwg", 4},
		{"testfile-7.dwg", 4},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			data := loadTestFile(t, tc.file)
			h, err := ParseFileHeader(data)
			if err != nil {
				t.Fatalf("ParseFileHeader: %v", err)
			}
			sec, err := h.HeaderSection(data)
			if err != nil {
				t.Fatalf("HeaderSection: %v", err)
			}
			hv, err := ParseHeaderVars(sec, h.Version)
			if err != nil {
				t.Fatalf("ParseHeaderVars: %v", err)
			}
			if hv.INSUNITS != tc.want {
				t.Errorf("INSUNITS = %d, want %d", hv.INSUNITS, tc.want)
			}
		})
	}
}

// TestParseHeaderVarsDimScale checks a value past INSUNITS in the field stream is still
// reachable — really an anchor that the R2000 path (which interleaves inline strings and
// handles) stays in sync: testfile-2's DIMSCALE is 50.
func TestParseHeaderVarsDimScale(t *testing.T) {
	data := loadTestFile(t, "testfile-2.dwg")
	h, err := ParseFileHeader(data)
	if err != nil {
		t.Fatal(err)
	}
	sec, err := h.HeaderSection(data)
	if err != nil {
		t.Fatal(err)
	}
	hv, err := ParseHeaderVars(sec, h.Version)
	if err != nil {
		t.Fatal(err)
	}
	if hv.DimScale != 50 {
		t.Errorf("DIMSCALE = %g, want 50", hv.DimScale)
	}
	if hv.LTScale != 50 {
		t.Errorf("LTSCALE = %g, want 50", hv.LTScale)
	}
}

// TestMetersPerUnit covers the $INSUNITS → metres mapping, including the unitless and
// unknown cases that fall back to document units.
func TestMetersPerUnit(t *testing.T) {
	cases := []struct {
		code int
		m    float64
		ok   bool
	}{
		{0, 0, false},     // unitless
		{1, 0.0254, true}, // inches
		{4, 0.001, true},  // mm
		{5, 0.01, true},   // cm
		{6, 1, true},      // m
		{7, 1000, true},   // km
		{99, 0, false},    // unknown
	}
	for _, tc := range cases {
		m, ok := MetersPerUnit(tc.code)
		if ok != tc.ok || (ok && m != tc.m) {
			t.Errorf("MetersPerUnit(%d) = (%g, %v), want (%g, %v)", tc.code, m, ok, tc.m, tc.ok)
		}
	}
}
