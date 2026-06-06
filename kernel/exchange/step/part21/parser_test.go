// SPDX-License-Identifier: GPL-2.0-only

package part21

import "testing"

// tinyFile is a minimal but complete Part 21 file used across parser tests.
const tinyFile = `ISO-10303-21;
HEADER;
FILE_DESCRIPTION(('a cube'),'2;1');
FILE_NAME('cube.step','2026-01-01T00:00:00',('me'),('acme'),'pp 1','sys 1','');
FILE_SCHEMA(('CONFIG_CONTROL_DESIGN'));
ENDSEC;
DATA;
#1=CARTESIAN_POINT('',(0.,0.,0.));
#2=DIRECTION('',(0.,0.,1.));
#3=AXIS2_PLACEMENT_3D('',#1,#2,$);
#4=PLANE('',#3);
ENDSEC;
END-ISO-10303-21;
`

func TestParseTinyFile(t *testing.T) {
	f, err := Parse([]byte(tinyFile))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if f.Graph.Len() != 4 {
		t.Errorf("graph has %d entities, want 4", f.Graph.Len())
	}
	plane, err := f.Graph.Lookup(4)
	if err != nil {
		t.Fatalf("lookup #4: %v", err)
	}
	if plane.Keyword != "PLANE" {
		t.Errorf("#4 keyword = %q, want PLANE", plane.Keyword)
	}
}

func TestParseHeaderFields(t *testing.T) {
	f, err := Parse([]byte(tinyFile))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := f.Header.Description; len(got) != 1 || got[0] != "a cube" {
		t.Errorf("description = %v, want [a cube]", got)
	}
	if f.Header.Name != "cube.step" {
		t.Errorf("name = %q, want cube.step", f.Header.Name)
	}
	if got := f.Header.SchemaIdentifiers; len(got) != 1 || got[0] != "CONFIG_CONTROL_DESIGN" {
		t.Errorf("schema = %v, want [CONFIG_CONTROL_DESIGN]", got)
	}
}

func TestParseResolvesReferences(t *testing.T) {
	f, _ := Parse([]byte(tinyFile))
	placement, _ := f.Graph.Lookup(3)
	loc, err := placement.Params[1].AsRef()
	if err != nil {
		t.Fatalf("placement location ref: %v", err)
	}
	pt, err := f.Graph.Lookup(loc)
	if err != nil || pt.Keyword != "CARTESIAN_POINT" {
		t.Errorf("resolved location = %v (%v), want CARTESIAN_POINT", pt, err)
	}
}

func TestParseNullParameter(t *testing.T) {
	f, _ := Parse([]byte(tinyFile))
	placement, _ := f.Graph.Lookup(3)
	if !placement.Params[3].IsNull() {
		t.Error("AXIS2_PLACEMENT_3D ref_direction should parse as $ (null)")
	}
}

func TestParseNestedCoordinates(t *testing.T) {
	f, _ := Parse([]byte(tinyFile))
	pt, _ := f.Graph.Lookup(1)
	coords, err := pt.Params[1].AsList()
	if err != nil {
		t.Fatalf("coordinates list: %v", err)
	}
	if len(coords) != 3 {
		t.Fatalf("got %d coordinates, want 3", len(coords))
	}
	if x, _ := coords[0].AsFloat(); x != 0.0 {
		t.Errorf("x = %v, want 0", x)
	}
}

func TestParseComplexInstance(t *testing.T) {
	src := wrapData("#5=(NAMED_UNIT(*)SI_UNIT(.MILLI.,.METRE.)LENGTH_UNIT());")
	f, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse complex: %v", err)
	}
	ent, _ := f.Graph.Lookup(5)
	if len(ent.Components) != 3 {
		t.Fatalf("complex instance has %d components, want 3", len(ent.Components))
	}
	if ent.Components[1].Keyword != "SI_UNIT" {
		t.Errorf("component[1] = %q, want SI_UNIT", ent.Components[1].Keyword)
	}
}

func TestParseDanglingReferenceCaughtOnLookup(t *testing.T) {
	f, _ := Parse([]byte(tinyFile))
	if _, err := f.Graph.Lookup(999); err == nil {
		t.Error("lookup of a missing id should error, not panic")
	}
}

func TestParseMalformedMissingSemicolon(t *testing.T) {
	src := `ISO-10303-21;
HEADER;
FILE_DESCRIPTION((''),'')
ENDSEC;`
	if _, err := Parse([]byte(src)); err == nil {
		t.Error("missing semicolon should be a parse error")
	}
}

func TestParseDuplicateIDErrors(t *testing.T) {
	src := wrapData("#1=DIRECTION('',(0.,0.,1.));\n#1=DIRECTION('',(1.,0.,0.));")
	if _, err := Parse([]byte(src)); err == nil {
		t.Error("duplicate entity id should error")
	}
}

// wrapData wraps DATA statements in a minimal valid file shell.
func wrapData(stmts string) string {
	return "ISO-10303-21;\nHEADER;\n" +
		"FILE_DESCRIPTION((''),'');\n" +
		"FILE_NAME('','',(''),(''),'','','');\n" +
		"FILE_SCHEMA(('CONFIG_CONTROL_DESIGN'));\nENDSEC;\n" +
		"DATA;\n" + stmts + "\nENDSEC;\nEND-ISO-10303-21;\n"
}
