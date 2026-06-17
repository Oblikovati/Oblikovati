// SPDX-License-Identifier: GPL-2.0-only

package dxf

import (
	"math"
	"strings"
	"testing"

	"oblikovati.org/kernel/exchange/drawing"
)

// TestEncodeDecodeRoundTrip writes one of each simple entity to a DXF and decodes it back,
// asserting the geometry and unit code survive — the contract the export feature depends on
// (Encode and Decode are inverses for the supported types).
func TestEncodeDecodeRoundTrip(t *testing.T) {
	in := &drawing.Drawing{Units: drawing.INSCentimetres, Entities: []drawing.Entity{
		&drawing.Line{Start: [3]float64{0, 0, 0}, End: [3]float64{10, 5, 0}},
		&drawing.Circle{Center: [3]float64{3, 4, 0}, Radius: 2.5, Normal: [3]float64{0, 0, 1}},
		&drawing.Point{Position: [3]float64{7, 8, 0}},
	}}
	data, err := Encode(in, R2000)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	dr, warns, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
	if dr.Units != drawing.INSCentimetres {
		t.Errorf("units = %d, want %d", dr.Units, drawing.INSCentimetres)
	}
	if len(dr.Entities) != 3 {
		t.Fatalf("entities = %d, want 3", len(dr.Entities))
	}
	l := dr.Entities[0].(*drawing.Line)
	if l.Start != [3]float64{0, 0, 0} || l.End != [3]float64{10, 5, 0} {
		t.Errorf("line round-trip: %+v", l)
	}
	c := dr.Entities[1].(*drawing.Circle)
	if c.Center != [3]float64{3, 4, 0} || c.Radius != 2.5 {
		t.Errorf("circle round-trip: %+v", c)
	}
	p := dr.Entities[2].(*drawing.Point)
	if p.Position != [3]float64{7, 8, 0} {
		t.Errorf("point round-trip: %+v", p)
	}
}

// TestRoundTripBothVersions checks the geometry and unit code survive Encode→Decode for
// both export versions (the entity group codes are version-independent).
func TestRoundTripBothVersions(t *testing.T) {
	in := &drawing.Drawing{Units: drawing.INSMillimetres, Entities: []drawing.Entity{
		&drawing.Line{End: [3]float64{4, 2, 0}},
		&drawing.Arc{Center: [3]float64{1, 1, 0}, Radius: 2, StartAngle: 0.3, EndAngle: 1.1},
		&drawing.Spline{Degree: 3, ControlPoints: [][3]float64{{0, 0, 0}, {1, 1, 0}, {2, 0, 0}, {3, 1, 0}}},
	}}
	for _, v := range []Version{R2000, R2018} {
		data, err := Encode(in, v)
		if err != nil {
			t.Fatalf("Encode(%v): %v", v, err)
		}
		dr, warns, err := Decode(data)
		if err != nil {
			t.Fatalf("Decode(%v): %v", v, err)
		}
		if len(warns) != 0 {
			t.Errorf("version %v warnings: %v", v, warns)
		}
		if dr.Units != drawing.INSMillimetres || len(dr.Entities) != 3 {
			t.Errorf("version %v: units=%d entities=%d", v, dr.Units, len(dr.Entities))
		}
	}
}

// TestR2018EmitsClasses checks R2018 carries a CLASSES section (R2000 omits it).
func TestR2018EmitsClasses(t *testing.T) {
	r18, _ := Encode(&drawing.Drawing{}, R2018)
	if !strings.Contains(string(r18), "\nCLASSES\n") {
		t.Error("R2018 output missing CLASSES section")
	}
	r15, _ := Encode(&drawing.Drawing{}, R2000)
	if strings.Contains(string(r15), "\nCLASSES\n") {
		t.Error("R2000 output should not carry a CLASSES section")
	}
}

// TestEncodeEmitsStandardSections checks the encoder writes the full standard section set,
// not just the bare ENTITIES, so the file opens in AutoCAD without repair.
func TestEncodeEmitsStandardSections(t *testing.T) {
	data, err := Encode(&drawing.Drawing{}, R2000)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := string(data)
	for _, want := range []string{
		"\nHEADER\n", "\nTABLES\n", "\nBLOCKS\n", "\nENTITIES\n", "\nOBJECTS\n", "\nEOF\n",
		"\nAC1015\n", "\nVPORT\n", "\nLTYPE\n", "\nLAYER\n", "\n*Model_Space\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

// TestEncodeVersionACADVer checks the version selector drives $ACADVER (R2000→AC1015,
// R2018→AC1032).
func TestEncodeVersionACADVer(t *testing.T) {
	for v, want := range map[Version]string{R2000: "AC1015", R2018: "AC1032"} {
		data, err := Encode(&drawing.Drawing{}, v)
		if err != nil {
			t.Fatalf("Encode(%v): %v", v, err)
		}
		if !strings.Contains(string(data), "\n"+want+"\n") {
			t.Errorf("version %v: $ACADVER not %s", v, want)
		}
	}
}

// TestFormatFloat checks integers gain a decimal point (the DXF convention) and fractions
// are preserved exactly.
func TestFormatFloat(t *testing.T) {
	cases := map[float64]string{0: "0.0", 10: "10.0", 2.5: "2.5", -3: "-3.0", 0.1: "0.1"}
	for in, want := range cases {
		if got := formatFloat(in); got != want {
			t.Errorf("formatFloat(%g) = %q, want %q", in, got, want)
		}
	}
	if math.Abs(mustFloat(t, formatFloat(123456.789))-123456.789) > 1e-9 {
		t.Error("large value did not round-trip through formatFloat")
	}
}

func mustFloat(t *testing.T, s string) float64 {
	t.Helper()
	v, err := pair{code: 0, value: s}.float()
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return v
}
