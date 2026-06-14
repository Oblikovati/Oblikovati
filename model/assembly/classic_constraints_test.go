// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// TestAngleRotatesToTarget checks an angle constraint rotates the free component until its
// direction sits at the target angle from the grounded one.
func TestAngleRotatesToTarget(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Identity4())

	// Start the moving direction perpendicular to the base (a non-singular start; an
	// exactly-parallel start sits at the cosine's zero-gradient extremum) and drive to 45°.
	set := NewConstraintSet(occs, nil)
	set.AddAngle(ref(base, LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))), ref(moving, LinePrimitive(math.P3(0, 0, 0), unit(t, 1, 0, 0))), stdmath.Pi/4, types.AngleSolutionUndirected)
	if rep := set.Solve(); !rep.Converged {
		t.Fatalf("solve did not converge: %+v", rep)
	}

	dir := moving.Transform().TransformVector(math.V3(1, 0, 0))
	if cos := dir.Dot(math.V3(0, 0, 1)); stdmath.Abs(cos-stdmath.Cos(stdmath.Pi/4)) > 1e-6 {
		t.Errorf("cos(angle) = %v, want cos(45°)=%v", cos, stdmath.Cos(stdmath.Pi/4))
	}
}

// TestInsertSeatsAxis checks an insert makes the free component's axis collinear with the
// grounded one and seats it at the offset (here 0 → origin), leaving the spin free.
func TestInsertSeatsAxis(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(3, 4, 10)))

	set := NewConstraintSet(occs, nil)
	set.AddInsert(ref(base, LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))), ref(moving, LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))), 0, false)
	rep := set.Solve()

	if p := moving.Transform().Translation(); !math.V3(p.X, p.Y, p.Z).IsEqualTo(math.V3(0, 0, 0), 1e-6) {
		t.Errorf("moving axis point = %+v, want origin", p)
	}
	if got := dofOf(rep, moving.ID()); got != 1 {
		t.Errorf("inserted component DOF = %d, want 1 (spin)", got)
	}
}

// TestTangentHoldsRadius checks a plane-cylinder tangent moves the cylinder so its axis is
// one radius from the plane.
func TestTangentHoldsRadius(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(0, 0, 10)))

	set := NewConstraintSet(occs, nil)
	plane := PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))     // z = 0 plane
	cyl := CylinderPrimitive(math.P3(0, 0, 0), unit(t, 1, 0, 0), 2) // axis +X, radius 2
	set.AddTangent(ref(base, plane), ref(moving, cyl), false)
	if rep := set.Solve(); !rep.Converged {
		t.Fatalf("solve did not converge: %+v", rep)
	}

	if z := moving.Transform().Translation().Z; stdmath.Abs(z-2) > 1e-6 {
		t.Errorf("cylinder axis height = %v, want 2 (one radius above plane)", z)
	}
}

// TestSymmetryMirrors checks a symmetry constraint mirrors the free point across the plane.
func TestSymmetryMirrors(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(5, 5, 5)))

	set := NewConstraintSet(occs, nil)
	pointA := PointPrimitive(math.P3(3, 1, 2))
	pointB := PointPrimitive(math.P3(0, 0, 0))
	plane := PlanePrimitive(math.P3(0, 0, 0), unit(t, 1, 0, 0)) // x = 0 mirror plane
	set.AddSymmetry(ref(base, pointA), ref(moving, pointB), ref(base, plane))
	set.Solve()

	if got := moving.Transform().TransformPoint(math.P3(0, 0, 0)); !got.IsEqualTo(math.P3(-3, 1, 2), 1e-6) {
		t.Errorf("mirrored point = %+v, want (-3,1,2)", got)
	}
}
