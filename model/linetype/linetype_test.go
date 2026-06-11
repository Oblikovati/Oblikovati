// SPDX-License-Identifier: GPL-2.0-only

package linetype

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
)

func TestBuiltinPatterns(t *testing.T) {
	for _, kind := range []types.SketchLineType{
		types.SketchLineDashed, types.SketchLineHidden,
		types.SketchLineCenter, types.SketchLinePhantom,
	} {
		if len(Builtin(kind)) == 0 {
			t.Errorf("Builtin(%s) = empty, want a dash pattern", kind)
		}
	}
	if Builtin(types.SketchLineContinuous) != nil || Builtin(types.SketchLineCustom) != nil {
		t.Error("continuous/custom must return nil (solid / definition-held)")
	}
}

func TestParseLINDefinitions(t *testing.T) {
	src := `;; comment line
*DASHDOT,Dash dot __ . __ .
A,.5,-.25,0,-.25

*BORDER,Border __ __ . __ __ .
A,.5,-.25,.5,-.25,0,-.25
`
	defs, err := ParseLIN(src)
	if err != nil {
		t.Fatalf("ParseLIN: %v", err)
	}
	if len(defs) != 2 || defs[0].Name != "DASHDOT" || defs[1].Name != "BORDER" {
		t.Fatalf("defs = %+v, want DASHDOT and BORDER", defs)
	}
	want := []float64{0.5, -0.25, 0, -0.25}
	for i, v := range defs[0].Pattern {
		if v != want[i] {
			t.Errorf("DASHDOT pattern[%d] = %v, want %v", i, v, want[i])
		}
	}
	if d, ok := Find(defs, "dashdot"); !ok || d.Name != "DASHDOT" {
		t.Errorf("Find is not case-insensitive: %v %v", d, ok)
	}
}

func TestParseLINRejections(t *testing.T) {
	bad := map[string]string{
		"pattern before header": `A,.5,-.25`,
		"empty name":            "*,desc\nA,.5",
		"text element":          "*GASLINE,gas\nA,.5,-.2,[\"GAS\",STANDARD],-.25",
		"non-numeric":           "*X,x\nA,.5,abc",
		"bad alignment":         "*X,x\nB,.5,-.25",
	}
	for name, src := range bad {
		if _, err := ParseLIN(src); err == nil {
			t.Errorf("%s: expected an error for %q", name, src)
		}
	}
}

// TestDashPolylineCoversLengthExactly checks pen-down + pen-up lengths add back up
// to the polyline length, and the pattern flows across the vertex.
func TestDashPolylineCoversLengthExactly(t *testing.T) {
	pts := []gmath.Point2{gmath.P2(0, 0), gmath.P2(1, 0), gmath.P2(1, 1)} // length 2
	segs := DashPolyline(pts, false, []float64{0.5, -0.25})
	down := 0.0
	for _, s := range segs {
		down += float64(s[0].DistanceTo(s[1]))
	}
	// cycle 0.75 → 2cm holds 2 full cycles (1.0 down) + 0.5 remainder, all pen-down.
	if stdmath.Abs(down-1.5) > 1e-9 {
		t.Errorf("pen-down length = %v, want 1.5", down)
	}
	// 3 dashes, but the second crosses the corner at arclength 1.0 and splits in two.
	if len(segs) != 4 {
		t.Errorf("segments = %d, want 4 (3 dashes, one split at the vertex)", len(segs))
	}
}

// TestDashPolylineDotsAndClosure: dots render as short dashes and a closed loop
// dashes its closing edge too.
func TestDashPolylineDotsAndClosure(t *testing.T) {
	square := []gmath.Point2{gmath.P2(0, 0), gmath.P2(1, 0), gmath.P2(1, 1), gmath.P2(0, 1)}
	segs := DashPolyline(square, true, []float64{0, -0.5})
	if len(segs) == 0 {
		t.Fatal("dot pattern produced no segments")
	}
	for i, s := range segs {
		if l := float64(s[0].DistanceTo(s[1])); stdmath.Abs(l-dotLength) > 1e-9 {
			t.Errorf("dot %d length = %v, want %v", i, l, dotLength)
		}
	}
	maxY := 0.0
	for _, s := range segs {
		maxY = stdmath.Max(maxY, float64(s[0].Y))
	}
	if maxY < 0.9 {
		t.Errorf("no dots on the closing half of the loop (maxY=%v) — closure not walked", maxY)
	}
}

// TestDashPolylineDegenerateInputs: nil pattern, zero-sum pattern, and short
// polylines must return nil rather than loop or panic.
func TestDashPolylineDegenerateInputs(t *testing.T) {
	pts := []gmath.Point2{gmath.P2(0, 0), gmath.P2(1, 0)}
	if DashPolyline(pts, false, nil) != nil {
		t.Error("nil pattern must return nil (solid)")
	}
	if DashPolyline(pts[:1], false, []float64{0.5, -0.25}) != nil {
		t.Error("1-point polyline must return nil")
	}
	if got := DashPolyline(pts, false, []float64{}); got != nil {
		t.Errorf("empty pattern = %v, want nil", got)
	}
}

// TestDashPolylineAllGapPattern pins that an all-gap pattern draws nothing but
// still terminates.
func TestDashPolylineAllGapPattern(t *testing.T) {
	pts := []gmath.Point2{gmath.P2(0, 0), gmath.P2(5, 0)}
	if segs := DashPolyline(pts, false, []float64{-0.5}); len(segs) != 0 {
		t.Errorf("all-gap pattern drew %d segments, want 0", len(segs))
	}
}
