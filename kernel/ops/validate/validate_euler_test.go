// SPDX-License-Identifier: GPL-2.0-only

package validate

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestEulerAdmissible is the admissibility gate at the heart of the Euler check: χ for a closed
// orientable solid must be EVEN and ≤ 2 per shell (χ = Σ 2−2g, genus ≥ 0). It catches the odd or
// too-large χ a dropped/doubled face would produce while every edge stays used twice (#1407).
func TestEulerAdmissible(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		chi, shells int
		want        bool
	}{
		{2, 1, true},   // a sphere/box (genus 0)
		{0, 1, true},   // a torus shell (genus 1)
		{-2, 1, true},  // genus 2
		{4, 2, true},   // two genus-0 shells
		{3, 1, false},  // odd χ — impossible for an orientable closed manifold
		{4, 1, false},  // χ > 2 on one shell — implies negative genus
		{-1, 1, false}, // odd (negative)
	} {
		if got := eulerAdmissible(c.chi, c.shells); got != c.want {
			t.Errorf("eulerAdmissible(χ=%d, shells=%d) = %v, want %v", c.chi, c.shells, got, c.want)
		}
	}
}

// TestValidateEulerCharacteristic is #1407 criterion 2: the validator computes the Euler–Poincaré
// characteristic χ = V−E+2F−L and admits it for a closed orientable solid. A simple solid has χ = 2;
// an oblique cylinder cut — whose result has a HOLED face, so the naive V−E+F would be 3 (odd) and
// falsely flagged — still computes χ = 2 through the loop-corrected form. Both are EulerConsistent and
// Valid, confirming the check admits real B-reps (including holed and periodic faces) rather than
// false-positiving on them.
func TestValidateEulerCharacteristic(t *testing.T) {
	t.Parallel()
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	cutTarget, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 4)
	plane, _ := geom.NewPlane(math.P3(0, 0, 2), math.V3(1, 0, 2))
	oblique, err := brep.HalfSpaceCut(cutTarget, plane)
	if err != nil {
		t.Fatalf("oblique cut: %v", err)
	}

	for _, c := range []struct {
		name string
		body *topo.Body
		chi  int
	}{
		{"genus-0 cylinder", cyl, 2},
		{"oblique cut (holed face)", oblique, 2},
	} {
		r := Validate(c.body)
		if r.EulerCharacteristic != c.chi {
			t.Errorf("%s: Euler χ = %d, want %d", c.name, r.EulerCharacteristic, c.chi)
		}
		if !r.EulerConsistent {
			t.Errorf("%s: EulerConsistent = false, want true (χ=%d should be admissible)", c.name, r.EulerCharacteristic)
		}
		if !r.Valid {
			t.Errorf("%s: a valid closed solid was reported invalid: %+v", c.name, r)
		}
	}
}
