// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/exchange/step/part21"
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
	t.Parallel()
	stmts := "#1=(LENGTH_UNIT()NAMED_UNIT(*)SI_UNIT(.MILLI.,.METRE.));\n" +
		"#2=GLOBAL_UNIT_ASSIGNED_CONTEXT((#1));"
	scale, found := parseUnits(t, stmts)
	if !found || scale != 1.0 {
		t.Errorf("mm scale = %g (found=%v), want 1.0", scale, found)
	}
}

func TestMetreUnitScale(t *testing.T) {
	t.Parallel()
	stmts := "#1=(LENGTH_UNIT()NAMED_UNIT(*)SI_UNIT($,.METRE.));\n" +
		"#2=GLOBAL_UNIT_ASSIGNED_CONTEXT((#1));"
	scale, found := parseUnits(t, stmts)
	if !found || scale != 1000.0 {
		t.Errorf("metre scale = %g (found=%v), want 1000.0", scale, found)
	}
}

func TestSIPrefixUnitScales(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		prefix string
		want   float64
	}{
		{".CENTI.", 10},
		{".DECI.", 100},
		{".KILO.", 1_000_000},
		{".MICRO.", 0.001},
	} {
		stmts := "#1=(LENGTH_UNIT()NAMED_UNIT(*)SI_UNIT(" + tc.prefix + ",.METRE.));\n" +
			"#2=GLOBAL_UNIT_ASSIGNED_CONTEXT((#1));"
		scale, found := parseUnits(t, stmts)
		if !found || !approx(scale, tc.want, 1e-12) {
			t.Fatalf("%s scale = %g (found=%v), want %g", tc.prefix, scale, found, tc.want)
		}
	}
}

func TestInchConversionUnitScale(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	scale, found := parseUnits(t, "#1=CARTESIAN_POINT('',(0.,0.,0.));")
	if found {
		t.Error("a file with no unit context should report found=false")
	}
	if scale != 1.0 {
		t.Errorf("default scale = %g, want 1.0 (mm)", scale)
	}
}

func TestUnitResolutionFailureWarnsAndAssumesMillimeters(t *testing.T) {
	t.Parallel()
	g := parseUnitGraph(t, "#1=CONVERSION_BASED_UNIT('BAD','not-a-ref');\n"+
		"#2=GLOBAL_UNIT_ASSIGNED_CONTEXT((#1));")
	scale, warns := unitScale(g)
	if scale != 0 {
		t.Fatalf("failed unit scale = %g, want raw failed scale 0", scale)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "unit resolution failed") {
		t.Fatalf("warnings = %v", warns)
	}
}

func TestUnitHelperBranches(t *testing.T) {
	t.Parallel()
	g := parseUnitGraph(t, "#1=GLOBAL_UNIT_ASSIGNED_CONTEXT(('not-a-ref',#999));\n"+
		"#2=SI_UNIT(.MILLI.,.METRE.);\n"+
		"#3=(LENGTH_UNIT()NAMED_UNIT(*)SI_UNIT(.MILLI.,.METRE.));")
	ent, err := g.Lookup(1)
	if err != nil {
		t.Fatalf("lookup context: %v", err)
	}
	refs := refsInFirstList(ent.Params)
	if len(refs) != 1 || refs[0] != 999 {
		t.Fatalf("refsInFirstList = %v, want [999]", refs)
	}
	si, err := g.Lookup(2)
	if err != nil {
		t.Fatalf("lookup si: %v", err)
	}
	if !enumContains(si.Params, "metre") || enumContains(si.Params, "second") {
		t.Fatalf("enumContains did not match expected SI params")
	}
	complexUnit, err := g.Lookup(3)
	if err != nil {
		t.Fatalf("lookup complex unit: %v", err)
	}
	if !hasComponentNamed(complexUnit, "LENGTH_UNIT") || !hasLengthComponent(complexUnit) {
		t.Fatalf("complex length-unit helpers failed for %#v", complexUnit.Components)
	}
	if isLengthUnit(g, 999) {
		t.Fatal("missing unit ref reported as length unit")
	}
}

func parseUnitGraph(t *testing.T, stmts string) *part21.EntityGraph {
	t.Helper()
	src := "ISO-10303-21;\nHEADER;\n" +
		"FILE_DESCRIPTION((''),'');\nFILE_NAME('','',(''),(''),'','','');\n" +
		"FILE_SCHEMA(('CONFIG_CONTROL_DESIGN'));\nENDSEC;\nDATA;\n" + stmts +
		"\nENDSEC;\nEND-ISO-10303-21;\n"
	f, err := part21.Parse([]byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return f.Graph
}
