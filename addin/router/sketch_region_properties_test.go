// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/wire"
)

// TestSketchRegionPropertiesOverWire: the seeded 4×3 rectangle reports its
// exact section property set (M06-F08, #623).
func TestSketchRegionPropertiesOverWire(t *testing.T) {
	r, s := seededSession(t)
	var got wire.RegionPropertiesResult
	call(t, r, s, "sketch.regionProperties", `{"sketchIndex":0,"profileIndex":0}`, &got)

	if stdmath.Abs(got.Area-12) > 1e-9 || stdmath.Abs(got.Perimeter-14) > 1e-9 {
		t.Errorf("area/perimeter = %v/%v, want 12/14", got.Area, got.Perimeter)
	}
	if len(got.Centroid) != 2 || stdmath.Abs(got.Centroid[0]-2) > 1e-9 || stdmath.Abs(got.Centroid[1]-1.5) > 1e-9 {
		t.Errorf("centroid = %v, want (2, 1.5)", got.Centroid)
	}
	// 4 wide × 3 tall: Ixx = 4·27/12 = 9, Iyy = 3·64/12 = 16, Ixy = 0.
	if len(got.MomentsOfInertia) != 3 || stdmath.Abs(got.MomentsOfInertia[0]-9) > 1e-9 ||
		stdmath.Abs(got.MomentsOfInertia[1]-16) > 1e-9 || stdmath.Abs(got.MomentsOfInertia[2]) > 1e-9 {
		t.Errorf("moments = %v, want [9 16 0]", got.MomentsOfInertia)
	}
	if len(got.PrincipalMoments) != 2 || stdmath.Abs(got.PrincipalMoments[0]-16) > 1e-9 {
		t.Errorf("principal moments = %v, want I1 = 16", got.PrincipalMoments)
	}
	if got.Accuracy != "high" {
		t.Errorf("accuracy = %q, want the documented high default", got.Accuracy)
	}
	if len(got.PrincipalAxes) != 2 {
		t.Errorf("principal axes = %v, want two unit axes", got.PrincipalAxes)
	}
}

// TestSketchRegionPropertiesRejectsBadInput: unknown accuracy spellings and
// out-of-range profile indices error with the offending value.
func TestSketchRegionPropertiesRejectsBadInput(t *testing.T) {
	r, s := seededSession(t)
	if _, err := r.Handle(s, "sketch.regionProperties",
		[]byte(`{"sketchIndex":0,"profileIndex":0,"accuracy":"extreme"}`)); err == nil {
		t.Error("an unknown accuracy spelling must be rejected")
	}
	if _, err := r.Handle(s, "sketch.regionProperties",
		[]byte(`{"sketchIndex":0,"profileIndex":9}`)); err == nil {
		t.Error("an out-of-range profile index must be rejected")
	}
}
