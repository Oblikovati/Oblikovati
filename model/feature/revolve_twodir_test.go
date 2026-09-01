// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// Two-directional revolve (M08 PBI-093, #313): the sweep spans [-angle2,
// +angle1] about the axis; a combined full turn collapses to the closed solid.

// TestRevolveTwoDirectionalVolume: 90° forward + 90° back = the same material
// as a single 180° revolve of the washer profile.
func TestRevolveTwoDirectionalVolume(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	sk := offsetSquareSketch(2, 2)
	axis := yWorkAxis()
	pf := NewRevolveFeatures(fs).AddTwoDirectional(sk, 0, axis,
		angleConst(stdmath.Pi/2), angleConst(stdmath.Pi/2), ops.NewBody)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("two-directional revolve sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("two-directional revolve not a valid solid: %+v", r)
	}
	want := stdmath.Pi * (4*4 - 2*2) * 2 / 2 // half the 24π washer
	if got := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(got, want) > 0.01 {
		t.Errorf("two-directional half washer = %g, want ≈%g (12π)", got, want)
	}
}

// TestRevolveTwoDirectionalSpansAcrossPlane: the solid genuinely straddles the
// sketch plane (material on both angular sides), unlike the one-way revolve.
func TestRevolveTwoDirectionalSpansAcrossPlane(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	sk := offsetSquareSketch(2, 2)
	pf := NewRevolveFeatures(fs).AddTwoDirectional(sk, 0, yWorkAxis(),
		angleConst(stdmath.Pi/4), angleConst(stdmath.Pi/4), ops.NewBody)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("revolve sick: %+v", pf.Health())
	}
	box := fs.Result()[0].RangeBox()
	// The profile starts in the +X half-plane (z=0); ±45° swings reach equal
	// |z| on both sides.
	if float64(box.Max.Z) < 1 || float64(box.Min.Z) > -1 {
		t.Errorf("two-directional revolve box z = [%v, %v], want material on both sides", box.Min.Z, box.Max.Z)
	}
	if stdmath.Abs(float64(box.Max.Z)+float64(box.Min.Z)) > 0.1 {
		t.Errorf("two-directional ±45° revolve should be z-symmetric, box z = [%v, %v]", box.Min.Z, box.Max.Z)
	}
}

// TestRevolveTwoDirectionalFullTurnCollapses: angle+angle2 ≥ 2π is the full
// revolution (closed, no caps), identical volume to the plain full revolve.
func TestRevolveTwoDirectionalFullTurnCollapses(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	sk := offsetSquareSketch(2, 2)
	pf := NewRevolveFeatures(fs).AddTwoDirectional(sk, 0, yWorkAxis(),
		angleConst(1.5*stdmath.Pi), angleConst(0.5*stdmath.Pi), ops.NewBody)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("revolve sick: %+v", pf.Health())
	}
	want := stdmath.Pi * (4*4 - 2*2) * 2 // the full 24π washer
	if got := query.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume; relErr(got, want) > 0.01 {
		t.Errorf("collapsed full revolve = %g, want ≈%g (24π)", got, want)
	}
}

func angleConst(v float64) func() float64 { return func() float64 { return v } }

// yWorkAxis is a transient +Y axis through the origin.
func yWorkAxis() *WorkAxis {
	dir, _ := math.UnitVector3FromVector(math.V3(0, 1, 0))
	return &WorkAxis{origin: math.P3(0, 0, 0), dir: dir}
}
