// SPDX-License-Identifier: GPL-2.0-only

package osfont

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati/model/text"
)

// TestScanDirsEnumeratesFonts checks ScanDirs finds .ttf/.otf files recursively, reads each
// font's real family from its name table, and skips non-font files.
func TestScanDirsEnumeratesFonts(t *testing.T) {
	data, ok := text.EmbeddedFontBytes(text.DefaultFontFamily)
	if !ok {
		t.Fatal("no embedded font bytes to build the fixture from")
	}
	dir := t.TempDir()
	write(t, filepath.Join(dir, "face.ttf"), data)
	write(t, filepath.Join(dir, "notafont.txt"), []byte("not a font"))
	sub := filepath.Join(dir, "vendor")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(sub, "nested.otf"), data) // ext .otf, ttf content — sfnt parses by content

	faces := ScanDirs([]string{dir})
	if len(faces) != 2 {
		t.Fatalf("ScanDirs found %d faces, want 2 (the .txt skipped): %+v", len(faces), faces)
	}
	for _, f := range faces {
		if f.Family != text.DefaultFontFamily {
			t.Errorf("face family = %q, want %q", f.Family, text.DefaultFontFamily)
		}
		if f.Path == "" {
			t.Error("face has no path (cannot embed its bytes on select)")
		}
	}
}

// TestScanDirsMissingDirIsNotFatal confirms a non-existent directory is silently skipped.
func TestScanDirsMissingDirIsNotFatal(t *testing.T) {
	if faces := ScanDirs([]string{filepath.Join(t.TempDir(), "does-not-exist")}); len(faces) != 0 {
		t.Errorf("missing dir yielded %d faces, want 0", len(faces))
	}
}

func write(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
