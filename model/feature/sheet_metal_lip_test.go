// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"
)

// lipDef returns a lip recipe with a 90° flange of the given height and a return wall — the
// common stiffening-lip case.
func lipDef(edge []byte, height, returnLen float64) *SheetMetalLipDefinition {
	return &SheetMetalLipDefinition{
		EdgeKey:      edge,
		Height:       func() float64 { return height },
		Radius:       func() float64 { return 0.2 },
		Angle:        func() float64 { return stdmath.Pi / 2 },
		ReturnLength: func() float64 { return returnLen },
	}
}

// TestLipBuildsWatertightStiffener a lip folds a flange + 180° return onto an edge, yielding one
// watertight solid that rises above the sheet and adds material (the lip band).
func TestLipBuildsWatertightStiffener(t *testing.T) {
	t.Parallel()
	fs, edge := seedSheetMetalSheet(t, 4, map[string]string{"BendRadius": "2 mm"})
	base := sheetVolume(fs.Result()[0])

	pf := NewSheetMetalLipFeatures(fs).Add(lipDef(edge.ReferenceKey(), 1.0, 0.4))
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("lip sick: %s", pf.Health().Reason)
	}
	if n := len(fs.Result()); n != 1 {
		t.Fatalf("lip result = %d bodies, want 1", n)
	}
	body := fs.Result()[0]
	assertWatertightSolid(t, body)
	if v := sheetVolume(body); !(v > base) {
		t.Errorf("lip added no material: %g vs base %g", v, base)
	}
	if z := body.RangeBox().Max.Z; z < 0.8 {
		t.Errorf("lip flange should rise (maxZ=%g), want it to stand up", z)
	}
}

// TestLipRejectsBadDims a lip with a non-positive height/return errors.
func TestLipRejectsBadDims(t *testing.T) {
	t.Parallel()
	fs, edge := seedSheetMetalSheet(t, 4, map[string]string{"BendRadius": "2 mm"})
	pf := NewSheetMetalLipFeatures(fs).Add(lipDef(edge.ReferenceKey(), 1.0, 0)) // zero return length
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("lip with a zero return length should be sick")
	}
}
