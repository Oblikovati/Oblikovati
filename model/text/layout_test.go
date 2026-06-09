// SPDX-License-Identifier: GPL-2.0-only

package text

import (
	"testing"

	"oblikovati.org/api/types"
)

// boundsOf returns the [minX,maxX,minY,maxY] of a set of contours.
func boundsOf(cs [][][2]float64) (minX, maxX, minY, maxY float64) {
	first := true
	for _, c := range cs {
		for _, p := range c {
			if first {
				minX, maxX, minY, maxY = p[0], p[0], p[1], p[1]
				first = false
				continue
			}
			minX, maxX = min2(minX, p[0]), max2(maxX, p[0])
			minY, maxY = min2(minY, p[1]), max2(maxY, p[1])
		}
	}
	return
}

func min2(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
func max2(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// floatPairs flattens contours to [][2]float64 for assertions.
func floatPairs(r Request) ([][][2]float64, error) {
	cs, err := AlignedOutlines(r, DefaultResolver())
	if err != nil {
		return nil, err
	}
	out := make([][][2]float64, len(cs))
	for i, c := range cs {
		pts := make([][2]float64, len(c))
		for j, p := range c {
			pts[j] = [2]float64{float64(p.X), float64(p.Y)}
		}
		out[i] = pts
	}
	return out, nil
}

func TestEmbeddedDefaultResolves(t *testing.T) {
	ft, err := DefaultResolver().Resolve("")
	if err != nil {
		t.Fatalf("resolve default: %v", err)
	}
	cs, err := ft.Outlines("A", 1.0)
	if err != nil {
		t.Fatalf("outlines: %v", err)
	}
	if len(cs) != 2 {
		t.Fatalf("'A' = %d contours, want 2 (outer + counter)", len(cs))
	}
}

func TestLeftAlignStartsAtOrigin(t *testing.T) {
	cs, err := floatPairs(Request{Content: "HI", Height: 1, HAlign: types.TextAlignLeft, VAlign: types.TextAlignBaseline})
	if err != nil {
		t.Fatal(err)
	}
	minX, _, minY, _ := boundsOf(cs)
	if minX < -1e-6 {
		t.Errorf("left-aligned minX = %.4f, want >= 0", minX)
	}
	if minY < -1e-6 {
		t.Errorf("baseline minY = %.4f, want >= 0 (caps above baseline)", minY)
	}
}

func TestCenterAlignStraddlesOrigin(t *testing.T) {
	cs, err := floatPairs(Request{Content: "HI", Height: 1, HAlign: types.TextAlignCenter, VAlign: types.TextAlignBaseline})
	if err != nil {
		t.Fatal(err)
	}
	minX, maxX, _, _ := boundsOf(cs)
	if minX >= 0 || maxX <= 0 {
		t.Errorf("center-aligned bounds [%.3f,%.3f] should straddle x=0", minX, maxX)
	}
	if d := minX + maxX; d > 0.05 || d < -0.05 {
		t.Errorf("center not balanced: minX+maxX = %.3f, want ~0", d)
	}
}

func TestRightAlignEndsAtOrigin(t *testing.T) {
	cs, err := floatPairs(Request{Content: "HI", Height: 1, HAlign: types.TextAlignRight, VAlign: types.TextAlignBaseline})
	if err != nil {
		t.Fatal(err)
	}
	_, maxX, _, _ := boundsOf(cs)
	if maxX > 1e-6 {
		t.Errorf("right-aligned maxX = %.4f, want <= 0", maxX)
	}
}

func TestMiddleVAlignCentresVertically(t *testing.T) {
	cs, err := floatPairs(Request{Content: "HI", Height: 1, VAlign: types.TextAlignMiddle})
	if err != nil {
		t.Fatal(err)
	}
	_, _, minY, maxY := boundsOf(cs)
	if mid := (minY + maxY) / 2; mid > 0.15 || mid < -0.15 {
		t.Errorf("middle valign mid-Y = %.3f, want ~0", mid)
	}
}

func TestFontSizeOverridesHeight(t *testing.T) {
	small, err := floatPairs(Request{Content: "I", Height: 1, FontSize: 0.5, VAlign: types.TextAlignBaseline})
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, maxYSmall := boundsOf(small)
	big, err := floatPairs(Request{Content: "I", Height: 1, VAlign: types.TextAlignBaseline})
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, maxYBig := boundsOf(big)
	if maxYSmall >= maxYBig {
		t.Errorf("fontSize=0.5 cap %.3f should be smaller than height=1 cap %.3f", maxYSmall, maxYBig)
	}
}
