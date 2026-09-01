// SPDX-License-Identifier: GPL-2.0-only

package testimpact

import (
	"path/filepath"
	"sync"
	"testing"
)

// TestToolBinariesResolveAbsolute pins the go:S4036 hardening: when the tool is on PATH
// the commands run a resolved ABSOLUTE path, not a bare name a mutated PATH could
// re-point. go and git are both present wherever this suite runs.
func TestToolBinariesResolveAbsolute(t *testing.T) {
	for name, got := range map[string]string{"go": goBinary(), "git": gitBinary()} {
		if !filepath.IsAbs(got) {
			t.Errorf("%s resolved to %q, want an absolute path", name, got)
		}
	}
}

// TestToolBinaryFallsBackToTheBareName pins the other half: a tool that is not on PATH
// keeps its bare name, so the failure reads as the command's own "not found" rather than
// as an empty path.
func TestToolBinaryFallsBackToTheBareName(t *testing.T) {
	var once = new(sync.Once)
	var dst string
	if got := toolBinary("definitely-not-a-real-binary-9f3c", once, &dst); got != "definitely-not-a-real-binary-9f3c" {
		t.Errorf("toolBinary() = %q, want the bare name back", got)
	}
}
