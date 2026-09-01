// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// rimCrossBodies builds the r=3 h=10 target and an oblique r=0.9 tool at the given base x, at 45° in the
// x–z plane (the slice-2 recognizer fixture; base -5.6 crosses the rim, base -6.5 exits inside it).
func rimCrossBodies(t *testing.T, baseX float64) (target, tool *topo.Body) {
	t.Helper()
	s := 1 / stdmath.Sqrt2
	tg, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("target: %v", err)
	}
	tl, err := SolidCylinder(math.P3(math.Scalar(baseX), 0, 2), math.V3(math.Scalar(s), 0, math.Scalar(s)), 0.9, 16)
	if err != nil {
		t.Fatalf("tool: %v", err)
	}
	return tg, tl
}

// TestRimCrossingAcceptsRimCrossing: the base -5.6 tool's exit ellipse crosses the top rim, so the slice-2
// recognizer builds a four-face result (notched wall + mixed-arc cap + bottom cap + tunnel).
func TestRimCrossingAcceptsRimCrossing(t *testing.T) {
	t.Parallel()
	target, tool := rimCrossBodies(t, -5.6)
	res, ok := RimCrossingCutGeneral(target, tool, &diag.Recorder{})
	if !ok {
		t.Fatal("rim-crossing cut declined a genuine rim-crossing tool")
	}
	if n := len(res.Faces()); n != 4 {
		t.Errorf("rim-crossing result has %d faces; want 4", n)
	}
}

// TestRimCrossingDeclinesInteriorExit: the base -6.5 tool exits STRICTLY inside the rim — slice 1's job — so
// the rim-crossing recognizer must decline (its two-corner cap gate finds zero corners).
func TestRimCrossingDeclinesInteriorExit(t *testing.T) {
	t.Parallel()
	target, tool := rimCrossBodies(t, -6.5)
	if _, ok := RimCrossingCutGeneral(target, tool, &diag.Recorder{}); ok {
		t.Error("rim-crossing cut accepted an interior-exit tool; want decline (slice 1 handles it)")
	}
}

// TestRimCrossingDeclinesNoContact: a tool whose axis clears the target entirely never reaches a cap, so the
// recognizer declines with no diagnostic-worthy trace.
func TestRimCrossingDeclinesNoContact(t *testing.T) {
	t.Parallel()
	target, tool := rimCrossBodies(t, -20)
	if _, ok := RimCrossingCutGeneral(target, tool, &diag.Recorder{}); ok {
		t.Error("rim-crossing cut accepted a tool that misses the target; want decline")
	}
}
