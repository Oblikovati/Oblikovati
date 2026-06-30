// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/model/sketch"
)

// TestRevolveAboutSketchCenterline spins a profile about the sketch's own centerline (Inventor's
// common flow), producing the same washer as revolving about an explicit Y work axis.
func TestRevolveAboutSketchCenterline(t *testing.T) {
	fs := NewPartFeatures(nil)
	sk := offsetSquareSketch(2, 2)                                // square x∈[2,4], y∈[0,2]
	cl := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2)) // vertical centerline = Y axis
	cl.SetCenterline(true)
	pf := NewRevolveFeatures(fs).AddAboutCenterline(sk, 0, nil, ops.NewBody)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("revolve about centerline sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("centerline-revolved body not a valid solid: %+v", r)
	}
	want := stdmath.Pi * (4*4 - 2*2) * 2 // 24π washer
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(got, want) > 0.01 {
		t.Errorf("centerline-revolved washer = %g, want ≈%g (24π)", got, want)
	}
}

// A revolve with neither an axis nor a sketch centerline is sick.
func TestRevolveNoAxisOrCenterlineSick(t *testing.T) {
	fs := NewPartFeatures(nil)
	pf := NewRevolveFeatures(fs).AddAboutCenterline(offsetSquareSketch(2, 2), 0, nil, ops.NewBody)
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("revolve with no axis/centerline = %v, want sick", pf.Health().Status)
	}
}

// offsetSquareSketch returns a sketch with a square in the XY plane spanning
// x∈[x0,x0+side], y∈[0,side] — a profile offset from the Y axis for revolving.
func offsetSquareSketch(x0, side float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	c0 := s.Points().Add(math.P2(x0, 0))
	c1 := s.Points().Add(math.P2(x0+side, 0))
	c2 := s.Points().Add(math.P2(x0+side, side))
	c3 := s.Points().Add(math.P2(x0, side))
	s.Lines().Add(c0, c1)
	s.Lines().Add(c1, c2)
	s.Lines().Add(c2, c3)
	s.Lines().Add(c3, c0)
	return s
}

// yAxis is the Y axis as a datum line for revolving about.
func yAxis() *WorkAxis {
	return &WorkAxis{origin: math.P3(0, 0, 0), dir: math.V3(0, 1, 0).AsUnit()}
}

func TestRevolveFullMakesValidWasher(t *testing.T) {
	// Square x∈[2,4], y∈[0,2] revolved 360° about Y → a washer: inner r=2, outer r=4,
	// height 2 → volume π(4²−2²)·2 = 24π.
	fs := NewPartFeatures(nil)
	pf := NewRevolveFeatures(fs).Add(offsetSquareSketch(2, 2), 0, yAxis(), nil, ops.NewBody)
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("revolve went sick: %+v", pf.Health())
	}
	if len(fs.Result()) != 1 {
		t.Fatalf("revolve result = %d bodies, want 1", len(fs.Result()))
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("revolved washer not a valid solid: %+v solid=%v", r, body.IsSolid())
	}
	want := stdmath.Pi * (4*4 - 2*2) * 2
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(got, want) > 0.01 {
		t.Errorf("washer volume = %g, want ≈%g (24π) within 1%%", got, want)
	}
}

func TestRevolvePartialIsCappedSolid(t *testing.T) {
	// A 90° revolve of the same square → a quarter washer, volume 24π/4 = 6π, capped.
	fs := NewPartFeatures(nil)
	pf := NewRevolveFeatures(fs).Add(offsetSquareSketch(2, 2), 0, yAxis(),
		func() float64 { return stdmath.Pi / 2 }, ops.NewBody)
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("partial revolve went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("partial revolve not a valid solid: %+v", r)
	}
	want := stdmath.Pi * (4*4 - 2*2) * 2 / 4
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(got, want) > 0.02 {
		t.Errorf("quarter-washer volume = %g, want ≈%g (6π)", got, want)
	}
}

func TestRevolveOpenProfileGoesSick(t *testing.T) {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	c0 := s.Points().Add(math.P2(2, 0))
	c1 := s.Points().Add(math.P2(4, 0))
	s.Lines().Add(c0, c1) // a single open segment — no closed region
	fs := NewPartFeatures(nil)
	pf := NewRevolveFeatures(fs).Add(s, 0, yAxis(), nil, ops.NewBody)
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("revolve of an open/absent profile should go sick")
	}
}
