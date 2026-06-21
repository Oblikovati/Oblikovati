// SPDX-License-Identifier: GPL-2.0-only

package userpaths

import (
	"path/filepath"
	"testing"
)

func TestOblikovatiHomeEndsInOblikovati(t *testing.T) {
	home, err := OblikovatiHome()
	if err != nil {
		t.Fatalf("OblikovatiHome: %v", err)
	}
	if filepath.Base(home) != "oblikovati" {
		t.Errorf("OblikovatiHome() = %q, want a path ending in /oblikovati", home)
	}
	if !filepath.IsAbs(home) {
		t.Errorf("OblikovatiHome() = %q, want an absolute path", home)
	}
}
