// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	"testing"

	"oblikovati/kernel/exchange/step/part21"
)

// parseUnits parses a DATA-only fixture and returns its length scale to mm.
func parseUnits(t *testing.T, stmts string) (float64, bool) {
	t.Helper()
	src := "ISO-10303-21;\nHEADER;\n" +
		"FILE_DESCRIPTION((''),'');\nFILE_NAME('','',(''),(''),'','','');\n" +
		"FILE_SCHEMA(('CONFIG_CONTROL_DESIGN'));\nENDSEC;\nDATA;\n" + stmts +
		"\nENDSEC;\nEND-ISO-10303-21;\n"
	f, err := part21.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	scale, found, err := mmPerLengthUnit(f.Graph)
	if err != nil {
		t.Fatalf("mmPerLengthUnit: %v", err)
	}
	return scale, found
}

func TestMillimeterUnitScale(t *testing.T) {
	stmts := "#1=(LENGTH_UNIT()NAMED_UNIT(*)SI_UNIT(.MILLI.,.METRE.));\n" +
		"#2=GLOBAL_UNIT_ASSIGNED_CONTEXT((#1));"
	scale, found := parseUnits(t, stmts)
	if !found || scale != 1.0 {
		t.Errorf("mm scale = %g (found=%v), want 1.0", scale, found)
	}
}

func TestMetreUnitScale(t *testing.T) {
	stmts := "#1=(LENGTH_UNIT()NAMED_UNIT(*)SI_UNIT($,.METRE.));\n" +
		"#2=GLOBAL_UNIT_ASSIGNED_CONTEXT((#1));"
	scale, found := parseUnits(t, stmts)
	if !found || scale != 1000.0 {
		t.Errorf("metre scale = %g (found=%v), want 1000.0", scale, found)
	}
}

func TestInchConversionUnitScale(t *testing.T) {
	stmts := "#1=(LENGTH_UNIT()NAMED_UNIT(*)SI_UNIT($,.METRE.));\n" +
		"#2=LENGTH_MEASURE_WITH_UNIT(LENGTH_MEASURE(0.0254),#1);\n" +
		"#3=(CONVERSION_BASED_UNIT('INCH',#2)LENGTH_UNIT()NAMED_UNIT(*));\n" +
		"#4=GLOBAL_UNIT_ASSIGNED_CONTEXT((#3));"
	scale, found := parseUnits(t, stmts)
	const want = 25.4 // one inch in mm
	if !found || !approx(scale, want, 1e-9) {
		t.Errorf("inch scale = %g (found=%v), want %g", scale, found, want)
	}
}

func TestNoUnitDefaultsToMillimeters(t *testing.T) {
	scale, found := parseUnits(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));")
	if found {
		t.Error("a file with no unit context should report found=false")
	}
	if scale != 1.0 {
		t.Errorf("default scale = %g, want 1.0 (mm)", scale)
	}
}
