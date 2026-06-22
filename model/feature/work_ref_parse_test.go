// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"
)

// TestParseWorkRefClassifies: work-feature refs pass through verbatim; anything else is wrapped as
// a face ref (and decodes back to the original key).
func TestParseWorkRefClassifies(t *testing.T) {
	if got := ParseWorkRef("origin/plane/xy"); got != WorkRef("origin/plane/xy") {
		t.Errorf("work-feature ref = %q, want it verbatim", got)
	}
	if got := ParseWorkRef("plane/3"); got != WorkRef("plane/3") {
		t.Errorf("user plane ref = %q, want it verbatim", got)
	}
	key := []byte("some-face-key")
	wr := ParseWorkRef(string(key))
	if back, ok := FaceRefKey(wr); !ok || string(back) != string(key) {
		t.Errorf("raw key was not wrapped as a face ref: got %q (decoded %q, ok=%v)", wr, back, ok)
	}
}

// TestPlaneTargetFromRefResolvesPlanes: an origin plane and a user work plane both resolve to a
// *WorkPlane target, and an unknown reference errors.
func TestPlaneTargetFromRefResolvesPlanes(t *testing.T) {
	g := NewWorkGeometry()

	xy, err := g.PlaneTargetFromRef("origin/plane/xy")
	if err != nil {
		t.Fatalf("origin XY: %v", err)
	}
	if n := xy.Plane().Normal().AsVector(); stdmath.Abs(float64(n.Z)) < 0.999 {
		t.Errorf("origin XY normal = %v, want ±Z", n)
	}

	off := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 5 }) // z = 5
	wp, err := g.PlaneTargetFromRef(string(off.Key()))
	if err != nil {
		t.Fatalf("user plane: %v", err)
	}
	if z := float64(wp.Plane().Origin().Z); stdmath.Abs(z-5) > 1e-6 {
		t.Errorf("offset plane origin z = %v, want 5", z)
	}

	if _, err := g.PlaneTargetFromRef("plane/99"); err == nil {
		t.Error("an unknown plane reference should error")
	}
}
