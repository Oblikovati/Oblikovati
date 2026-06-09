// SPDX-License-Identifier: GPL-2.0-only

// Package osfont enumerates the TrueType/OpenType fonts installed on the host so the font
// picker can offer them alongside the application's bundled faces. It is the one place that
// touches the host font directories; the headless model never does (model/text stays pure),
// and a selected OS font is embedded into the document as bytes (ADR-0031), after which
// reopening no longer needs the host font at all.
package osfont

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"oblikovati/model/text"
)

// Face is one installed font face the picker can offer: its family + style and the file it
// was read from (so a selection can embed that file's bytes).
type Face struct {
	Family string
	Style  string
	Path   string
}

// System enumerates the installed faces under the platform's conventional font directories.
// Best-effort: missing directories and unparseable files are skipped, never fatal.
func System() []Face { return ScanDirs(DefaultDirs()) }

// DefaultDirs are the conventional system and per-user font directories on this host. The
// /run/host path covers a Flatpak/containerized run where the host tree is mounted there.
func DefaultDirs() []string {
	dirs := []string{"/usr/share/fonts", "/usr/local/share/fonts", "/run/host/usr/share/fonts"}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		dirs = append(dirs,
			filepath.Join(home, ".fonts"),
			filepath.Join(home, ".local", "share", "fonts"),
		)
	}
	return dirs
}

// ScanDirs walks each directory recursively for .ttf/.otf files and returns one Face per
// readable, parseable font (deduped by path), sorted by family then style. Parsing reads each
// file once to pull its real family/style from the name table — done when the picker opens.
func ScanDirs(dirs []string) []Face {
	var faces []Face
	seen := map[string]bool{}
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !isFontFile(path) || seen[path] {
				return nil
			}
			seen[path] = true
			if face, ok := faceAt(path); ok {
				faces = append(faces, face)
			}
			return nil
		})
	}
	sort.Slice(faces, func(i, j int) bool {
		if faces[i].Family != faces[j].Family {
			return faces[i].Family < faces[j].Family
		}
		return faces[i].Style < faces[j].Style
	})
	return faces
}

// ReadFont returns a font file's raw bytes — the bytes embedded into the document when its
// face is selected (ADR-0031).
func ReadFont(path string) ([]byte, error) { return os.ReadFile(path) }

func isFontFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ttf", ".otf":
		return true
	default:
		return false
	}
}

// faceAt parses one font file and reads its family/style; ok is false for an unreadable,
// unparseable, or family-less file (silently skipped during enumeration).
func faceAt(path string) (Face, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Face{}, false
	}
	ft, err := text.Parse(data)
	if err != nil {
		return Face{}, false
	}
	family := ft.Family()
	if family == "" {
		return Face{}, false
	}
	return Face{Family: family, Style: ft.Subfamily(), Path: path}, true
}
