// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestNeutralHingeIsPlaneIntersection checks the neutral-plane hinge (#1801): drafting the +X face
// about a neutral plane at z=1 must pivot on the line x=2,z=1 (direction ±Y), not the face's lowest
// vertex. Both the direction and a point on it are verified against the two planes exactly.
func TestNeutralHingeIsPlaneIntersection(t *testing.T) {
	face, _ := geom.NewPlane(math.P3(2, 0, 0), math.V3(1, 0, 0))    // +X face at x=2
	neutral, _ := geom.NewPlane(math.P3(0, 0, 1), math.V3(0, 0, 1)) // z=1

	dir, pivot, ok := neutralHinge(face, neutral)
	if !ok {
		t.Fatal("neutralHinge returned ok=false for two perpendicular planes")
	}
	// Direction is ±Y (perpendicular to both normals).
	if stdmath.Abs(float64(dir.X())) > 1e-12 || stdmath.Abs(float64(dir.Z())) > 1e-12 ||
		stdmath.Abs(stdmath.Abs(float64(dir.Y()))-1) > 1e-12 {
		t.Errorf("hinge direction = %v, want ±Y", dir)
	}
	// Pivot lies on BOTH planes.
	if d := geom.SignedDistanceToPlane(face, pivot); stdmath.Abs(d) > 1e-9 {
		t.Errorf("pivot %v is %g off the face plane (x=2)", pivot, d)
	}
	if d := geom.SignedDistanceToPlane(neutral, pivot); stdmath.Abs(d) > 1e-9 {
		t.Errorf("pivot %v is %g off the neutral plane (z=1)", pivot, d)
	}
}

// TestNeutralHingeParallelPlanesNoHinge: a neutral plane parallel to the face has no intersection line.
func TestNeutralHingeParallelPlanesNoHinge(t *testing.T) {
	face, _ := geom.NewPlane(math.P3(2, 0, 0), math.V3(1, 0, 0))
	neutral, _ := geom.NewPlane(math.P3(5, 0, 0), math.V3(1, 0, 0)) // parallel to the face
	if _, _, ok := neutralHinge(face, neutral); ok {
		t.Error("parallel planes must yield no hinge (ok=false)")
	}
}
