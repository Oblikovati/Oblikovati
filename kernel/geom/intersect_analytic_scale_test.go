// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// The analytic surface-intersection classifier must be SCALE-EQUIVARIANT
// (Oblikovati/Oblikovati#1399): a plane that cuts a fixed RELATIVE depth into a sphere has to
// classify the same way — section circle vs. grazing miss — at every model scale, because the
// clearance test now derives its tolerance from a model-relative Resolution rather than a
// cm-anchored absolute epsilon. With the old absolute 1e-9 a sub-resolution graze stayed a
// graze on a unit part but flipped into a spurious tiny circle on a km-scale copy; with the
// relative tolerance the result topology is invariant under uniform scaling.
func TestAnalyticIntersectionScaleEquivariant(t *testing.T) {
	t.Parallel()
	scales := []float64{1, 1e3, 1e6}
	cases := []struct {
		name       string
		depthFrac  float64 // plane inset below the sphere surface, as a fraction of the radius
		wantCurves int
	}{
		// A real, model-scale cut yields a section circle at every scale.
		{"deep cut keeps its section circle", 0.1, 1},
		// A cut shallower than the model resolution is a graze at every scale — it must NOT
		// materialise a sub-resolution circle on the large copy (the absolute-epsilon bug).
		{"sub-resolution cut grazes at every scale", 2e-10, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, k := range scales {
				radius := k
				res := ResolutionForSize(2 * radius) // bbox-diameter scale of the operand
				inset := tc.depthFrac * radius
				sp, err := NewSphere(math.P3(0, 0, 0), math.Scalar(radius))
				if err != nil {
					t.Fatalf("NewSphere(r=%g): %v", radius, err)
				}
				pl := mustPlane(t, 0, 0, radius-inset, 0, 0, 1)
				curves, ok := IntersectSurfacesAnalytic(pl, sp, res)
				if !ok {
					t.Fatalf("scale %g: handled=false, want handled", k)
				}
				if len(curves) != tc.wantCurves {
					t.Errorf("scale %g: got %d section curve(s), want %d (scale-equivariance broken)",
						k, len(curves), tc.wantCurves)
				}
			}
		})
	}
}

// The grazing classification at large scale is exactly where a cm-anchored epsilon misbehaves:
// pin the boundary so a regression to an absolute literal is caught. At km scale a cut 2e-4 cm
// deep (2e-10 relative) sits far below the model weld (~2e-3 cm) and must be a graze, while the
// same absolute depth on a unit part (well above its 2e-9 weld) is a genuine section circle.
func TestAnalyticGrazeIsRelativeNotAbsolute(t *testing.T) {
	t.Parallel()
	// Unit part, absolute cut depth 2e-4 cm → above the unit weld → a real circle.
	unit, _ := NewSphere(math.P3(0, 0, 0), 1)
	curves, ok := IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 1-2e-4, 0, 0, 1), unit, ResolutionForSize(2))
	if !ok || len(curves) != 1 {
		t.Fatalf("unit sphere, 2e-4 cm cut: got %d curves ok=%v, want one section circle", len(curves), ok)
	}
	// km part, SAME absolute 2e-4 cm cut → far below the km weld → a graze (no circle). An
	// absolute-epsilon classifier would wrongly report a section circle here.
	km, _ := NewSphere(math.P3(0, 0, 0), 1e6)
	curves, ok = IntersectSurfacesAnalytic(mustPlane(t, 0, 0, 1e6-2e-4, 0, 0, 1), km, ResolutionForSize(2e6))
	if !ok || len(curves) != 0 {
		t.Fatalf("km sphere, 2e-4 cm cut: got %d curves ok=%v, want a graze with no curve", len(curves), ok)
	}
}
