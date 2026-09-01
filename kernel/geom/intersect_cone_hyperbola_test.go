// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// A plane parallel to a cone's axis cuts an exact hyperbola; every sampled point must lie on BOTH
// the cone surface and the plane (Oblikovati/Oblikovati#1372). The vertex distance from the apex
// along the axis is |D|/tanα, where D is the apex-to-plane offset.
func TestPlaneConeAxisParallelHyperbola(t *testing.T) {
	t.Parallel()
	cone, err := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/6) // tanα = 1/√3
	if err != nil {
		t.Fatalf("NewCone: %v", err)
	}
	pl, err := NewPlane(math.P3(2, 0, 0), math.V3(1, 0, 0)) // x=2, parallel to the z axis
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	curves, handled := IntersectSurfacesAnalytic(pl, cone, ResolutionForSize(1))
	if !handled || len(curves) != 1 {
		t.Fatalf("imprint handled=%v curves=%d, want handled=true with one curve", handled, len(curves))
	}
	h, ok := curves[0].(Hyperbola)
	if !ok {
		t.Fatalf("imprint curve is %T, want Hyperbola", curves[0])
	}
	wantA := 2 / stdmath.Tan(stdmath.Pi/6)
	if stdmath.Abs(h.A-wantA) > 1e-9 || stdmath.Abs(h.B-2) > 1e-9 {
		t.Errorf("semi-axes (A=%g,B=%g), want (A=%g,B=2)", h.A, h.B, wantA)
	}
	for _, theta := range []float64{-1.5, -0.4, 0, 0.4, 1.5} {
		p := h.PointAt(theta)
		if dx := float64(p.X) - 2; stdmath.Abs(dx) > 1e-9 {
			t.Errorf("θ=%g: point %v not on plane x=2 (Δ=%g)", theta, p, dx)
		}
		if off := coneRadialResidual(cone, p); off > 1e-9 {
			t.Errorf("θ=%g: point %v not on cone (residual=%g)", theta, p, off)
		}
	}
}

// A plane through the apex (D≈0) is the degenerate two-generator case and is deferred.
func TestPlaneConeThroughApexDeferred(t *testing.T) {
	t.Parallel()
	cone, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/6)
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 1, 0)) // contains the apex and the axis
	if _, handled := IntersectSurfacesAnalytic(pl, cone, ResolutionForSize(1)); handled {
		t.Error("plane through the apex should defer (handled=false)")
	}
}

// coneRadialResidual is |radial distance from axis − v·tanα| at a point: zero on the cone surface.
func coneRadialResidual(c Cone, p math.Point3) float64 {
	axis := c.AxisDir.AsVector()
	ap := c.Apex.VectorTo(p)
	v := float64(ap.Dot(axis))
	radial := ap.Sub(axis.Scale(ap.Dot(axis)))
	return stdmath.Abs(float64(radial.Length()) - v*stdmath.Tan(c.HalfAngle))
}
