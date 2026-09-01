// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
)

// TestRevolveSurfacePartialOpenSheet: a 90° Surface-operation revolve of the square (r 2..4, h 2)
// builds an OPEN surface of revolution — no start/end profile caps — via buildRevolveSheet →
// sweptShell. Its area is the analytic θ·h·(r1+r2)+θ·(r2²−r1²) = 12π (the OCCT-grounded
// surface-of-revolution formula), within the tessellation tolerance (#1858).
func TestRevolveSurfacePartialOpenSheet(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	pf := NewRevolveFeatures(fs).Add(offsetSquareSketch(2, 2), 0, yAxis(),
		func() float64 { return stdmath.Pi / 2 }, ops.Surface)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("surface revolve went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if body.IsSolid() {
		t.Error("partial surface-operation revolve should be an OPEN sheet, got a solid")
	}
	want := stdmath.Pi/2*2*(2+4) + stdmath.Pi/2*(4*4-2*2) // 12π
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Area; relErr(got, want) > 0.03 {
		t.Errorf("open surface-of-revolution area = %g, want ≈%g (12π) within 3%%", got, want)
	}
}

// TestRevolveSurfaceFullClosedSheet: a full 360° Surface revolve of the square boundary builds the
// closed surface of revolution (a watertight sheet), exercising sweptShell's closed-shell outward
// orientation flip. Area = 2π·h·(r1+r2)+2π·(r2²−r1²) = 48π (#1858).
func TestRevolveSurfaceFullClosedSheet(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	pf := NewRevolveFeatures(fs).Add(offsetSquareSketch(2, 2), 0, yAxis(), nil, ops.Surface)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("full surface revolve went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	want := 2*stdmath.Pi*2*(2+4) + 2*stdmath.Pi*(4*4-2*2) // 48π
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Area; relErr(got, want) > 0.03 {
		t.Errorf("closed surface-of-revolution area = %g, want ≈%g (48π) within 3%%", got, want)
	}
}
