// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenMissingFileErrors(t *testing.T) {
	if _, err := OpenPackage(filepath.Join(t.TempDir(), "nope.obk")); err == nil {
		t.Error("OpenPackage of a missing file returned no error")
	}
	if _, err := ReadDataFromFile(filepath.Join(t.TempDir(), "nope.obk"), "s"); err == nil {
		t.Error("ReadDataFromFile of a missing file returned no error")
	}
}

func TestOpenCorruptArchiveErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.obk")
	if err := os.WriteFile(path, []byte("this is not a zip archive"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := OpenPackage(path); err == nil {
		t.Error("OpenPackage of a non-zip file returned no error")
	}
}

func TestSaveToMissingDirectoryErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-dir", "p.obk")
	if err := NewPackage().Save(path); err == nil {
		t.Error("Save into a nonexistent directory returned no error")
	}
	// WriteDataToFile funnels through the same atomic write and must also fail.
	if err := WriteDataToFile(path, "s", []byte("x")); err == nil {
		t.Error("WriteDataToFile into a nonexistent directory returned no error")
	}
}

func TestBadManifestJSONErrors(t *testing.T) {
	p := NewPackage()
	p.WriteStream(manifestStream, []byte("{ not valid json"))
	if _, err := p.Manifest(); err == nil {
		t.Error("Manifest accepted invalid JSON")
	}
	if err := Migrate(p); err == nil {
		t.Error("Migrate accepted a package with an invalid manifest")
	}
}
