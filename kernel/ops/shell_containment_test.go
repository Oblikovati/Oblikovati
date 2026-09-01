// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestShellContainmentClassifiesSingleShell exercises query.ShellContainment directly
// (commit 1261acd7 cut it over to the analytic brep.ClassifyShellPoint). Unlike
// query.BodyContainment, query.ShellContainment classifies against the region a SINGLE shell
// bounds — it does not see the other shell — so the outer shell treats its whole
// 4³ interior (cavity included) as inside, and the void shell treats only its own
// small 2³ region as inside. That distinction is the point of this test.
func TestShellContainmentClassifiesSingleShell(t *testing.T) {
	t.Parallel()
	body := cavityBody(t)
	q := DefaultQuality()

	outer, void := pickCavityShells(t, body)

	// Outer shell bounds the whole 4³ box region ([0,4]³, ignoring the cavity).
	if c := query.ShellContainment(outer, math.P3(0.5, 0.5, 0.5), q, 1e-6); c != query.ContainInside {
		t.Errorf("outer shell, deep interior point = %v, want inside", c)
	}
	if c := query.ShellContainment(outer, math.P3(0, 2, 2), q, 1e-6); c != query.ContainOn {
		t.Errorf("outer shell, wall point = %v, want on", c)
	}
	if c := query.ShellContainment(outer, math.P3(10, 0, 0), q, 1e-6); c != query.ContainOutside {
		t.Errorf("outer shell, far point = %v, want outside", c)
	}
	// The cavity center is INSIDE the outer shell's bounded region: a single shell
	// sees only its own faces, so the outer skin classifies the whole 4³ interior
	// as inside. query.BodyContainment, which nets the void skin's opposite contribution,
	// instead reads this same point as OUTSIDE the material (TestShellContainmentVerdicts).
	// That contrast is exactly the shell-region vs body-material distinction.
	if c := query.ShellContainment(outer, math.P3(2, 2, 2), q, 1e-6); c != query.ContainInside {
		t.Errorf("outer shell, cavity center = %v, want inside (shell region ignores the cavity)", c)
	}

	// Void shell bounds only the small 2³ cavity region ([1,3]³).
	if c := query.ShellContainment(void, math.P3(2, 2, 2), q, 1e-6); c != query.ContainInside {
		t.Errorf("void shell, cavity center = %v, want inside (inside the void region)", c)
	}
	// A point in the material between the two shells is outside the void's small region.
	if c := query.ShellContainment(void, math.P3(0.5, 0.5, 0.5), q, 1e-6); c != query.ContainOutside {
		t.Errorf("void shell, material point = %v, want outside (outside the small void region)", c)
	}
}

// pickCavityShells splits the two shells of a cavity body into its outer
// (material-enclosing) and void (cavity) skins by ShellIsVoidInBody.
func pickCavityShells(t *testing.T, body *topo.Body) (outer, void *topo.Shell) {
	t.Helper()
	shells := body.Shells()
	if len(shells) != 2 {
		t.Fatalf("cavity body has %d shells, want 2 (outer + void)", len(shells))
	}
	for _, sh := range shells {
		if ShellIsVoidInBody(body, sh) {
			void = sh
		} else {
			outer = sh
		}
	}
	if outer == nil || void == nil {
		t.Fatalf("expected one outer and one void shell, got outer=%v void=%v", outer, void)
	}
	return outer, void
}
