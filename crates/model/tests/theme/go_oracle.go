//go:build ignore

// SPDX-License-Identifier: GPL-2.0-only
// Go aa8297d1 is the behavior oracle; this helper never writes user preferences.
package main

import (
	"encoding/json"
	"oblikovati.org/api/types"
	"oblikovati.org/theme"
	"os"
)

type themeRecord struct {
	Name     string                          `json:"name"`
	Kind     types.ThemeKind                 `json:"kind"`
	Colors   map[string]string               `json:"colors"`
	Channels map[types.ThemeToken]types.Rgba `json:"channels"`
}
type libraryRecord struct {
	Stage    string            `json:"stage"`
	Active   string            `json:"active"`
	Revision uint64            `json:"revision"`
	Names    []string          `json:"names"`
	Colors   map[string]string `json:"colors"`
}
type themeOracle struct {
	ApplicationRevision string             `json:"applicationRevision"`
	Tokens              []types.ThemeToken `json:"tokens"`
	Builtins            []themeRecord      `json:"builtins"`
	Library             []libraryRecord    `json:"library"`
}

func snapshotTheme(t *theme.Theme) themeRecord {
	return themeRecord{t.Name(), t.Kind(), t.Palette().Hex(), t.Palette().Clone()}
}
func recordLibrary(out *[]libraryRecord, lib *theme.Library, stage string) {
	names := make([]string, 0, len(lib.Themes()))
	for _, item := range lib.Themes() {
		names = append(names, item.Name())
	}
	*out = append(*out, libraryRecord{stage, lib.ActiveName(), lib.Revision(), names, lib.Active().Palette().Hex()})
}
func selectStages(out *[]libraryRecord, lib *theme.Library) {
	recordLibrary(out, lib, "unknown preference falls back")
	lib.SetActive("Light")
	recordLibrary(out, lib, "select light")
	lib.SetActive("Light")
	recordLibrary(out, lib, "select same")
	lib.SetActive("Missing")
	recordLibrary(out, lib, "select missing")
	lib.EditActiveColor(types.TokenChromeAccent, types.Rgba{R: .1, G: .2, B: .3, A: .4})
	recordLibrary(out, lib, "edit built-in")
}
func customStages(out *[]libraryRecord, lib *theme.Library) {
	if _, err := lib.Duplicate("Light", "Workshop"); err != nil {
		panic(err)
	}
	recordLibrary(out, lib, "duplicate light")
	lib.EditActiveColor(types.TokenChromeAccent, types.Rgba{R: .1, G: .2, B: .3, A: .4})
	recordLibrary(out, lib, "edit direct")
	tint := types.Rgba{R: .9, G: .8, B: .7, A: .6}
	lib.EditActiveColor(types.TokenChromeButtonHover, tint)
	recordLibrary(out, lib, "edit derived")
	lib.EditActiveColor(types.TokenChromeButtonHover, tint)
	recordLibrary(out, lib, "edit same")
	if _, err := lib.Duplicate("Dark", "Second"); err != nil {
		panic(err)
	}
	recordLibrary(out, lib, "duplicate dark")
}
func removeStages(out *[]libraryRecord, lib *theme.Library) {
	if err := lib.Remove("Workshop"); err != nil {
		panic(err)
	}
	recordLibrary(out, lib, "remove before active")
	if err := lib.Remove("Second"); err != nil {
		panic(err)
	}
	recordLibrary(out, lib, "remove active")
}
func main() {
	result := themeOracle{ApplicationRevision: "aa8297d1b0ac33c024f19e54821536d76e8a447a", Tokens: types.AllThemeTokens()}
	for _, builtin := range theme.Builtins() {
		result.Builtins = append(result.Builtins, snapshotTheme(builtin))
	}
	library := theme.NewLibrary(nil, "Missing")
	selectStages(&result.Library, library)
	customStages(&result.Library, library)
	removeStages(&result.Library, library)
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		panic(err)
	}
}
