// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

// TestDecodeRectPattern checks the rectangular-pattern part decodes to 3 occurrences at
// 1.5 cm spacing (the two model params authored after the box + pocket distances).
func TestDecodeRectPattern(t *testing.T) {
	d := openDoc(t, "21_pocket_rect.ipt")
	rp, ok := DecodeRectPattern(d)
	if !ok {
		t.Fatal("no rectangular pattern decoded")
	}
	if rp.Count != 3 {
		t.Errorf("count = %d, want 3", rp.Count)
	}
	if absf(rp.Spacing-1.5) > 1e-9 {
		t.Errorf("spacing = %.4f cm, want 1.5", rp.Spacing)
	}
}

// TestDecodeCircPattern checks the circular-pattern part decodes to 6 occurrences over a
// full 2π sweep (the two model params after the disk + pocket distances).
func TestDecodeCircPattern(t *testing.T) {
	d := openDoc(t, "22_pocket_circ.ipt")
	cp, ok := DecodeCircPattern(d)
	if !ok {
		t.Fatal("no circular pattern decoded")
	}
	if cp.Count != 6 {
		t.Errorf("count = %d, want 6", cp.Count)
	}
	if absf(cp.Angle-2*3.141592653589793) > 1e-6 {
		t.Errorf("angle = %.6f rad, want 2π", cp.Angle)
	}
}

// TestDecodePatternsDontCrossMatch confirms the two pattern kinds are distinguished by the
// node name: the rectangular part is not read as circular, and vice versa.
func TestDecodePatternsDontCrossMatch(t *testing.T) {
	rect := openDoc(t, "21_pocket_rect.ipt")
	if _, ok := DecodeCircPattern(rect); ok {
		t.Error("rectangular part decoded as circular")
	}
	circ := openDoc(t, "22_pocket_circ.ipt")
	if _, ok := DecodeRectPattern(circ); ok {
		t.Error("circular part decoded as rectangular")
	}
}

// TestDecodeRectPatternAbsent confirms non-pattern parts report no pattern.
func TestDecodeRectPatternAbsent(t *testing.T) {
	for _, file := range []string{"10_box.ipt", "17_box_cut.ipt", "19_box_hole.ipt"} {
		d := openDoc(t, file)
		if _, ok := DecodeRectPattern(d); ok {
			t.Errorf("%s: decoded a pattern where there is none", file)
		}
		if _, ok := DecodeCircPattern(d); ok {
			t.Errorf("%s: decoded a circular pattern where there is none", file)
		}
	}
}
