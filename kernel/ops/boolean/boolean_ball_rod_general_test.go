// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// The first curved-versus-curved crossing the GENERAL pipeline charts (ADR-0061 stage 4): a ball and a
// coaxial rod meet along a circle that lies on a sphere and on a cylinder. The mixed boolean used to
// decline the pair on box overlap alone and hand the whole ball-and-rod family to its bespoke
// recognizers; it now solves the crossing once and imprints it on both charts.
//
// It goes through brep.Boolean, BELOW the recognizer list, so it measures the general pipeline and not
// the recognizer that still claims this shape first.
func TestGeneralPipelineChartsACoaxialBallAndRod(t *testing.T) {
	t.Parallel()
	const ballR, rodR = 5.0, 3.0
	ball, err := brep.SolidSphere(math.P3(0, 0, 0), ballR, "ball")
	if err != nil {
		t.Fatalf("ball: %v", err)
	}
	rod, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 1, 0), rodR, 15)
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	plug, err := brep.Boolean(brep.Intersection, ball, rod)
	if err != nil {
		t.Fatalf("intersection: %v", err)
	}
	if r := ops.Validate(plug); !r.Valid || !plug.IsSolid() {
		t.Fatalf("the plug is not a valid solid: %+v", r.Issues)
	}
	// The rod's wall from the ball's centre to the crossing circle, its base disc, and the spherical
	// cap beyond — three analytic faces, no facets.
	if n := len(plug.Faces()); n != 3 {
		t.Errorf("the plug has %d faces, want 3 (wall, base disc, spherical cap)", n)
	}
	// A cylinder of radius 3 up to the crossing at y = √(25−9) = 4, plus the cap of height 1 above it.
	h := ballR - stdmath.Sqrt(ballR*ballR-rodR*rodR)
	want := stdmath.Pi*rodR*rodR*stdmath.Sqrt(ballR*ballR-rodR*rodR) + stdmath.Pi*h*h*(3*ballR-h)/3
	got := query.BodyGeometryProperties(plug, ops.DefaultQuality()).Volume
	if rel := stdmath.Abs(got-want) / want; rel > 0.001 {
		t.Errorf("plug volume %.6f, want %.6f — rel %.5f", got, want, rel)
	}
}
