// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cap-crossing slice-1 recognizer guards (EPIC Oblikovati/Oblikovati#1724, ADR-0046). CapCrossingCutGeneral
// is a TIGHT POSITIVE recognizer: it builds ONLY the interior-exit case (oblique cylinder enters the wall
// once, exits one planar cap through an ellipse strictly inside the rim). Every neighbouring configuration
// must DECLINE (ok=false) so ops.Boolean keeps its recorded CSG fallback — a partial curved-boolean that
// silently built a manifold-but-wrong solid for an out-of-slice case would be worse than the fallback.

func capCrossTool45(baseX float64) *topo.Body {
	s := 1 / stdmath.Sqrt2
	tl, _ := SolidCylinder(math.P3(math.Scalar(baseX), 0, 2), math.V3(math.Scalar(s), 0, math.Scalar(s)), 0.9, 16)
	return tl
}

// TestCapCrossingAcceptsInteriorExit is the positive control: the certified slice-1 fixture builds.
func TestCapCrossingAcceptsInteriorExit(t *testing.T) {
	t.Parallel()
	target, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	res, ok := CapCrossingCutGeneral(target, capCrossTool45(-6.5), nil)
	if !ok {
		t.Fatal("interior-exit cap-crossing cut should be recognised and built")
	}
	if n := len(res.Faces()); n != 4 {
		t.Errorf("interior-exit result has %d faces; want 4 (holed wall + holed top cap + bottom cap + tunnel)", n)
	}
}

// TestCapCrossingDeclinesRimCrossing: shifting the tool outward pushes the exit ellipse PAST the cap rim (a
// rim-crossing corner, the deferred sub-family), so ellipseInsideRim fails and slice 1 declines.
func TestCapCrossingDeclinesRimCrossing(t *testing.T) {
	t.Parallel()
	target, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	// base x=-5.6 → exit-ellipse centre near x=2.4, semi-major 1.27 reaches ~3.67 > rim 3 → crosses the rim.
	if _, ok := CapCrossingCutGeneral(target, capCrossTool45(-5.6), nil); ok {
		t.Error("a tool whose exit ellipse crosses the cap rim must decline (deferred rim-crossing corner)")
	}
}

// TestCapCrossingDeclinesNoCapExit: a horizontal tool crossing the wall but staying strictly between the
// caps has no cap-exit ellipse — the plain crossing cut CrossingCylinderCutGeneral owns it, so the
// cap-crossing recognizer declines.
func TestCapCrossingDeclinesNoCapExit(t *testing.T) {
	t.Parallel()
	target, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	horizontal, _ := SolidCylinder(math.P3(-6, 0, 5), math.V3(1, 0, 0), 0.9, 12)
	if _, ok := CapCrossingCutGeneral(target, horizontal, nil); ok {
		t.Error("a wall-only crossing (no cap exit) must decline — it is the plain crossing cut")
	}
}

// TestCapCrossingDeclinesNearCoaxial: a tool nearly parallel to the target axis meets the cap almost
// head-on (|n·d| > capEllipseCosMax), so there is no transversal wall crossing — capExitEllipse gates it
// out and slice 1 declines (near-coaxial is the drill/boss family, not a cap crossing).
func TestCapCrossingDeclinesNearCoaxial(t *testing.T) {
	t.Parallel()
	target, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	nearAxial, _ := SolidCylinder(math.P3(1, 0, -2), math.V3(0.02, 0, 1), 0.9, 16)
	if _, ok := CapCrossingCutGeneral(target, nearAxial, nil); ok {
		t.Error("a near-coaxial tool (no transversal wall crossing) must decline")
	}
}
