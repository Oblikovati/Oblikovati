// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// firstCylinderFace returns the body's cylinder side as a curvedFace (the recogniser's input).
func firstCylinderFace(t *testing.T, faces []curvedFace) curvedFace {
	t.Helper()
	for _, f := range faces {
		if _, ok := f.surface.(geom.Cylinder); ok {
			return f
		}
	}
	t.Fatal("no cylinder side face on body")
	return curvedFace{}
}

// notchedCylinder clips a cylinder with an oblique plane so ONE rim survives as a full circle and the other is
// notched (a rim arc + a section ellipse) — the already-cut side cutCylinderSideBand must accept/decline on.
// topNotch true clips the top rim (bottom survives = valid v=0 anchor); false clips the bottom (anchor gone).
func notchedCylinder(t *testing.T, topNotch bool) []curvedFace {
	t.Helper()
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	var pl geom.Plane
	if topNotch {
		pl, _ = geom.NewPlane(math.P3(1.5, 0, 8), math.V3(1, 0, 1)) // wedge off the top, bottom disc intact
	} else {
		pl, _ = geom.NewPlane(math.P3(1.5, 0, 2), math.V3(1, 0, -1)) // wedge off the bottom, top disc intact
	}
	out, err := HalfSpaceCut(cyl, pl)
	if err != nil {
		t.Fatalf("half-space cut: %v", err)
	}
	return facesOfAny(out)
}

// TestCutCylinderSideBandAcceptsTopNotch: a cylinder whose top rim was clipped keeps a full bottom circle plus
// a notched top loop, so cutCylinderSideBand accepts it, anchors v=0 on the bottom, and recovers vMax at the
// surviving top-rim height (10) plus the pad — never the removed original height by assumption, but here the
// top-rim arc survives so vMax rounds to ~10.
func TestCutCylinderSideBandAcceptsTopNotch(t *testing.T) {
	f := firstCylinderFace(t, notchedCylinder(t, true))
	cyl, band, prior, ok := cutCylinderSideBand(f)
	if !ok {
		t.Fatal("top-notched cylinder side declined; want accepted with a v=0 bottom anchor")
	}
	if cyl.Radius != 3 {
		t.Errorf("radius %v, want 3", cyl.Radius)
	}
	if band.vMin != 0 || band.rBot != 3 {
		t.Errorf("band vMin=%v rBot=%v, want 0 and 3", band.vMin, band.rBot)
	}
	if band.vMax <= 10 || band.vMax > 10.1 {
		t.Errorf("recovered vMax=%v, want just above the surviving top-rim height 10 (pad ~3e-3)", band.vMax)
	}
	if len(prior.edges) == 0 {
		t.Error("prior trim loop is empty; want the notched-rim boundary (arc + section conic)")
	}
	if hasFullCircle(prior) {
		t.Error("prior trim loop contains a full circle; the anchor rim must have been split out")
	}
}

// TestCutCylinderSideBandDeclinesBottomNotch: clipping the BOTTOM rim leaves the surviving full circle at the
// TOP with the notch below it, so the v=0 anchor is gone — cutCylinderSideBand must decline (band-extent
// recovery has no bottom to anchor on).
func TestCutCylinderSideBandDeclinesBottomNotch(t *testing.T) {
	f := firstCylinderFace(t, notchedCylinder(t, false))
	if _, _, _, ok := cutCylinderSideBand(f); ok {
		t.Fatal("bottom-notched cylinder side accepted; the v=0 anchor rim was removed and must decline")
	}
}

// TestCutCylinderSideBandDeclinesBareCylinder: a bare cylinder side has TWO full-circle rims — fullCylinderSideBand's
// case — so cutCylinderSideBand declines it, keeping the two operands partitioned (no double classification).
func TestCutCylinderSideBandDeclinesBareCylinder(t *testing.T) {
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	f := firstCylinderFace(t, facesOfAny(cyl))
	if _, _, _, ok := cutCylinderSideBand(f); ok {
		t.Fatal("bare two-circle cylinder side accepted by cutCylinderSideBand; must be cylinderOperand's job")
	}
}

// hasFullCircle reports whether the prior loop still carries a whole-rim circle (it must not — the anchor).
func hasFullCircle(prior priorTrimLoop) bool {
	for _, e := range prior.edges {
		if _, ok := fullRimCircle(e); ok {
			return true
		}
	}
	return false
}
