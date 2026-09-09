// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A flush contact on an exact-framed face (ADR-0060, #3508): a cylinder whose top cap — a circle rim
// with a square hole pocketed through it — meets a block IN that cap's plane. Before this the mixed
// boolean declined every coplanar pair carrying a conic loop and the operation fell to triangle CSG.

// holedCapCylinder is a Ø2×2 cylinder along Z with a 0.6×0.6 pocket punched through its top cap.
func holedCapCylinder(t *testing.T) *topo.Body {
	t.Helper()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1, 2)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	pocket, _ := brep.SolidBlock(math.P3(-0.3, -0.3, 1.5), math.P3(0.3, 0.3, 2.5), "pocket")
	res, err := brep.BooleanDiag(brep.Difference, cyl, pocket, nil)
	if err != nil {
		t.Fatalf("pocket: %v", err)
	}
	return res
}

func holedCapVolume() float64 { return stdmath.Pi*2 - 0.6*0.6*0.5 }

func TestFlushJoinOnAHoledCapStaysExact(t *testing.T) {
	t.Parallel()
	body := holedCapCylinder(t)
	block, _ := brep.SolidBlock(math.P3(0, -1.5, 2), math.P3(1.5, 1.5, 2.5), "block")
	res, err := brep.BooleanDiag(brep.Union, body, block, nil)
	if err != nil {
		t.Fatalf("flush join declined: %v", err)
	}
	if rep := ops.Validate(res); !rep.ValidSolid() {
		t.Fatalf("flush join invalid: %+v", rep)
	}
	want := holedCapVolume() + 1.5*3*0.5
	if got := vol(res); stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("flush join volume = %.12g, want %.12g", got, want)
	}
	if n := cylinderFaceCount(res); n != 1 {
		t.Errorf("flush join has %d cylinder faces, want the one wall", n)
	}
}

func TestFlushCutBelowAHoledCapStaysExact(t *testing.T) {
	t.Parallel()
	body := holedCapCylinder(t)
	// The tool's TOP face lies in the cap's plane: the cut removes the slab z∈[1.7,2] of the x≥0 half,
	// less the part the pocket had already taken.
	block, _ := brep.SolidBlock(math.P3(0, -1.5, 1.7), math.P3(1.5, 1.5, 2), "block")
	res, err := brep.BooleanDiag(brep.Difference, body, block, nil)
	if err != nil {
		t.Fatalf("flush cut declined: %v", err)
	}
	if rep := ops.Validate(res); !rep.ValidSolid() {
		t.Fatalf("flush cut invalid: %+v", rep)
	}
	want := holedCapVolume() - (0.3*stdmath.Pi/2 - 0.3*0.6*0.3)
	if got := vol(res); stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("flush cut volume = %.12g, want %.12g", got, want)
	}
}

func TestFlushTouchAboveAHoledCapRemovesNothing(t *testing.T) {
	t.Parallel()
	body := holedCapCylinder(t)
	block, _ := brep.SolidBlock(math.P3(0, -1.5, 2), math.P3(1.5, 1.5, 2.5), "block")
	res, err := brep.BooleanDiag(brep.Difference, body, block, nil)
	if err != nil {
		t.Fatalf("flush touch declined: %v", err)
	}
	if rep := ops.Validate(res); !rep.ValidSolid() {
		t.Fatalf("flush touch invalid: %+v", rep)
	}
	if got := vol(res); stdmath.Abs(got-holedCapVolume()) > 1e-9 {
		t.Errorf("a tool touching the cap from above removed %.12g", holedCapVolume()-got)
	}
}
