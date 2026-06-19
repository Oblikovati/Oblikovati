// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	gmath "oblikovati.org/math"
)

// TestSpline3DHandleLifecycle covers the 3D spline tangency handles: activating a handle on a
// fit point exposes a natural tangent and weight; SetTangent re-aims it (and rejects a zero
// direction); a handled fit spline samples through the Hermite path; deactivation is idempotent.
func TestSpline3DHandleLifecycle(t *testing.T) {
	s := NewSketches3D().Add()
	sp := s.AddSpline3D([]gmath.Point3{
		{X: 0, Y: 0, Z: 0}, {X: 2, Y: 0, Z: 0}, {X: 4, Y: 1, Z: 0}, {X: 6, Y: 0, Z: 0},
	}, false, true)

	h, err := s.ActivateSplineHandle3D(sp, 1)
	if err != nil {
		t.Fatalf("ActivateSplineHandle3D: %v", err)
	}
	if _, ok := h.TangentDirection(); !ok {
		t.Error("an active handle should report a tangent direction")
	}
	if h.Weight() <= 0 {
		t.Errorf("handle weight = %g, want > 0", h.Weight())
	}
	if len(sp.Handles()) != 1 {
		t.Fatalf("Handles() = %d, want 1", len(sp.Handles()))
	}

	if err := h.SetTangent(gmath.V3(1, 0, 0), 2); err != nil {
		t.Errorf("SetTangent: %v", err)
	}
	if dir, ok := h.TangentDirection(); !ok || float64(dir.X) <= 0 {
		t.Errorf("after SetTangent, direction = %v ok=%v, want +X-leaning", dir, ok)
	}
	if err := h.SetTangent(gmath.V3(0, 0, 0), 1); err == nil {
		t.Error("SetTangent with a zero direction should error")
	}

	// A handled fit spline samples through the Hermite-chain path.
	if len(sp.Sample()) < 2 {
		t.Error("handled spline produced too few sample points")
	}

	if !s.DeactivateSplineHandle3D(sp, 1) {
		t.Error("DeactivateSplineHandle3D should report it removed the handle")
	}
	if s.DeactivateSplineHandle3D(sp, 1) {
		t.Error("deactivating an already-removed handle should report false")
	}
	if len(sp.Handles()) != 0 {
		t.Errorf("Handles() = %d after deactivate, want 0", len(sp.Handles()))
	}
}
