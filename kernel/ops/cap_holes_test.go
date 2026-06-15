// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// boxWithThroughHole returns a 4×4×2 block (vol 32) with a 2×2 column through it in z (a
// through-hole of volume 8, leaving 24), built by extruding a rectangle-with-rectangular-hole
// profile so the faces are clean quads with real inner loops (not a triangle cage).
func boxWithThroughHole(t *testing.T) *topo.Body {
	t.Helper()
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	addSketchRect(sk, 0, 0, 4, 4) // outer boundary
	addSketchRect(sk, 1, 1, 3, 3) // the hole
	fs := feature.NewPartFeatures(nil, nil)
	feature.NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 2 })
	fs.Recompute()
	holey := fs.Result()[0]
	if v := csgVolume(holey); stdmath.Abs(v-24) > 1e-6 {
		t.Fatalf("holey block volume = %g, want 24 (32 − 8 hole)", v)
	}
	return holey
}

// addSketchRect adds a closed axis-aligned rectangle [x0,x1]×[y0,y1] to a sketch.
func addSketchRect(sk *sketch.Sketch, x0, y0, x1, y1 float64) {
	c0 := sk.Points().Add(math.P2(x0, y0))
	c1 := sk.Points().Add(math.P2(x1, y0))
	c2 := sk.Points().Add(math.P2(x1, y1))
	c3 := sk.Points().Add(math.P2(x0, y1))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
}

// TestCapHolesByDiameterFillsThroughHole detects and caps the through-hole, restoring the solid
// 4×4×2 block (vol 32) — the #721 acceptance.
func TestCapHolesByDiameterFillsThroughHole(t *testing.T) {
	capped, err := ops.CapHolesByDiameter(boxWithThroughHole(t), 3)
	if err != nil {
		t.Fatalf("CapHolesByDiameter: %v", err)
	}
	if r := ops.Validate(capped); !r.Valid || !capped.IsSolid() {
		t.Fatalf("capped body not a valid solid: %+v", r.Issues)
	}
	if v := csgVolume(capped); stdmath.Abs(v-32) > 1e-6 {
		t.Errorf("capped volume = %g, want 32 (hole filled flush)", v)
	}
}

// TestCapHolesExplicitSelection caps the same hole by selecting its four wall faces directly.
func TestCapHolesExplicitSelection(t *testing.T) {
	holey := boxWithThroughHole(t)
	var walls [][]byte
	for _, f := range holey.Faces() {
		if n := f.Geometry().NormalAt(0, 0); stdmath.Abs(n.Z) < 0.1 && interiorXY(f) {
			walls = append(walls, f.ReferenceKey())
		}
	}
	if len(walls) != 4 {
		t.Fatalf("selected %d wall faces, want 4", len(walls))
	}
	capped, err := ops.CapHoles(holey, walls)
	if err != nil {
		t.Fatalf("CapHoles: %v", err)
	}
	if v := csgVolume(capped); stdmath.Abs(v-32) > 1e-6 || !capped.IsSolid() {
		t.Errorf("capped volume = %g solid=%v, want 32 solid", v, capped.IsSolid())
	}
}

// TestCapHolesByDiameterIgnoresWideOpening leaves the hole when its opening exceeds the max
// diameter (returns the body unchanged).
func TestCapHolesByDiameterIgnoresWideOpening(t *testing.T) {
	holey := boxWithThroughHole(t)
	out, err := ops.CapHolesByDiameter(holey, 1) // the 2×2 opening is wider than 1
	if err != nil {
		t.Fatalf("CapHolesByDiameter: %v", err)
	}
	if v := csgVolume(out); stdmath.Abs(v-24) > 1e-6 {
		t.Errorf("volume = %g, want 24 (hole left intact)", v)
	}
}

// TestCapHolesEmptySelection rejects an empty wall set.
func TestCapHolesEmptySelection(t *testing.T) {
	if _, err := ops.CapHoles(csgBox(math.P3(0, 0, 0), 1, 1, 1), nil); err == nil {
		t.Error("expected an error for an empty wall selection")
	}
}

// interiorXY reports whether a face's vertices all lie within the bored column footprint
// (x,y ∈ (0.5, 3.5)) — i.e. it is a hole wall, not an outer side.
func interiorXY(f *topo.Face) bool {
	for _, v := range f.Vertices() {
		p := v.Point()
		if p.X < 0.5 || p.X > 3.5 || p.Y < 0.5 || p.Y > 3.5 {
			return false
		}
	}
	return true
}
