// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

func TestPhase0aEventIDsDistinct(t *testing.T) {
	t.Parallel()
	got := map[string]uint32{
		"refs":   uint32(PanelReferencesChanged{}.EventID()),
		"closed": uint32(TaskPanelClosed{}.EventID()),
		"value":  uint32(PanelValueChanged{}.EventID()),
	}
	if got["refs"] != 0x0514 {
		t.Fatalf("PanelReferencesChanged EventID = %#x, want 0x0514", got["refs"])
	}
	if got["closed"] != 0x0515 {
		t.Fatalf("TaskPanelClosed EventID = %#x, want 0x0515", got["closed"])
	}
	if got["refs"] == got["value"] || got["closed"] == got["value"] || got["refs"] == got["closed"] {
		t.Fatalf("event ids collide: %#v", got)
	}
}
