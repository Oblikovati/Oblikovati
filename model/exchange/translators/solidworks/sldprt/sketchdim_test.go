// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"math"
	"testing"
)

// TestDimensionDecode checks decoded sketch dimensions against generated known values: a rectangle
// dimensioned 25mm x 15mm yields D1 = 0.025 m and D2 = 0.015 m; a circle dimensioned Ø8mm yields a
// 0.016 m diameter (2 x radius). Values validated against the SolidWorks 2026 DisplayDimension.
func TestDimensionDecode(t *testing.T) {
	cases := []struct {
		file string
		want map[string]float64
	}{
		{"dimrect_fmtb.sldprt", map[string]float64{"D1": 0.025, "D2": 0.015}},
		{"dimcirc_fmtb.sldprt", map[string]float64{"D1": 0.016}},
		{"angledim_fmtb.sldprt", map[string]float64{"D1": 40 * math.Pi / 180}}, // 40 deg, stored in radians
	}
	for _, c := range cases {
		d, err := Open(readTestdata(t, c.file))
		if err != nil {
			t.Fatalf("Open %s: %v", c.file, err)
		}
		got := map[string]float64{}
		for _, dm := range d.Sketches()[0].Dimensions {
			got[dm.Name] = dm.Value
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: got %d dimensions %v, want %v", c.file, len(got), got, c.want)
		}
		for name, v := range c.want {
			if math.Abs(got[name]-v) > 1e-9 {
				t.Errorf("%s: %s = %g, want %g", c.file, name, got[name], v)
			}
		}
	}
}

// TestNoDimensions verifies a sketch with only constraints (no dimensions) yields none.
func TestNoDimensions(t *testing.T) {
	d, err := Open(readTestdata(t, "box10_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if n := len(d.Sketches()[0].Dimensions); n != 0 {
		t.Errorf("got %d dimensions, want 0", n)
	}
}

func TestDimNameAt(t *testing.T) {
	// ff fe ff, length 2, "D1" (UTF-16LE) -> name "D1".
	region := []byte{0xff, 0xfe, 0xff, 0x02, 'D', 0, '1', 0}
	name, end, ok := dimNameAt(region, 0)
	if !ok || name != "D1" || end != 8 {
		t.Errorf("dimNameAt = (%q,%d,%v), want (D1,8,true)", name, end, ok)
	}
	// A non-D name is rejected.
	if _, _, ok := dimNameAt([]byte{0xff, 0xfe, 0xff, 0x02, 'X', 0, '1', 0}, 0); ok {
		t.Error("dimNameAt accepted a non-dimension name")
	}
}
