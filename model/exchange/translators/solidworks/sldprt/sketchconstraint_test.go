// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import "testing"

func countKinds(cs []Constraint) map[ConstraintKind]int {
	m := map[ConstraintKind]int{}
	for _, c := range cs {
		m[c.Kind]++
	}
	return m
}

// TestConstraintDecode checks the decoded per-sketch relations against the SolidWorks 2026
// SketchRelationManager: a rectangle carries 2 horizontal + 2 vertical + equal-length + coincident;
// a rounded rectangle carries 2 horizontal + 2 vertical + 3 tangent. The origin's FIXED relation
// belongs to the origin sketch, not these — so per-sketch attribution keeps it out.
func TestConstraintDecode(t *testing.T) {
	cases := []struct {
		file string
		want map[ConstraintKind]int
	}{
		{"box10_fmtb.sldprt", map[ConstraintKind]int{Horizontal: 2, Vertical: 2, EqualLength: 1, Coincident: 1}},
		{"rrect_fmtb.sldprt", map[ConstraintKind]int{Horizontal: 2, Vertical: 2, Tangent: 3}},
	}
	for _, c := range cases {
		d, err := Open(readTestdata(t, c.file))
		if err != nil {
			t.Fatalf("Open %s: %v", c.file, err)
		}
		got := countKinds(d.Sketches()[0].Constraints)
		for k, n := range c.want {
			if got[k] != n {
				t.Errorf("%s: %v = %d, want %d (all: %v)", c.file, k, got[k], n, got)
			}
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: got kinds %v, want %v", c.file, got, c.want)
		}
	}
}

// TestConstraintPerSketch verifies a multi-sketch part attributes constraints to the right sketch:
// the rectangle sketch carries relations, the circle sketch none.
func TestConstraintPerSketch(t *testing.T) {
	d, err := Open(readTestdata(t, "twosketch_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	sk := d.Sketches()
	if len(sk) != 2 {
		t.Fatalf("got %d sketches, want 2", len(sk))
	}
	if len(sk[0].Constraints) == 0 {
		t.Error("rectangle sketch should carry relations")
	}
	if n := len(sk[1].Constraints); n != 0 {
		t.Errorf("circle sketch has %d constraints, want 0", n)
	}
}

func TestConstraintKindIsGeometric(t *testing.T) {
	if !Horizontal.IsGeometric() || !Tangent.IsGeometric() || !Coincident.IsGeometric() {
		t.Error("H/tangent/coincident must be geometric")
	}
	if Distance.IsGeometric() || Radius.IsGeometric() || Fixed.IsGeometric() {
		t.Error("dimensions and fixed are not geometric constraints")
	}
}
