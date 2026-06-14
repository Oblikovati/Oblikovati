// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"strings"
	"testing"

	"oblikovati.org/head/internal/addinhost"
)

// TestIncompatibleNotice covers the status-bar message for version-skipped add-ins:
// empty when none were skipped, the id + reason for one, and a count + ids for many.
func TestIncompatibleNotice(t *testing.T) {
	if got := incompatibleNotice(nil); got != "" {
		t.Errorf("no skips: notice = %q, want empty", got)
	}

	one := []addinhost.IncompatibleAddIn{
		{ID: "com.x.bridge", Reason: "add-in built against API 1.5, newer than host API 1.4"},
	}
	got := incompatibleNotice(one)
	for _, want := range []string{"com.x.bridge", "incompatible API version", "newer than host API 1.4"} {
		if !strings.Contains(got, want) {
			t.Errorf("single skip notice %q missing %q", got, want)
		}
	}

	many := []addinhost.IncompatibleAddIn{
		{ID: "com.x.a", Reason: "r1"},
		{ID: "com.x.b", Reason: "r2"},
	}
	got = incompatibleNotice(many)
	for _, want := range []string{"2 add-ins skipped", "com.x.a", "com.x.b"} {
		if !strings.Contains(got, want) {
			t.Errorf("multi skip notice %q missing %q", got, want)
		}
	}
}
