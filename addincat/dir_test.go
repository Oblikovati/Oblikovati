// SPDX-License-Identifier: GPL-2.0-only

package addincat

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestUserAddInsDirEnvOverride(t *testing.T) {
	t.Setenv("OBK_USER_ADDINS_DIR", "/tmp/custom/addins")
	dir, err := UserAddInsDir()
	if err != nil {
		t.Fatalf("UserAddInsDir: %v", err)
	}
	if dir != "/tmp/custom/addins" {
		t.Errorf("dir = %q, want the override", dir)
	}
}

func TestUserAddInsDirDefaultEndsInAddins(t *testing.T) {
	t.Setenv("OBK_USER_ADDINS_DIR", "")
	dir, err := UserAddInsDir()
	if err != nil {
		t.Fatalf("UserAddInsDir: %v", err)
	}
	if filepath.Base(dir) != "addins" || filepath.Base(filepath.Dir(dir)) != "oblikovati" {
		t.Errorf("default dir = %q, want .../oblikovati/addins", dir)
	}
}

func TestPlatform(t *testing.T) {
	p := Platform()
	if !strings.Contains(p, "-") {
		t.Errorf("Platform() = %q, want GOOS-GOARCH", p)
	}
}
