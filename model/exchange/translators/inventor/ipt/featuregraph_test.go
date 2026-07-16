// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

// TestExtrudeDepths pins the depth binding on parts whose depth is known independently: the
// generated corpus authored the distance explicitly, and a revolve must yield NO extrude depth
// (its property 2 is not a DirectionAxis).
func TestExtrudeDepths(t *testing.T) {
	cases := []struct {
		file string
		want []float64
	}{
		{"10_box.ipt", []float64{1.0}},          // rectangle extruded 1 cm
		{"15_cylinder.ipt", []float64{2.0}},     // circle extruded 2 cm
		{"14_box_two.ipt", []float64{1.0, 2.0}}, // two extrudes, 1 cm then 2 cm — the positional
		{"16_revolve.ipt", nil},                 // guess bound BOTH to the first parameter
	}
	for _, tc := range cases {
		got := ExtrudeDepths(openDoc(t, tc.file))
		if len(got) != len(tc.want) {
			t.Errorf("%s: got %v depths, want %v", tc.file, got, tc.want)
			continue
		}
		for i := range got {
			if absf(got[i]-tc.want[i]) > 1e-9 {
				t.Errorf("%s: depth[%d] = %g cm, want %g", tc.file, i, got[i], tc.want[i])
			}
		}
	}
}

// TestExtrudeProfiles pins the profile binding: each extrude names the sketch it consumes, so
// 14_box_two's two extrudes take sketch 0 and sketch 1 respectively.
func TestExtrudeProfiles(t *testing.T) {
	if got := ExtrudeProfiles(openDoc(t, "14_box_two.ipt")); len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Errorf("profiles = %v, want [0 1]", got)
	}
	if got := ExtrudeProfiles(openDoc(t, "10_box.ipt")); len(got) != 1 || got[0] != 0 {
		t.Errorf("single-extrude profiles = %v, want [0]", got)
	}
}
