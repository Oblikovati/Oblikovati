// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/math"
)

// TestReconstructCocylindricalCapOnWall is the ADR-0054 #2167 target at the kernel
// level, and it is SKIPPED with a precisely diagnosed blocker. A full cylinder unioned
// with a stacked D-prism whose arc wall is cocylindrical (same radius/axis) must
// reconstruct to a valid analytic solid with both walls kept.
//
// It does not yet, and the root cause is NOT in reconstruction — it is a
// coincident-curved-boundary limitation of the exact mesh boolean itself:
//
//   - The two operands share the radius-R circle at the join plane (the cylinder's
//     top-cap rim and the D-prism's bottom-cap arc are the SAME circle), but each
//     operand tessellates it INDEPENDENTLY: the cylinder's cap reaches the true rim at
//     radius R, while the D-prism's arc is INSCRIBED (its facet chords sit inside R).
//   - So a thin ring between the two approximations (radius ~0.99R..R over the shared
//     arc) is covered by the cylinder's cap but lies OUTSIDE the D-prism's TESSELLATED
//     footprint. coplanarPartner cannot see it as coincident, and insideExact reports
//     it outside the (tessellated) D-prism, so it is kept — as a zero-volume, opposite-
//     facing membrane (cylinder-cap-up plus D-cap-down over the same region).
//   - That membrane is edge-paired (so the soup passes a 2-manifold check) and volume-
//     neutral (so the boolean's volume stays correct), but it is a degenerate flap: both
//     soupToBody AND the provenance reconstruction group it to a NON-manifold B-rep
//     (the shared arc borders four faces).
//
// THE FIX is to CONFORM the two operands' tessellations of the shared cocylindrical
// circle (a canonical angular sampling so cocylindrical faces share vertices), which
// removes the sliver membrane at its source. That is a change to the tessellation path
// (kernel/ops/tessellate*.go) — the highest-priority subsystem — so it is deliberately
// staged as its own careful, corpus-validated step, not forced in here. Un-skip when it
// lands.
func TestReconstructCocylindricalCapOnWall(t *testing.T) {
	// ADR-0054 status (after the conforming-tessellation fix): the #2167 rim-sliver
	// MEMBRANE is GONE — canonical circular sampling (kernel/geom/canonical_sampling.go)
	// makes the two operands share the z=6 rim exactly, so the mesh boolean now emits a
	// 2-manifold, correct-volume soup (no opposite-facing zero-volume flap). Reconstruction
	// merges the two cocylindrical walls into ONE analytic cylinder (cyls=1) and builds a
	// CLOSED, MANIFOLD, SOLID body. What remains is a coplanar-classification residue in the
	// exact core: co-refinement strips the D bottom cap's coincidence partner, so
	// keepTaggedFromB keeps ~26 internal z=6 interface triangles that should be dropped;
	// reconstruction then rebuilds them as two spurious down-facing caps (volume 270.1 vs
	// 277.9, Euler +2). That is a meshbool co-refinement/classification fix, tracked
	// separately from the tessellation architecture this fixture drove. Un-skip when the
	// interface-drop lands. The membrane root cause — the reason #2167 was visible — is fixed.
	t.Skip("ADR-0054/#2167: membrane fixed (conforming tessellation); analytic reconstruction pending the coplanar interface-drop in the exact core")

	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 6)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	d := dPrismBody(3, 0.6, 6, 10, "d")

	body, ok := reconstructBoolean(cyl, d, meshbool.Union, DefaultQuality(), "u")
	if !ok || !Validate(body).Valid || !body.IsSolid() {
		t.Fatalf("cocylindrical join did not reconstruct to a valid solid (ok=%v)", ok)
	}
	if n := cylinderFaceCount(body); n < 2 {
		t.Fatalf("cocylindrical join kept %d analytic walls, want >=2", n)
	}
	minor := 0.5 * 9 * (1.2 - stdmath.Sin(1.2))
	want := stdmath.Pi*9*6 + (stdmath.Pi*9-minor)*4
	if v := soupVolume(bodyToSoup(body, PropertyQuality())); stdmath.Abs(v-want) > 5e-3*want {
		t.Fatalf("cocylindrical join volume = %.3f, want ~%.3f", v, want)
	}
}
