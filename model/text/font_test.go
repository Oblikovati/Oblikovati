// SPDX-License-Identifier: GPL-2.0-only

package text

import (
	"os"
	"path/filepath"
	"testing"
)

// arialPath returns a located Arial .ttf, or "" to skip (Arial is not redistributable, so
// it is not vendored; tests that need it find a system copy or skip).
func arialPath() string {
	candidates := []string{
		"/home/vmiguel/.steam/debian-installation/steamapps/common/Proton - Experimental/files/share/fonts/arial.ttf",
		"/usr/share/fonts/truetype/msttcorefonts/Arial.ttf",
		"/usr/share/fonts/truetype/arial.ttf",
		"/Library/Fonts/Arial.ttf",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	matches, _ := filepath.Glob(os.Getenv("HOME") + "/.steam/**/arial.ttf")
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func loadArial(t *testing.T) *Font {
	t.Helper()
	p := arialPath()
	if p == "" {
		t.Skip("Arial .ttf not found on this system; skipping true-type test")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	f, err := Parse(data)
	if err != nil {
		t.Fatalf("parse Arial: %v", err)
	}
	return f
}

// TestArialLetterAHasCounter checks that an 'A' produces two contours (outer boundary + the
// triangular counter/hole), the hallmark of real glyph geometry, at roughly the asked height.
func TestArialLetterAHasCounter(t *testing.T) {
	f := loadArial(t)
	contours, err := f.Outlines("A", 1.0) // 1 cm em
	if err != nil {
		t.Fatalf("Outlines(A): %v", err)
	}
	if len(contours) != 2 {
		t.Fatalf("Arial 'A' = %d contours, want 2 (outer + counter)", len(contours))
	}
	// Cap height of A is roughly 0.7 em; assert the glyph spans a sane fraction of 1 cm.
	var maxY float64
	for _, c := range contours {
		for _, p := range c {
			if y := float64(p.Y); y > maxY {
				maxY = y
			}
		}
	}
	if maxY < 0.5 || maxY > 1.0 {
		t.Errorf("Arial 'A' top = %.3f cm, want ~0.7 (cap height of a 1 cm em)", maxY)
	}
}

// TestArialLetterIIsSingleContour checks a glyph with no counter yields one contour.
func TestArialLetterIIsSingleContour(t *testing.T) {
	f := loadArial(t)
	contours, err := f.Outlines("I", 1.0)
	if err != nil {
		t.Fatalf("Outlines(I): %v", err)
	}
	if len(contours) != 1 {
		t.Fatalf("Arial 'I' = %d contours, want 1", len(contours))
	}
}
