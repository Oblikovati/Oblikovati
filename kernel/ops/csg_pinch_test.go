// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// tetTris returns the four CCW-outward triangles of a tetrahedron with apex at p and a small
// base offset so the two test tets meet the shared point only at their apexes.
func tetTris(base math.Point3, dir float64) []tri {
	a := base
	b := math.P3(base.X+dir, base.Y, base.Z-dir)
	c := math.P3(base.X+dir, base.Y+0.6*math.Scalar(dir), base.Z-dir)
	d := math.P3(base.X+0.4*math.Scalar(dir), base.Y+0.2*math.Scalar(dir), base.Z-1.6*math.Scalar(dir))
	mk := func(p, q, r math.Point3) tri {
		t, _ := newTri(p, q, r)
		return t
	}
	if dir > 0 {
		return []tri{mk(a, b, c), mk(a, c, d), mk(a, d, b), mk(b, d, c)}
	}
	return []tri{mk(a, c, b), mk(a, d, c), mk(a, b, d), mk(b, c, d)}
}

// TestCSGCageSplitsPinchedVertex pins #1693: a triangle soup whose two closed shells meet at ONE
// coincident point (the sub-resolution tangency lens a near-tangent JOIN welds down to) must
// assemble into a VALID body — the pinched vertex cut apart into coincident duplicates (χ = 4 for
// two shells), never a χ = 3 one-vertex pinch that ships as a silently inadmissible solid. This is
// the minimized fan-blade-tip-on-rim-wall failure (13 red bridge e2e, MCPBridge#86).
func TestCSGCageSplitsPinchedVertex(t *testing.T) {
	apex := math.P3(0, 0, 0)
	soup := append(tetTris(apex, 1), tetTris(apex, -1)...)
	b := trianglesToBody(soup, "pinch")
	if b == nil {
		t.Fatal("trianglesToBody returned nil")
	}
	r := Validate(b)
	if !r.Valid {
		t.Errorf("point-contact shells assemble invalid: χ=%d issues=%v", r.EulerCharacteristic, r.Issues)
	}
	if r.EulerCharacteristic != 4 {
		t.Errorf("χ = %d, want 4 (two sphere shells after the pinch split)", r.EulerCharacteristic)
	}
	if nv := len(b.Vertices()); nv != 8 {
		t.Errorf("body has %d vertices, want 8 (7 welded + one coincident duplicate at the contact)", nv)
	}
}
