//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strings"
	"testing"
)

// monoWidth is a stand-in text measurer: every character is 10 units wide. It makes the wrap
// arithmetic exact and independent of any font atlas, so these tests need no window.
func monoWidth(s string) float32 { return float32(len(s)) * 10 }

func TestWrapCaptionLeavesShortLabelsAlone(t *testing.T) {
	for _, label := range []string{"Hole", "Rib", "", "Extrude"} {
		got := wrapCaption(label, 100, monoWidth)
		if len(got) != 1 || got[0] != label {
			t.Errorf("wrapCaption(%q, 100) = %q, want the label on one line", label, got)
		}
	}
}

// TestWrapCaptionBalancesTheSplit is the reason wrapCaption picks the minimising boundary
// rather than the first one that fits: "New 2D Sketch" must read "New 2D" / "Sketch", not
// "New" / "2D Sketch", so the tile is as narrow as the label allows.
func TestWrapCaptionBalancesTheSplit(t *testing.T) {
	got := wrapCaption("New 2D Sketch", 60, monoWidth)
	want := []string{"New 2D", "Sketch"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("wrapCaption(\"New 2D Sketch\", 60) = %q, want %q", got, want)
	}
}

func TestWrapCaptionSplitsTwoWordLabels(t *testing.T) {
	got := wrapCaption("Replace Face", 60, monoWidth)
	want := []string{"Replace", "Face"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("wrapCaption(\"Replace Face\", 60) = %q, want %q", got, want)
	}
}

// TestWrapCaptionCannotBreakOneWord: a single long word has no boundary, so it stays whole and
// widens its tile — clipping a command name would be worse, and the panel can still collapse.
func TestWrapCaptionCannotBreakOneWord(t *testing.T) {
	got := wrapCaption("Circumference", 20, monoWidth)
	if len(got) != 1 || got[0] != "Circumference" {
		t.Errorf("wrapCaption on a single long word = %q, want it left whole on one line", got)
	}
}

// TestWrapCaptionNeverExceedsTheLineBudget: the band reserves height for exactly
// largeCaptionMaxLines, so wrapCaption must never return more however long the label.
func TestWrapCaptionNeverExceedsTheLineBudget(t *testing.T) {
	got := wrapCaption("Extract Shape From Selected Surface Body", 30, monoWidth)
	if len(got) > largeCaptionMaxLines {
		t.Errorf("wrapCaption returned %d lines, want at most largeCaptionMaxLines=%d",
			len(got), largeCaptionMaxLines)
	}
}

// TestWrapCaptionKeepsEveryWord: wrapping is a layout change, never a content change — no word
// of a command name may be dropped on the way to the second line.
func TestWrapCaptionKeepsEveryWord(t *testing.T) {
	const label = "Delete Face From Body"
	got := strings.Join(wrapCaption(label, 40, monoWidth), " ")
	if got != label {
		t.Errorf("wrapCaption round-trip = %q, want %q (no word lost)", got, label)
	}
}
