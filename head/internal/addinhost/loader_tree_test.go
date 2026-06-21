//go:build linux || darwin

// SPDX-License-Identifier: GPL-2.0-only

package addinhost

import (
	"os"
	"path/filepath"
	"testing"
)

// copyFile duplicates src to dst (used to place a built fixture into an install tree).
func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, b, 0o755); err != nil {
		t.Fatal(err)
	}
}

// TestLoadInstalledTreeNested proves the per-user install layout loads: a real c-shared
// add-in placed under <root>/<name>/<subfolder>/ (as the installer extracts a bundle) is
// found and loaded, with its id resolved over the C ABI.
func TestLoadInstalledTreeNested(t *testing.T) {
	so := buildFixture(t, "echoaddin")
	root := t.TempDir()
	nested := filepath.Join(root, "com.oblikovati.echo-fixture", "bundle")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	copyFile(t, so, filepath.Join(nested, filepath.Base(so)))

	libs, skipped, err := LoadInstalledTree(root)
	if err != nil {
		t.Fatalf("LoadInstalledTree: %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("unexpected skips: %+v", skipped)
	}
	if len(libs) != 1 {
		t.Fatalf("loaded %d add-ins, want 1", len(libs))
	}
	defer func() { _ = libs[0].Close() }()
	if got := libs[0].ID(); got != "com.oblikovati.echo-fixture" {
		t.Errorf("ID = %q, want com.oblikovati.echo-fixture", got)
	}
}

func TestLoadInstalledTreeMissingDirIsEmpty(t *testing.T) {
	libs, skipped, err := LoadInstalledTree(filepath.Join(t.TempDir(), "absent"))
	if err != nil || libs != nil || skipped != nil {
		t.Errorf("missing dir = (%v,%v,%v), want all nil", libs, skipped, err)
	}
}

func TestFirstSharedLib(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "lib.so"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, ok := firstSharedLib(root)
	if !ok || filepath.Base(got) != "lib.so" {
		t.Errorf("firstSharedLib = (%q,%v), want a nested lib.so", got, ok)
	}
	if _, ok := firstSharedLib(t.TempDir()); ok {
		t.Error("firstSharedLib found a library in an empty dir")
	}
}
