// SPDX-License-Identifier: GPL-2.0-only

package userconfig

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDirIsDotOblikovatiOnUnix(t *testing.T) {
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if runtime.GOOS == "windows" {
		if filepath.Base(dir) != "oblikovati" {
			t.Errorf("windows dir base = %q, want oblikovati", filepath.Base(dir))
		}
		return
	}
	if filepath.Base(dir) != ".oblikovati" {
		t.Errorf("unix dir = %q, want it to end in .oblikovati", dir)
	}
	if !strings.HasPrefix(dir, "/") {
		t.Errorf("dir %q should be absolute", dir)
	}
}

func TestFileJoinsName(t *testing.T) {
	p, err := File("preferences.yaml")
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	if filepath.Base(p) != "preferences.yaml" {
		t.Errorf("File base = %q, want preferences.yaml", filepath.Base(p))
	}
}
