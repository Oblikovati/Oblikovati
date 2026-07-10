// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	gmath "oblikovati.org/math"
)

// TestOnFace3DPinsPointToSurface: a point off a radius-5 sphere, constrained on-face, loses one DOF
// and solves onto the sphere (#1839).
func TestOnFace3DPinsPointToSurface(t *testing.T) {
	s := NewSketches3D().Add()
	p := s.AddPoint3D(gmath.P3(10, 0, 0)) // 10 from the origin; the sphere has radius 5
	sphere, err := geom.NewSphere(gmath.P3(0, 0, 0), 5)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	before := s.DegreesOfFreedom()
	s.GeometricConstraints3D().Add(NewOnFace3D(p, StaticSurface(sphere), ""))
	if after := s.DegreesOfFreedom(); after != before-1 {
		t.Errorf("onFace removed %d DOF, want 1", before-after)
	}

	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: residual=%v", r.Residual)
	}
	if d := float64(p.Position().DistanceTo(gmath.P3(0, 0, 0))); stdmath.Abs(d-5) > 1e-6 {
		t.Errorf("solved point distance from centre = %v, want 5 (on the sphere)", d)
	}
}

// TestOnFace3DUnboundIsInactive: a constraint with no live surface (restored, not yet rebound)
// contributes no equation and keeps its face key for rebind (#1839).
func TestOnFace3DUnboundIsInactive(t *testing.T) {
	s := NewSketches3D().Add()
	p := s.AddPoint3D(gmath.P3(1, 2, 3))
	before := s.DegreesOfFreedom()
	c := NewOnFace3D(p, nil, "face-key")
	s.GeometricConstraints3D().Add(c)

	if after := s.DegreesOfFreedom(); after != before {
		t.Errorf("an unbound onFace removed %d DOF, want 0 (inactive)", before-after)
	}
	if c.SurfaceRef() != "face-key" {
		t.Errorf("SurfaceRef() = %q, want the retained face key", c.SurfaceRef())
	}
	if c.Residuals() != nil || c.Variables() != nil {
		t.Error("an unbound onFace should contribute no residual or variables")
	}
}
