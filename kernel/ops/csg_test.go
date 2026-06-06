// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/subd"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// csgBox builds a validated solid box of the given size with its near corner at p.
func csgBox(p math.Point3, sx, sy, sz float64) *topo.Body {
	m := subd.Box(sx, sy, sz)
	for i := range m.Verts {
		m.Verts[i] = m.Verts[i].TranslateBy(p.AsVector())
	}
	return subd.ToBody(m, "box")
}

func csgVolume(b *topo.Body) float64 {
	return ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
}

// Two 2×2×2 boxes overlapping in a 1×2×2 = 4 slab: A=[0,2]³ (8), B=[1,3]×[0,2]² (8),
// overlap [1,2]×[0,2]² = 4.
func csgOverlap(t *testing.T) (a, b *topo.Body) {
	t.Helper()
	return csgBox(math.P3(0, 0, 0), 2, 2, 2), csgBox(math.P3(1, 0, 0), 2, 2, 2)
}

func TestBooleanJoinIntersecting(t *testing.T) {
	a, b := csgOverlap(t)
	res, err := ops.Boolean(ops.Join, a, b)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("union not a valid solid: %+v", r)
	}
	if got := csgVolume(res); stdmath.Abs(got-12) > 1e-6 { // 8 + 8 − 4
		t.Errorf("union volume = %g, want 12", got)
	}
}

func TestBooleanCutIntersecting(t *testing.T) {
	a, b := csgOverlap(t)
	res, err := ops.Boolean(ops.Cut, a, b)
	if err != nil {
		t.Fatalf("cut: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("difference not a valid solid: %+v", r)
	}
	if got := csgVolume(res); stdmath.Abs(got-4) > 1e-6 { // 8 − 4
		t.Errorf("difference volume = %g, want 4", got)
	}
}

func TestBooleanIntersectIntersecting(t *testing.T) {
	a, b := csgOverlap(t)
	res, err := ops.Boolean(ops.Intersect, a, b)
	if err != nil {
		t.Fatalf("intersect: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("intersection not a valid solid: %+v", r)
	}
	if got := csgVolume(res); stdmath.Abs(got-4) > 1e-6 { // the overlap slab
		t.Errorf("intersection volume = %g, want 4", got)
	}
}

func TestBooleanCutThroughHole(t *testing.T) {
	// A bar [0,6]×[0,2]² (24) minus a tool [2,4]×[−1,3]×[0.5,1.5] passing all the way
	// through it: removed = tool ∩ bar = [2,4]×[0,2]×[0.5,1.5] = 2·2·1 = 4 ⇒ 20 left.
	bar := csgBox(math.P3(0, 0, 0), 6, 2, 2)
	tool := csgBox(math.P3(2, -1, 0.5), 2, 4, 1)
	res, err := ops.Boolean(ops.Cut, bar, tool)
	if err != nil {
		t.Fatalf("cut-through: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("through-cut not a valid solid: %+v", r)
	}
	if got := csgVolume(res); stdmath.Abs(got-20) > 1e-6 {
		t.Errorf("through-cut volume = %g, want 20", got)
	}
}
