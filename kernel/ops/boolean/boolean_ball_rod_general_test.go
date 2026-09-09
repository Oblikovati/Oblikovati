// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
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
	const ballR, rodR, rodLen = 5.0, 3.0, 15.0
	// A cylinder of radius 3 up to the crossing at y = √(25−9) = 4, plus the spherical cap above it.
	h := ballR - stdmath.Sqrt(ballR*ballR-rodR*rodR)
	plug := stdmath.Pi*rodR*rodR*stdmath.Sqrt(ballR*ballR-rodR*rodR) + stdmath.Pi*h*h*(3*ballR-h)/3
	ballVol := 4 * stdmath.Pi * ballR * ballR * ballR / 3
	rodVol := stdmath.Pi * rodR * rodR * rodLen
	for _, row := range []struct {
		name string
		op   brep.Op
		want float64
	}{
		{"ball and rod (the plug)", brep.Intersection, plug},
		{"ball less rod (a blind bore)", brep.Difference, ballVol - plug},
		{"ball with rod (the stud)", brep.Union, ballVol + rodVol - plug},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			res := coaxialBallRod(t, row.op, ballR, rodR, rodLen)
			if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
				t.Fatalf("not a valid solid: %+v", r.Issues)
			}
			// Three analytic faces every way round — the sphere, the rod's wall and one disc — no facets.
			if n := len(res.Faces()); n != 3 {
				t.Errorf("%d faces, want 3", n)
			}
			got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
			if rel := stdmath.Abs(got-row.want) / row.want; rel > 0.001 {
				t.Errorf("volume %.6f, want %.6f — rel %.5f", got, row.want, rel)
			}
		})
	}
}

// coaxialBallRod runs one boolean of a ball with a rod whose axis passes through its centre.
func coaxialBallRod(t *testing.T, op brep.Op, ballR, rodR, rodLen float64) *topo.Body {
	t.Helper()
	ball, err := brep.SolidSphere(math.P3(0, 0, 0), math.Scalar(ballR), "ball")
	if err != nil {
		t.Fatalf("ball: %v", err)
	}
	rod, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 1, 0), math.Scalar(rodR), math.Scalar(rodLen))
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	res, err := brep.Boolean(op, ball, rod)
	if err != nil {
		t.Fatalf("boolean: %v", err)
	}
	return res
}
