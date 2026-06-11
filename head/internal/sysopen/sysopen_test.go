// SPDX-License-Identifier: GPL-2.0-only

package sysopen

import "testing"

// TestRejectsNonHTTPSchemes guards the shell-escape filter: only web URLs reach
// the platform opener.
func TestRejectsNonHTTPSchemes(t *testing.T) {
	for _, url := range []string{"file:///etc/passwd", "javascript:alert(1)", "ftp://x", ""} {
		if err := (SystemOpener{}).OpenURL(url); err == nil {
			t.Errorf("OpenURL(%q) accepted a non-http(s) scheme", url)
		}
	}
}
