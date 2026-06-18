// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestAddHatchRegion: a hatch region fills a rectangle with clipped parallel fill lines; cross-hatch
// produces about twice as many.
func TestAddHatchRegion(t *testing.T) {
	c := NewContent()
	ss := c.Sheets().Active().Sketches()
	sk, err := ss.AddHatchRegion("", 100, 100, 60, 40, types.HatchGeneral, 0)
	if err != nil {
		t.Fatalf("AddHatchRegion: %v", err)
	}
	single := sk.CurveCount()
	if single == 0 {
		t.Fatalf("general hatch produced no fill lines")
	}

	cross, err := ss.AddHatchRegion("", 200, 100, 60, 40, types.HatchCross, 0)
	if err != nil {
		t.Fatalf("AddHatchRegion cross: %v", err)
	}
	if cross.CurveCount() <= single {
		t.Errorf("cross-hatch fill = %d lines, want more than the single family (%d)", cross.CurveCount(), single)
	}
}

// TestHatchRegionClipped: every generated fill line lies within the hatched rectangle.
func TestHatchRegionClipped(t *testing.T) {
	c := NewContent()
	ss := c.Sheets().Active().Sketches()
	const x, y, w, h = 100.0, 100.0, 60.0, 40.0
	sk, err := ss.AddHatchRegion("", x, y, w, h, types.HatchGeneral, 0)
	if err != nil {
		t.Fatalf("AddHatchRegion: %v", err)
	}
	const eps = 1e-6
	for _, cv := range sk.Curves() {
		for _, p := range [][2]float64{{float64(cv.Start().X), float64(cv.Start().Y)}, {float64(cv.End().X), float64(cv.End().Y)}} {
			if p[0] < x-eps || p[0] > x+w+eps || p[1] < y-eps || p[1] > y+h+eps {
				t.Fatalf("hatch line endpoint %v outside rectangle [%g,%g]+%gx%g", p, x, y, w, h)
			}
		}
	}
}

// TestHatchRegionRejectsBadSize: a non-positive hatch size errors.
func TestHatchRegionRejectsBadSize(t *testing.T) {
	c := NewContent()
	ss := c.Sheets().Active().Sketches()
	if _, err := ss.AddHatchRegion("", 0, 0, 0, 40, types.HatchGeneral, 0); err == nil {
		t.Error("AddHatchRegion with width 0 = ok, want error")
	}
}

// TestHatchRegionPersists: a hatch region survives a save/open round-trip and re-renders its fill.
func TestHatchRegionPersists(t *testing.T) {
	c := NewContent()
	ss := c.Sheets().Active().Sketches()
	if _, err := ss.AddHatchRegion("", 100, 100, 60, 40, types.HatchCross, 0); err != nil {
		t.Fatalf("AddHatchRegion: %v", err)
	}
	before := ss.Item(0).CurveCount()
	data, err := c.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	restored := NewContent()
	if err := restored.ApplyRecipe(data); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	rs := restored.Sheets().Active().Sketches()
	if rs.Count() != 1 || rs.Item(0).CurveCount() != before {
		t.Fatalf("restored hatch = %d sketches / %d curves, want 1 / %d (fill re-rendered)", rs.Count(), sketchCurveCount(rs), before)
	}
}
