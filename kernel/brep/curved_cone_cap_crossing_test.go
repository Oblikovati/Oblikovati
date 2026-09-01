// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

func coneCapTarget(t *testing.T) *topo.Body {
	t.Helper()
	tg, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("target: %v", err)
	}
	return tg
}

// coneCapTool builds the certified oblique frustum (rBase 0.9 → rTop 0.6, 45°) that enters the wall once and
// exits the top cap through an ellipse strictly inside the rim.
func coneCapTool(t *testing.T) *topo.Body {
	t.Helper()
	s := 1 / stdmath.Sqrt2
	top := math.P3(math.Scalar(-6.5+16*s), 0, math.Scalar(2+16*s))
	tl, err := SolidCylinderCone(math.P3(-6.5, 0, 2), top, 0.9, 0.6, "cone")
	if err != nil {
		t.Fatalf("tool: %v", err)
	}
	return tl
}

// TestConeCapAcceptsInteriorExit: the oblique frustum enters the wall once and exits ONE cap through an
// ellipse inside the rim — the recognizer accepts and builds a four-face result.
func TestConeCapAcceptsInteriorExit(t *testing.T) {
	t.Parallel()
	res, ok := ConeCapCrossingCutGeneral(coneCapTarget(t), coneCapTool(t), &diag.Recorder{})
	if !ok {
		t.Fatal("cone-cap cut declined a genuine cone interior-exit tool")
	}
	if n := len(res.Faces()); n != 4 {
		t.Errorf("cone-cap result has %d faces; want 4 (holed wall + holed top cap + bottom cap + tunnel)", n)
	}
}

// TestConeCapDeclinesCylinderTool: a CYLINDER tool (slice-1's own fixture) is not a cone, so coneOperand
// fails and the cone recognizer declines — slice 1 (CapCrossingCutGeneral) owns that case.
func TestConeCapDeclinesCylinderTool(t *testing.T) {
	t.Parallel()
	s := 1 / stdmath.Sqrt2
	tool, err := SolidCylinder(math.P3(-6.5, 0, 2), math.V3(math.Scalar(s), 0, math.Scalar(s)), 0.9, 16)
	if err != nil {
		t.Fatalf("tool: %v", err)
	}
	if _, ok := ConeCapCrossingCutGeneral(coneCapTarget(t), tool, &diag.Recorder{}); ok {
		t.Error("cone-cap cut accepted a cylinder tool; want decline (slice 1 handles it)")
	}
}

// TestConeCapDeclinesNoContact: a frustum sitting entirely outside the target never crosses it — no cap
// exit, no wall entry — so the recognizer declines.
func TestConeCapDeclinesNoContact(t *testing.T) {
	t.Parallel()
	tool, err := SolidCylinderCone(math.P3(-12, 0, 2), math.P3(-9, 0, 5), 0.9, 0.6, "cone")
	if err != nil {
		t.Fatalf("tool: %v", err)
	}
	if _, ok := ConeCapCrossingCutGeneral(coneCapTarget(t), tool, &diag.Recorder{}); ok {
		t.Error("cone-cap cut accepted a non-contacting tool; want decline")
	}
}

// wallBand frames a v-axis band [0,10] along +z from the origin — the target's own cap band — for the
// wallEntryLoops unit tests.
func wallBand() ruledUV {
	return ruledUV{base: math.P3(0, 0, 0), axis: math.V3(0, 0, 1), band: coneSideBand_{vMin: 0, vMax: 10}}
}

func zLoop(zs ...float64) geom.Polyline {
	pts := make([]math.Point3, len(zs))
	for i, z := range zs {
		pts[i] = math.P3(3, 0, math.Scalar(z))
	}
	lp, _ := geom.NewPolyline(pts)
	return lp
}

// TestWallEntryLoopsKeepsInBandDropsPhantom is the crux of the cone slice: a long frustum's trace against the
// target's INFINITE surface yields a real in-band wall loop AND a phantom loop entirely above the cap. The
// filter keeps the former and drops the latter, so exactly one wall-entry hole survives.
func TestWallEntryLoopsKeepsInBandDropsPhantom(t *testing.T) {
	t.Parallel()
	inBand := zLoop(4.3, 5.5, 6.6)     // strictly between the caps: real wall entry
	phantom := zLoop(10.6, 11.5, 12.4) // entirely above z=10: past the finite wall
	kept, ok := wallEntryLoops(wallBand(), []geom.Curve3{phantom, inBand})
	if !ok {
		t.Fatal("wallEntryLoops declined a clean in-band + phantom pair")
	}
	if len(kept) != 1 {
		t.Fatalf("kept %d loops; want 1 (the in-band wall entry, phantom dropped)", len(kept))
	}
	if z := float64(imprintLoopPoints(kept[0])[0].Z); z > 10 {
		t.Errorf("kept the phantom loop (z=%.2f > 10) instead of the in-band one", z)
	}
}

// TestWallEntryLoopsDeclinesStraddle: a loop that CROSSES a cap level (some points below, some above) is a
// rim-crossing/cap-reaching breach this interior-exit slice does not build — the filter declines.
func TestWallEntryLoopsDeclinesStraddle(t *testing.T) {
	t.Parallel()
	straddle := zLoop(9.0, 9.8, 10.4, 11.0) // crosses z=10
	if _, ok := wallEntryLoops(wallBand(), []geom.Curve3{straddle}); ok {
		t.Error("wallEntryLoops accepted a cap-straddling loop; want decline (rim-crossing case)")
	}
}
