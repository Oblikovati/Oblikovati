// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

// TestDecodeHole checks the two drilled-hole corpus parts decode to the right bore: both a
// Ø1 cm bore, one Through All (depth reaches the 2 cm slab) and one blind at depth 1 cm.
func TestDecodeHole(t *testing.T) {
	cases := []struct {
		file       string
		throughAll bool
		depth      float64
	}{
		{"19_box_hole.ipt", true, 0}, // depth ignored when through
		{"20_box_hole_blind.ipt", false, 1},
	}
	for _, tc := range cases {
		d := openDoc(t, tc.file)
		seg, ok := d.Segment("PmDCSegment")
		if !ok {
			t.Fatalf("%s: no PmDCSegment", tc.file)
		}
		h, ok := DecodeHole(seg)
		if !ok {
			t.Fatalf("%s: no hole decoded", tc.file)
		}
		if absf(h.Diameter-1) > 1e-9 {
			t.Errorf("%s: diameter = %.4f cm, want 1", tc.file, h.Diameter)
		}
		if h.ThroughAll != tc.throughAll {
			t.Errorf("%s: throughAll = %v, want %v", tc.file, h.ThroughAll, tc.throughAll)
		}
		if !tc.throughAll && absf(h.Depth-tc.depth) > 1e-9 {
			t.Errorf("%s: depth = %.4f cm, want %.4f", tc.file, h.Depth, tc.depth)
		}
	}
}

// TestDecodeRichHole checks the counterbore and countersink parts decode their type (a
// kind-3 enum node) and counter dimensions: Ø0.6 bore, Ø1.4 counter, 0.5 cm counterbore
// depth, 90° countersink angle.
func TestDecodeRichHole(t *testing.T) {
	cbore := mustHole(t, "25_hole_counterbore.ipt")
	if cbore.Type != CounterboreHole {
		t.Errorf("counterbore type = %d, want %d", cbore.Type, CounterboreHole)
	}
	if absf(cbore.Diameter-0.6) > 1e-9 || absf(cbore.CounterDiameter-1.4) > 1e-9 || absf(cbore.CounterDepth-0.5) > 1e-9 {
		t.Errorf("counterbore = Ø%.3f cbore Ø%.3f × %.3f, want Ø0.6 / Ø1.4 × 0.5", cbore.Diameter, cbore.CounterDiameter, cbore.CounterDepth)
	}
	csink := mustHole(t, "26_hole_countersink.ipt")
	if csink.Type != CountersinkHole {
		t.Errorf("countersink type = %d, want %d", csink.Type, CountersinkHole)
	}
	if absf(csink.Diameter-0.6) > 1e-9 || absf(csink.CounterDiameter-1.4) > 1e-9 || absf(csink.CounterAngle-mathPi/2) > 1e-9 {
		t.Errorf("countersink = Ø%.3f sink Ø%.3f @ %.4f rad, want Ø0.6 / Ø1.4 @ π/2", csink.Diameter, csink.CounterDiameter, csink.CounterAngle)
	}
	// a plain drilled hole reports the drilled type
	if h := mustHole(t, "19_box_hole.ipt"); h.Type != DrilledHole {
		t.Errorf("drilled hole type = %d, want %d", h.Type, DrilledHole)
	}
}

// TestDecodeTappedHole checks a tapped M6x1 hole decodes its tap flag + designation (from
// the thread-info Len32Text16 records) and its tap-drill bore diameter, while plain,
// counterbore, and countersink holes are NOT flagged as tapped.
func TestDecodeTappedHole(t *testing.T) {
	tapped := mustHole(t, "27_hole_tapped.ipt")
	if !tapped.Tapped {
		t.Error("tapped hole not detected as tapped")
	}
	if tapped.Designation != "M6x1" {
		t.Errorf("designation = %q, want M6x1", tapped.Designation)
	}
	if absf(tapped.Diameter-0.4917) > 1e-3 {
		t.Errorf("tap-drill diameter = %.4f cm, want ~0.4917", tapped.Diameter)
	}
	for _, file := range []string{"19_box_hole.ipt", "25_hole_counterbore.ipt", "26_hole_countersink.ipt"} {
		if h := mustHole(t, file); h.Tapped {
			t.Errorf("%s: falsely flagged as tapped (designation %q)", file, h.Designation)
		}
	}
}

func mustHole(t *testing.T, file string) Hole {
	t.Helper()
	d := openDoc(t, file)
	seg, _ := d.Segment("PmDCSegment")
	h, ok := DecodeHole(seg)
	if !ok {
		t.Fatalf("%s: no hole decoded", file)
	}
	return h
}

// TestDecodeHoleAbsentOnPlainParts confirms non-hole parts report no hole.
func TestDecodeHoleAbsentOnPlainParts(t *testing.T) {
	for _, file := range []string{"10_box.ipt", "15_cylinder.ipt", "17_box_cut.ipt"} {
		d := openDoc(t, file)
		seg, _ := d.Segment("PmDCSegment")
		if _, ok := DecodeHole(seg); ok {
			t.Errorf("%s: decoded a hole where there is none", file)
		}
	}
}
