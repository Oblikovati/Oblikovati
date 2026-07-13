// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

func TestExtractBrepTopologyMatchesOracle(t *testing.T) {
	cases := []struct {
		file                  string
		points, planes, cones int
	}{
		{"10_box.ipt", 8, 6, 0},
		{"15_cylinder.ipt", 2, 2, 1},
		{"16_revolve.ipt", 4, 2, 2},
	}
	for _, tc := range cases {
		d := openDoc(t, tc.file)
		seg, ok := d.Segment("PmBRepSegment")
		if !ok {
			t.Fatalf("%s: no PmBRepSegment", tc.file)
		}
		b := ExtractBrep(seg)
		if len(b.Points) != tc.points || len(b.Planes) != tc.planes || len(b.Cones) != tc.cones {
			t.Errorf("%s: got points=%d planes=%d cones=%d, want %d/%d/%d",
				tc.file, len(b.Points), len(b.Planes), len(b.Cones), tc.points, tc.planes, tc.cones)
		}
	}
}

// TestBoxVerticesMatchDimensions checks the extracted corners span 40x20x10 mm (4x2x1 cm).
func TestBoxVerticesMatchDimensions(t *testing.T) {
	d := openDoc(t, "10_box.ipt")
	seg, _ := d.Segment("PmBRepSegment")
	b := ExtractBrep(seg)
	min, max := [3]float64{1e9, 1e9, 1e9}, [3]float64{-1e9, -1e9, -1e9}
	for _, p := range b.Points {
		for k := 0; k < 3; k++ {
			if p[k] < min[k] {
				min[k] = p[k]
			}
			if p[k] > max[k] {
				max[k] = p[k]
			}
		}
	}
	want := [3]float64{4, 2, 1}
	for k := 0; k < 3; k++ {
		if min[k] != 0 || max[k] != want[k] {
			t.Errorf("axis %d span [%.3g,%.3g] cm, want [0,%.3g]", k, min[k], max[k], want[k])
		}
	}
}
