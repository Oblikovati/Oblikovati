// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

func decodeSketchesOf(t *testing.T, file string) []Sketch {
	t.Helper()
	d := openDoc(t, file)
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		t.Fatalf("%s: no PmDCSegment", file)
	}
	return DecodeSketches(seg)
}

// TestDecodeEllipse locks the ellipse discriminator. ke_ellipse.ipt is a two-line base sketch
// plus one ellipse authored at centre (10,5) with the major axis along +X, majorR 3, minorR 1.5.
// The node shares a circle's shape (centre by reference, +16 not a ref), so before the sentinel
// gate it decoded as a phantom circle of radius 1 (it read the major-axis X as the radius). The
// three-signal gate (sentinel 0x01170000 at +72 AND normal-range majorR@+52 AND minorR@+60)
// must now recover it exactly as an ellipse, with the two base lines intact and NO circle.
func TestDecodeEllipse(t *testing.T) {
	sks := decodeSketchesOf(t, "ke_ellipse.ipt")
	if len(sks) != 1 {
		t.Fatalf("got %d sketches, want 1", len(sks))
	}
	s := sks[0]
	if len(s.Circles) != 0 {
		t.Errorf("got %d circles, want 0 (the ellipse must not decode as a phantom circle)", len(s.Circles))
	}
	if len(s.Lines) != 2 {
		t.Errorf("got %d lines, want 2 (base sketch)", len(s.Lines))
	}
	if len(s.Ellipses) != 1 {
		t.Fatalf("got %d ellipses, want 1: %+v", len(s.Ellipses), s.Ellipses)
	}
	e := s.Ellipses[0]
	if absf(e.Center.X-10) > 1e-9 || absf(e.Center.Y-5) > 1e-9 {
		t.Errorf("centre = (%.4g,%.4g), want (10,5)", e.Center.X, e.Center.Y)
	}
	if absf(e.MajorAxis.X-1) > 1e-9 || absf(e.MajorAxis.Y) > 1e-9 {
		t.Errorf("major axis = (%.4g,%.4g), want (1,0)", e.MajorAxis.X, e.MajorAxis.Y)
	}
	if absf(e.MajorR-3) > 1e-9 || absf(e.MinorR-1.5) > 1e-9 {
		t.Errorf("radii = (%.4g,%.4g), want majorR=3 minorR=1.5", e.MajorR, e.MinorR)
	}
}

// TestDecodeEllipseNoFalsePositiveOnCircles guards the gate against misfiring: parts full of
// real circles must decode ZERO ellipses. A circle stores denormal garbage (~1e-308) where an
// ellipse keeps its radii and a node reference where an ellipse keeps its sentinel, so the gate
// can never promote a circle to an ellipse. (Cross-checked at scale: zero false positives across
// the 175-part ReelToReel library — see the batch report.)
func TestDecodeEllipseNoFalsePositiveOnCircles(t *testing.T) {
	for _, f := range []string{"15_cylinder.ipt", "22_pocket_circ.ipt", "k_rad.ipt", "k_dia.ipt", "ke_base.ipt"} {
		for _, s := range decodeSketchesOf(t, f) {
			if len(s.Ellipses) != 0 {
				t.Errorf("%s: got %d ellipses, want 0 (a circle must not read as an ellipse)", f, len(s.Ellipses))
			}
		}
	}
}
