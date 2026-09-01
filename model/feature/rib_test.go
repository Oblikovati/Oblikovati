// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// A rib over a straight open path is a wall of length×thickness×depth.
func TestRibGeneratesWall(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0)) // open path, length 4

	rib := &RibFeature{def: &RibDefinition{
		Sketch: sk, ProfileIndex: 0,
		Thickness: func() float64 { return 1 },
		Depth:     func() float64 { return 2 },
		Operation: ops.NewBody,
	}}
	pf := fs.Add(rib)
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("rib went sick: %+v", pf.Health())
	}
	bodies := fs.Result()
	if len(bodies) != 1 || !bodies[0].IsSolid() {
		t.Fatalf("rib result = %d bodies, want 1 solid", len(bodies))
	}
	if r := ops.Validate(bodies[0]); !r.Valid {
		t.Fatalf("rib body invalid: %+v", r)
	}
	vol := ops.BodyGeometryProperties(bodies[0], ops.DefaultQuality()).Volume
	if stdmath.Abs(vol-8) > 1e-6 { // 4 (length) × 1 (thickness) × 2 (depth)
		t.Errorf("rib volume = %g, want 8", vol)
	}
}

// The RibFeatures collection adds a named, healthy rib (the path the Rib tool drives).
func TestRibFeaturesAddNamesAndBuilds(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))

	pf := NewRibFeatures(fs).Add(sk, 0, func() float64 { return 1 }, func() float64 { return 2 }, ops.NewBody)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("rib went sick: %+v", pf.Health())
	}
	if pf.Name() != "Rib1" {
		t.Errorf("rib name = %q, want Rib1", pf.Name())
	}
	if vol := ops.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume; stdmath.Abs(vol-8) > 1e-6 {
		t.Errorf("rib volume = %g, want 8", vol)
	}
}

// An L-shaped open path ribs into a connected wall (two segments → one solid).
func TestRibLShapedPath(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	corner := sk.Points().Add(math.P2(4, 0))
	sk.Lines().Add(sk.Points().Add(math.P2(0, 0)), corner)
	sk.Lines().Add(corner, sk.Points().Add(math.P2(4, 3)))

	rib := &RibFeature{def: &RibDefinition{
		Sketch: sk, ProfileIndex: 0,
		Thickness: func() float64 { return 1 },
		Depth:     func() float64 { return 2 },
		Operation: ops.NewBody,
	}}
	pf := fs.Add(rib)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("L-rib went sick: %+v", pf.Health())
	}
	bodies := fs.Result()
	if len(bodies) != 1 || !bodies[0].IsSolid() {
		t.Fatalf("L-rib = %d bodies, want 1 solid", len(bodies))
	}
}

// A degenerate definition (no depth) reports sick, not a crash.
func TestRibNeedsDepth(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	rib := &RibFeature{def: &RibDefinition{Sketch: sk, ProfileIndex: 0, Thickness: func() float64 { return 1 }, Operation: ops.NewBody}}
	pf := fs.Add(rib)
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("a rib with no depth should be sick")
	}
}
