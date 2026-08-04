// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// FR5 — targeted coverage for the sphere-arm existence guards and cause-specific rejects the D5/E4 corpus
// (which always builds a clean arm) never triggers: the spindle (r≥R) and plane-clears declines of
// sphereArmSurface, and each sphereArmError message. Exact analytic inputs.

func unitZ(t *testing.T) math.UnitVector3 {
	t.Helper()
	u, err := math.UnitVector3FromVector(math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("unit +Z: %v", err)
	}
	return u
}

func TestSphereArmSurfaceBuildsAndDeclines(t *testing.T) {
	res := ResolutionForPoints([]math.Point3{math.P3(0, 0, 0), math.P3(0, 0, 300)})
	sp := geom.Sphere{Center: math.P3(0, 0, 0), Radius: 150}
	capPlane, _ := geom.NewPlane(math.P3(0, 0, 129.9038), math.V3(0, 0, 1))
	tor, reason := sphereArmSurface(sp, capPlane, unitZ(t), 10, res)
	if reason != sphereArmBuilt {
		t.Fatalf("sphereArmSurface(R150, capPlane, r10) reason=%d, want built", reason)
	}
	if stdmath.Abs(tor.MajorRadius-72.2709) > 1e-3 || tor.MinorRadius != 10 {
		t.Fatalf("sphereArmSurface torus major=%.4f minor=%.4f, want ~72.2709 / 10", tor.MajorRadius, tor.MinorRadius)
	}
	if _, reason := sphereArmSurface(geom.Sphere{Center: math.P3(0, 0, 0), Radius: 5}, capPlane, unitZ(t), 10, res); reason != sphereArmSpindle {
		t.Fatalf("sphereArmSurface(R5, r10) reason=%d, want sphereArmSpindle", reason)
	}
	far, _ := geom.NewPlane(math.P3(0, 0, 500), math.V3(0, 0, 1))
	if _, reason := sphereArmSurface(sp, far, unitZ(t), 10, res); reason != sphereArmClears {
		t.Fatalf("sphereArmSurface(far plane) reason=%d, want sphereArmClears", reason)
	}
}

func TestSphereArmErrorMessages(t *testing.T) {
	sp := geom.Sphere{Center: math.P3(0, 0, 0), Radius: 150}
	for _, tc := range []struct {
		reason sphereArmReject
		want   string
	}{
		{sphereArmVaryingRadius, "varying-radius"},
		{sphereArmSpindle, "spindle"},
		{sphereArmClears, "clears the offset sphere"},
		{sphereArmTangent, "grazing/tangent plane"},
	} {
		err := sphereArmError(tc.reason, nil, sp, nil, 10)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("sphereArmError(%d) = %v, want a message containing %q", tc.reason, err, tc.want)
		}
	}
}
