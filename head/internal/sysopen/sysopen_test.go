// SPDX-License-Identifier: GPL-2.0-only

package sysopen

import (
	"path/filepath"
	"testing"
)

// TestRejectsNonHTTPSchemes guards the shell-escape filter: only web URLs reach
// the platform opener.
func TestRejectsNonHTTPSchemes(t *testing.T) {
	for _, url := range []string{"file:///etc/passwd", "javascript:alert(1)", "ftp://x", ""} {
		if err := (SystemOpener{}).OpenURL(url); err == nil {
			t.Errorf("OpenURL(%q) accepted a non-http(s) scheme", url)
		}
	}
}

// TestOpenerBinaryIsAbsolute guards the fixed-directory rule: the resolved opener
// must never be a bare name the OS would search $PATH for.
func TestOpenerBinaryIsAbsolute(t *testing.T) {
	opener, err := openerBinary()
	if err != nil {
		t.Skipf("no opener on this machine: %v", err)
	}
	if !filepath.IsAbs(opener) {
		t.Errorf("openerBinary() = %q, want an absolute path (never a $PATH lookup)", opener)
	}
}
