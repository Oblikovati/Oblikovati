// SPDX-License-Identifier: GPL-2.0-only

package dxf

import (
	"math"
	"testing"

	"oblikovati.org/kernel/exchange/drawing"
)

// TestArcEllipseRoundTrip checks ARC and ELLIPSE survive Encode→Decode with their angles.
func TestArcEllipseRoundTrip(t *testing.T) {
	in := &drawing.Drawing{Entities: []drawing.Entity{
		&drawing.Arc{Center: [3]float64{1, 1, 0}, Radius: 3, StartAngle: 0.5, EndAngle: 2.0, Normal: [3]float64{0, 0, 1}},
		&drawing.Ellipse{Center: [3]float64{1, 1, 0}, MajorAxis: [3]float64{2, 0, 0}, AxisRatio: 0.5, StartAngle: 0, EndAngle: 1.2, Normal: [3]float64{0, 0, 1}},
	}}
	dr := reEncode(t, in)
	if len(dr.Entities) != 2 {
		t.Fatalf("entities = %d, want 2", len(dr.Entities))
	}
	a := dr.Entities[0].(*drawing.Arc)
	if !near(a.Radius, 3) || !near(a.StartAngle, 0.5) || !near(a.EndAngle, 2.0) {
		t.Errorf("arc round-trip: r=%g [%g,%g]", a.Radius, a.StartAngle, a.EndAngle)
	}
	e := dr.Entities[1].(*drawing.Ellipse)
	if !near(e.AxisRatio, 0.5) || !near(e.StartAngle, 0) || !near(e.EndAngle, 1.2) {
		t.Errorf("ellipse round-trip: ratio=%g [%g,%g]", e.AxisRatio, e.StartAngle, e.EndAngle)
	}
	if e.MajorAxis != [3]float64{2, 0, 0} {
		t.Errorf("ellipse major axis = %v", e.MajorAxis)
	}
}

// TestAngleAsymmetry locks the DXF quirk that ARC angles are stored in degrees (codes
// 50/51) while ELLIPSE parametric angles are stored in radians (codes 41/42): a π/2 model
// angle must encode as 90 on an ARC but stay ~1.5708 on an ELLIPSE.
func TestAngleAsymmetry(t *testing.T) {
	data, err := Encode(&drawing.Drawing{Entities: []drawing.Entity{
		&drawing.Arc{StartAngle: math.Pi / 2, EndAngle: math.Pi, Radius: 1},
		&drawing.Ellipse{StartAngle: math.Pi / 2, EndAngle: math.Pi, MajorAxis: [3]float64{1, 0, 0}, AxisRatio: 0.5},
	}}, R2000)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	pairs, err := scanPairs(data)
	if err != nil {
		t.Fatalf("scanPairs: %v", err)
	}
	arc := entityBody(t, pairs, "ARC")
	if v, _ := arc[50].float(); !near(v, 90) {
		t.Errorf("ARC code 50 = %g, want 90 (degrees)", v)
	}
	ell := entityBody(t, pairs, "ELLIPSE")
	if v, _ := ell[41].float(); !near(v, math.Pi/2) {
		t.Errorf("ELLIPSE code 41 = %g, want π/2 (radians)", v)
	}
}

// reEncode encodes a drawing and decodes it back, failing the test on any error.
func reEncode(t *testing.T, in *drawing.Drawing) *drawing.Drawing {
	t.Helper()
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
	return dr
}

// entityBody finds the first entity of the given type in an encoded file's pairs and returns
// its group codes indexed by code.
func entityBody(t *testing.T, pairs []pair, name string) map[int]pair {
	t.Helper()
	for _, g := range splitEntities(pairs) {
		if g.name == name {
			return indexByCode(g.body)
		}
	}
	t.Fatalf("no %s entity in output", name)
	return nil
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
