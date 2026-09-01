// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// TestWorkPlaneSurfaceSourceResolvesOriginPlane: the origin XY plane resolves to the z=0 surface
// (#1854).
func TestWorkPlaneSurfaceSourceResolvesOriginPlane(t *testing.T) {
	t.Parallel()
	def := compdef.NewPartComponentDefinition()
	surf := compdef.NewWorkPlaneSurfaceSource(def, feature.OriginXYPlane).Surface()
	if surf == nil {
		t.Fatal("origin XY plane should resolve to a surface")
	}
	// Every point of the XY plane lies at z=0.
	for _, uv := range [][2]float64{{0, 0}, {1, 2}, {-3, 4}} {
		if p := surf.PointAt(uv[0], uv[1]); stdmath.Abs(float64(p.Z)) > 1e-9 {
			t.Errorf("XY-plane point at (%g,%g) = %v, want z=0", uv[0], uv[1], p)
		}
	}
}

// TestWorkPlaneSurfaceSourceLostReferenceIsNil: an unknown work-plane ref yields no surface (#1854).
func TestWorkPlaneSurfaceSourceLostReferenceIsNil(t *testing.T) {
	t.Parallel()
	def := compdef.NewPartComponentDefinition()
	if surf := compdef.NewWorkPlaneSurfaceSource(def, feature.WorkRef("no-such-plane")).Surface(); surf != nil {
		t.Error("an unresolved work plane should yield a nil surface")
	}
}

// TestIntersectionCurveWithWorkPlaneOperand: a radius-5 sphere ∩ the origin XY plane is the z=0
// great circle, proving a work plane works as an intersection operand (#1854 AC1).
func TestIntersectionCurveWithWorkPlaneOperand(t *testing.T) {
	t.Parallel()
	def := compdef.NewPartComponentDefinition()
	sphere, err := geom.NewSphere(math.P3(0, 0, 0), 5)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	s := sketch.NewSketches3D().Add()
	c := s.AddIntersectionCurve3DRef(
		sketch.StaticSurface(sphere),
		compdef.NewWorkPlaneSurfaceSource(def, feature.OriginXYPlane),
		geom.SurfaceGrid{},
	)
	polys := c.Evaluate()
	if len(polys) == 0 {
		t.Fatal("sphere ∩ work plane produced no section curve")
	}
	for _, poly := range polys {
		for _, p := range poly {
			if stdmath.Abs(float64(p.Z)) > 1e-2 {
				t.Errorf("section point %v not on the z=0 plane", p)
			}
			if r := stdmath.Hypot(float64(p.X), float64(p.Y)); stdmath.Abs(r-5) > 1e-2 {
				t.Errorf("section point radius = %g, want 5 (the great circle)", r)
			}
		}
	}
}
