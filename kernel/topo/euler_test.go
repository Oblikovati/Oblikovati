// SPDX-License-Identifier: GPL-2.0-only

package topo

import "testing"

// TestEulerCharacteristicTetra checks the shared χ = V−E+2F−L accessor on a tetrahedron
// (V4, E6, F4, L4 → χ = 4−6+8−4 = 2) and its admissibility gate (#1600). Both the boolean's
// exact-tangent gate and the validator read χ through these, so they are covered directly here.
func TestEulerCharacteristicTetra(t *testing.T) {
	body := buildTetra()
	if chi := body.EulerCharacteristic(); chi != 2 {
		t.Fatalf("tetra χ = %d, want 2", chi)
	}
	if !body.EulerAdmissible() {
		t.Error("tetra χ=2 must be Euler-admissible")
	}
}

// TestEulerAdmissibleNonSolid confirms a non-solid (surface) body is unconstrained: EulerAdmissible
// reports true regardless of χ, since χ = 2−2g only binds a CLOSED orientable solid.
func TestEulerAdmissibleNonSolid(t *testing.T) {
	surf := NewBuilder(false, NewLineage(Tok("t", "surf", 0))).Build()
	if !surf.EulerAdmissible() {
		t.Error("a non-solid body must be Euler-unconstrained (admissible)")
	}
}
