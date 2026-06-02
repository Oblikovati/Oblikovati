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

func TestOpenNonDocumentYAMLErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.obk")
	// A bare scalar is valid YAML but not a document mapping — must be rejected.
	if err := os.WriteFile(path, []byte("this is not a document"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := OpenPackage(path); err == nil {
		t.Error("OpenPackage of a non-document YAML file returned no error")
	}
}

func TestOpenLegacyZipRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy-zip.obk")
	// "PK\x03\x04" is the ZIP local-file-header magic — a pre-ADR-0020 package.
	if err := os.WriteFile(path, []byte("PK\x03\x04rest-of-a-zip"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := OpenPackage(path); err == nil {
		t.Error("OpenPackage accepted a legacy ZIP .obk; it should reject pre-YAML packages")
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

func TestOpenMalformedYAMLErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.obk")
	// Unbalanced brackets — not parseable as YAML at all.
	if err := os.WriteFile(path, []byte("schemaVersion: 2\nmodel: [unclosed"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := OpenPackage(path); err == nil {
		t.Error("OpenPackage accepted malformed YAML")
	}
}
