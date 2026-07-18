// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

// TestDecodeSweep checks the sweep decodes its circular profile (Ø0.5 at the origin) and its
// L-path polyline (0,0)→(0,5)→(5,5), ordered from the origin end.
func TestDecodeSweep(t *testing.T) {
	d := openDoc(t, "29_sweep.ipt")
	seg, _ := d.Segment("PmDCSegment")
	sw, ok := DecodeSweep(seg)
	if !ok {
		t.Fatal("no sweep decoded")
	}
	if absf(sw.Profile.Radius-0.5) > 1e-9 {
		t.Errorf("profile radius = %.4f, want 0.5", sw.Profile.Radius)
	}
	want := []Point2D{{0, 0}, {0, 5}, {5, 5}}
	if len(sw.Path) != len(want) {
		t.Fatalf("path has %d points, want %d: %v", len(sw.Path), len(want), sw.Path)
	}
	for i, p := range want {
		if !samePt(sw.Path[i], p) {
			t.Errorf("path[%d] = %v, want %v", i, sw.Path[i], p)
		}
	}
}

// TestDecodeArcSweep checks the arc-path sweep: the arc (centre (3,0), radius 3, endpoints
// (0,0) and (3,3)) decodes and tessellates into a path from the origin end to (3,3), each
// point on the circle of radius 3 about (3,0).
func TestDecodeArcSweep(t *testing.T) {
	d := openDoc(t, "30_sweep_arc.ipt")
	seg, _ := d.Segment("PmDCSegment")

	// the arc is decoded as a first-class sketch arc
	var arc Arc
	found := false
	for _, s := range DecodeSketches(seg) {
		for _, a := range s.Arcs {
			arc, found = a, true
		}
	}
	if !found {
		t.Fatal("no arc decoded")
	}
	if absf(arc.Center.X-3) > 1e-9 || absf(arc.Center.Y) > 1e-9 || absf(arc.Radius-3) > 1e-9 {
		t.Errorf("arc centre=%v r=%.3f, want (3,0) r=3", arc.Center, arc.Radius)
	}

	sw, ok := DecodeSweep(seg)
	if !ok || len(sw.Path) < 3 {
		t.Fatalf("arc sweep path not decoded: ok=%v pts=%d", ok, len(sw.Path))
	}
	if !samePt(sw.Path[0], Point2D{0, 0}) {
		t.Errorf("path start = %v, want (0,0)", sw.Path[0])
	}
	if last := sw.Path[len(sw.Path)-1]; !samePt(last, Point2D{3, 3}) {
		t.Errorf("path end = %v, want (3,3)", last)
	}
	for _, p := range sw.Path {
		if r := absf((p.X-3)*(p.X-3) + p.Y*p.Y - 9); r > 1e-6 {
			t.Errorf("path point %v not on radius-3 circle about (3,0)", p)
			break
		}
	}
}

// TestHasSweepAbsent confirms non-sweep parts report no sweep.
func TestHasSweepAbsent(t *testing.T) {
	for _, file := range []string{"10_box.ipt", "28_loft.ipt", "15_cylinder.ipt"} {
		d := openDoc(t, file)
		seg, _ := d.Segment("PmDCSegment")
		if _, ok := DecodeSweep(seg); ok {
			t.Errorf("%s: decoded a sweep where there is none", file)
		}
	}
}
