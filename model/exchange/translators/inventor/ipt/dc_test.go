// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

func TestDecodeParametersMatchesGroundTruth(t *testing.T) {
	cases := []struct {
		file string
		want []Parameter
	}{
		{"blank_a.ipt", nil},
		{"param_L10.ipt", []Parameter{{"L", 1.0}}}, // 10 mm = 1.0 cm
		{"param_L20.ipt", []Parameter{{"L", 2.0}}}, // 20 mm = 2.0 cm
		{"sketch_line.ipt", nil},
		{"10_box.ipt", nil},
	}
	for _, tc := range cases {
		d := openDoc(t, tc.file)
		seg, ok := d.Segment("PmDCSegment")
		if !ok {
			t.Fatalf("%s: no PmDCSegment", tc.file)
		}
		got := DecodeParameters(seg)
		if len(got) != len(tc.want) {
			t.Errorf("%s: got %v, want %v", tc.file, got, tc.want)
			continue
		}
		for i := range got {
			if got[i].Name != tc.want[i].Name || absf(got[i].Value-tc.want[i].Value) > 1e-9 {
				t.Errorf("%s: param[%d] = %+v, want %+v", tc.file, i, got[i], tc.want[i])
			}
		}
	}
}
