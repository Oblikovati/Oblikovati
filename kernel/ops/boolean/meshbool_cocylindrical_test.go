// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/math"
)

// TestReconstructCocylindricalCapOnWall is the ADR-0056 #2167 target at the kernel level: a
// full cylinder unioned with a stacked D-prism whose arc wall is cocylindrical (same
// radius/axis) must reconstruct to a valid analytic solid.
//
// It exercises the whole ADR-0056 chain end to end:
//   - CONFORMING tessellation (canonical circular sampling) so the two operands share the
//     radius-R join-plane rim exactly — no rim-sliver membrane;
//   - CROSS-OPERAND VERTEX-ON-EDGE IMPRINT so the D-profile's chord corners, which sit on
//     the cylinder rim at a non-canonical angle, become vertices on the cylinder too — no
//     triple-point sliver the mesh co-refinement would otherwise leave;
//   - SAME-SURFACE MERGE so the lower cylinder wall and the upper cocylindrical arc wall
//     rebuild as ONE analytic cylinder face (the correct B-rep — a cocylindrical join is a
//     single face, exactly as Inventor/OCCT merge it), with the exposed cap trimmed to the
//     minor segment via SUB-ARC edge reuse.
//
// The result is a closed, manifold, solid B-rep of five analytic faces (bottom disc, the
// merged cylinder wall, the exposed minor-segment top, the D chord wall, the D top cap) at
// the exact stacked volume.
func TestReconstructCocylindricalCapOnWall(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~7s): `make test-corpus`")
	}
	t.Parallel()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 6)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	d := dPrismBody(3, 0.6, 6, 10, "d")

	body, ok := reconstructBoolean(cyl, d, meshbool.Union, DefaultQuality())
	if !ok {
		t.Fatal("cocylindrical join declined reconstruction")
	}
	if rep := Validate(body); !rep.Valid || !rep.Closed || !rep.Manifold || !body.IsSolid() {
		t.Fatalf("cocylindrical join is not a valid closed manifold solid: %+v", rep)
	}
	// The two cocylindrical walls MERGE into one analytic cylinder — the correct B-rep for a
	// coaxial-same-radius join. The faceted #2167 bug produced zero analytic cylinders.
	if n := cylinderFaceCount(body); n != 1 {
		t.Fatalf("cocylindrical join has %d analytic cylinder walls, want 1 (merged)", n)
	}
	// Exact stacked Volume: the cylinder plus the D-segment prism (disc minus the minor
	// segment the chord cuts). Re-tessellated fine, the analytic body holds it under 0.5%.
	minor := 0.5 * 9 * (1.2 - stdmath.Sin(1.2))
	want := stdmath.Pi*9*6 + (stdmath.Pi*9-minor)*4
	if v := soupVolume(bodyToSoup(body, PropertyQuality())); stdmath.Abs(v-want) > 5e-3*want {
		t.Fatalf("cocylindrical join volume = %.3f, want ~%.3f", v, want)
	}
}
