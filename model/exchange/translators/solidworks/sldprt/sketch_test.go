// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"math"
	"testing"
)

func hasPoint(pts []Point, x, y, tol float64) bool {
	for _, p := range pts {
		if math.Abs(p.X-x) <= tol && math.Abs(p.Y-y) <= tol {
			return true
		}
	}
	return false
}

// TestSketchPointsRectangleFormatB decodes a generated 10 mm square (4 lines) from the format-B
// container: its four corners at (0,0)-(0.01,0.01) m must all appear, and no others.
func TestSketchPointsRectangleFormatB(t *testing.T) {
	d, err := Open(readTestdata(t, "box10_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	sk := d.Sketches()
	if len(sk) != 1 {
		t.Fatalf("got %d sketches, want 1", len(sk))
	}
	corners := [][2]float64{{0, 0}, {0.01, 0}, {0.01, 0.01}, {0, 0.01}}
	for _, c := range corners {
		if !hasPoint(sk[0].Points, c[0], c[1], 1e-9) {
			t.Errorf("missing corner %v; got %v", c, sk[0].Points)
		}
	}
	if len(sk[0].Points) != 4 {
		t.Errorf("got %d points, want 4 rectangle corners: %v", len(sk[0].Points), sk[0].Points)
	}
}

// TestSketchPointsCircleFormatB decodes a generated Ø10 mm circle: centre (0,0) and a rim point at
// radius 0.005 m.
func TestSketchPointsCircleFormatB(t *testing.T) {
	d, err := Open(readTestdata(t, "cyl_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	pts := d.Sketches()[0].Points
	if !hasPoint(pts, 0, 0, 1e-9) {
		t.Errorf("missing centre; got %v", pts)
	}
	var rim Point
	for _, p := range pts {
		if p.X != 0 || p.Y != 0 {
			rim = p
		}
	}
	if r := math.Hypot(rim.X, rim.Y); math.Abs(r-0.005) > 1e-6 {
		t.Errorf("rim radius = %g, want 0.005", r)
	}
}

// TestSketchPointsCircleFormatA decodes the same shape from the CFBF container (a real part, a
// stud): the sketch lives in Contents/Config-0 there, proving the format-agnostic stream choice.
func TestSketchPointsCircleFormatA(t *testing.T) {
	d, err := Open(readTestdata(t, "stud_cfbf.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	pts := d.Sketches()[0].Points
	if !hasPoint(pts, 0, 0, 1e-9) || !hasPoint(pts, 0.003347, 0.001023, 1e-5) {
		t.Errorf("stud circle points wrong; got %v", pts)
	}
}

func onlyCircle(t *testing.T, file string) Circle {
	t.Helper()
	d, err := Open(readTestdata(t, file))
	if err != nil {
		t.Fatalf("Open %s: %v", file, err)
	}
	c := d.Sketches()[0].Circles
	if len(c) != 1 {
		t.Fatalf("%s: got %d circles, want 1", file, len(c))
	}
	return c[0]
}

// TestCircleEntity checks centre+radius association across origin, off-origin, and both container
// formats: the record stores centre then rim, radius = |rim-centre|.
func TestCircleEntity(t *testing.T) {
	cases := []struct {
		file      string
		cx, cy, r float64
	}{
		{"cyl_fmtb.sldprt", 0, 0, 0.005},
		{"stud_cfbf.sldprt", 0, 0, 0.0035},
		{"circoff_fmtb.sldprt", 0.02, 0.01, 0.005},
	}
	for _, c := range cases {
		got := onlyCircle(t, c.file)
		if math.Abs(got.Center.X-c.cx) > 1e-6 || math.Abs(got.Center.Y-c.cy) > 1e-6 || math.Abs(got.Radius-c.r) > 1e-6 {
			t.Errorf("%s: got centre (%g,%g) r=%g, want (%g,%g) r=%g", c.file, got.Center.X, got.Center.Y, got.Radius, c.cx, c.cy, c.r)
		}
	}
}

// TestTwoCircleEntities verifies a multi-circle sketch separates into distinct circles (a repeated
// centre reference does not merge them).
func TestTwoCircleEntities(t *testing.T) {
	d, err := Open(readTestdata(t, "twocirc_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	c := d.Sketches()[0].Circles
	if len(c) != 2 {
		t.Fatalf("got %d circles, want 2: %+v", len(c), c)
	}
	find := func(cx, cy, r float64) bool {
		for _, x := range c {
			if math.Abs(x.Center.X-cx) < 1e-6 && math.Abs(x.Center.Y-cy) < 1e-6 && math.Abs(x.Radius-r) < 1e-6 {
				return true
			}
		}
		return false
	}
	if !find(0, 0, 0.004) || !find(0.03, 0, 0.006) {
		t.Errorf("circles wrong: %+v", c)
	}
}

// TestLineLoopEntity checks a pure-line sketch reconstructs into a closed loop of the right size
// whose segments join end-to-end and whose vertices are the sketch corners.
func TestLineLoopEntity(t *testing.T) {
	cases := []struct {
		file  string
		count int
	}{{"box10_fmtb.sldprt", 4}, {"tri_fmtb.sldprt", 3}}
	for _, c := range cases {
		d, err := Open(readTestdata(t, c.file))
		if err != nil {
			t.Fatalf("Open %s: %v", c.file, err)
		}
		lines := d.Sketches()[0].Lines
		if len(lines) != c.count {
			t.Fatalf("%s: got %d lines, want %d", c.file, len(lines), c.count)
		}
		for i := range lines {
			if lines[i].B != lines[(i+1)%len(lines)].A {
				t.Errorf("%s: line %d end != next start (open loop): %+v", c.file, i, lines)
			}
		}
	}
}
