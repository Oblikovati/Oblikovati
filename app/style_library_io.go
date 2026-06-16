// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/model/style"
)

// fileStyleLibrary is the on-disk JSON shape of a style library: a name and its color styles.
// JSON keeps the loader on the standard library (the yaml dependency is reserved for the .obk
// document codec, CLAUDE.md). Color components use the [types.Color] JSON form.
type fileStyleLibrary struct {
	Name   string                 `json:"name"`
	Styles []fileStyleLibraryItem `json:"styles"`
}

type fileStyleLibraryItem struct {
	Name      string      `json:"name"`
	Diffuse   types.Color `json:"diffuse"`
	Ambient   types.Color `json:"ambient"`
	Specular  types.Color `json:"specular"`
	Emissive  types.Color `json:"emissive"`
	Shininess float64     `json:"shininess"`
	Opacity   float64     `json:"opacity"`
}

// loadStyleLibrary reads and parses a JSON style-library file, naming the path on any failure.
// The library name defaults to the file's base name (without extension) when the file omits it.
func loadStyleLibrary(path string) (style.Library, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return style.Library{}, fmt.Errorf("style library %q: %w", path, err)
	}
	var f fileStyleLibrary
	if err := json.Unmarshal(raw, &f); err != nil {
		return style.Library{}, fmt.Errorf("style library %q: bad JSON: %w", path, err)
	}
	name := f.Name
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	lib := style.Library{Name: name, Path: path}
	for _, it := range f.Styles {
		lib.Styles = append(lib.Styles, style.ColorStyle{
			Name: it.Name, Diffuse: it.Diffuse, Ambient: it.Ambient, Specular: it.Specular,
			Emissive: it.Emissive, Shininess: it.Shininess, Opacity: it.Opacity,
		})
	}
	return lib, nil
}
