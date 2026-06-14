// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app/cmdline"
)

func TestCompletionLabelAndColor(t *testing.T) {
	if completionLabel("LINE", true) == completionLabel("LINE", false) {
		t.Error("selected and unselected completion labels should differ")
	}
	if completionColor(true) == completionColor(false) {
		t.Error("selected and unselected completion colours should differ")
	}
	if got := completionLabel("LINE", true); got[len(got)-4:] != "LINE" {
		t.Errorf("completionLabel should end with the word, got %q", got)
	}
}

func TestSeverityColorDistinct(t *testing.T) {
	seen := map[[4]float32]cmdline.Severity{}
	for _, sev := range []cmdline.Severity{cmdline.Info, cmdline.Echo, cmdline.Prompt, cmdline.Warning, cmdline.Error} {
		c := severityColor(sev)
		if other, dup := seen[c]; dup {
			t.Errorf("severity %v shares a colour with %v", sev, other)
		}
		seen[c] = sev
	}
}
