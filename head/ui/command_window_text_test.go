// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app/cmdline"
)

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
