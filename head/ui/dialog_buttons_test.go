//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strings"
	"testing"
)

// TestShortCommitReason keeps the disabled-OK explanation to one readable line: it drops a
// bracketed detail list (the kernel's per-edge dump) and caps the length, so a huge
// inconsistent-orientation reason does not flood the panel.
func TestShortCommitReason(t *testing.T) {
	dump := "fillet: result is not a valid solid [inconsistent orientation at edge 1 inconsistent orientation at edge 2 inconsistent orientation at edge 3]"
	got := shortCommitReason(dump)
	if strings.Contains(got, "[") {
		t.Errorf("bracketed detail list should be dropped, got %q", got)
	}
	if got != "fillet: result is not a valid solid" {
		t.Errorf("shortCommitReason = %q, want the summary before the detail list", got)
	}

	long := "fillet: " + strings.Repeat("x", 300)
	if s := shortCommitReason(long); len(s) > 145 || !strings.HasSuffix(s, "…") {
		t.Errorf("a long reason should be capped with an ellipsis, got len %d", len(s))
	}

	short := "fillet: cannot round an edge bordering a curved (cylinder) face"
	if s := shortCommitReason(short); s != short {
		t.Errorf("a short reason should pass through unchanged, got %q", s)
	}
}
