// SPDX-License-Identifier: GPL-2.0-only

package icon

import (
	"embed"
	"io/fs"
)

// assetsFS embeds the ribbon glyph set. Each file is "<key>.svg"; the key is what a
// command's WithIcon names (e.g. "extrude" → assets/extrude.svg).
//
//go:embed assets/*.svg
var assetsFS embed.FS

// SVG returns the embedded SVG bytes for an icon key, or false if no such asset is
// bundled (the caller then falls back to a text button, never a blank icon).
func SVG(key string) ([]byte, bool) {
	b, err := assetsFS.ReadFile("assets/" + key + ".svg")
	if err != nil {
		return nil, false
	}
	return b, true
}

// Keys returns every bundled icon key (filename without the .svg extension), so a test
// can rasterize the whole set and a tool can list what is available.
func Keys() []string {
	entries, err := fs.ReadDir(assetsFS, "assets")
	if err != nil {
		return nil
	}
	keys := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		keys = append(keys, name[:len(name)-len(".svg")])
	}
	return keys
}
