// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/linetype"
	"oblikovati.org/model/sketch"
)

// lineStyleSketch builds a sketch with a normal, a construction, and a centerline line.
func lineStyleSketch(t *testing.T) (*sketch.Sketch, *sketch.Line, *sketch.Line, *sketch.Line) {
	t.Helper()
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	normal := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	cons := sk.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(1, 1))
	cons.SetConstruction(true)
	axis := sk.Lines().AddByTwoPoints(math.P2(0, 2), math.P2(1, 2))
	axis.SetCenterline(true)
	return sk, normal, cons, axis
}

// TestSketchEntityPatternStyles pins the #161 rendering rules: centerline → center
// pattern, construction → dashed, normal → solid by default.
func TestSketchEntityPatternStyles(t *testing.T) {
	sk, normal, cons, axis := lineStyleSketch(t)
	if p := SketchEntityPattern(sk, normal); p != nil {
		t.Errorf("normal line pattern = %v, want nil (solid)", p)
	}
	if p := SketchEntityPattern(sk, cons); len(p) != 2 {
		t.Errorf("construction pattern = %v, want the 2-element dashed pattern", p)
	}
	if p := SketchEntityPattern(sk, axis); len(p) != 4 {
		t.Errorf("centerline pattern = %v, want the 4-element center pattern", p)
	}
}

// TestSketchEntityPatternOverrides: a sketch-level line type styles normal geometry,
// and a loaded custom definition wins over the built-ins.
func TestSketchEntityPatternOverrides(t *testing.T) {
	sk, normal, cons, _ := lineStyleSketch(t)
	sk.SetLineType("hidden")
	if p := SketchEntityPattern(sk, normal); len(p) != 2 {
		t.Errorf("hidden override pattern = %v, want the hidden pattern", p)
	}
	if p := SketchEntityPattern(sk, cons); len(p) != 2 {
		t.Errorf("construction must stay dashed under an override, got %v", p)
	}
	sk.SetCustomLineType(linetype.Definition{Name: "X", Pattern: []float64{1, -0.5, 0}}, "x.lin")
	if p := SketchEntityPattern(sk, normal); len(p) != 3 {
		t.Errorf("custom pattern = %v, want the 3-element loaded pattern", p)
	}
}
