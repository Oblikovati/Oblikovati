// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestColorStyleCRUD drives list -> set(add) -> get -> set(update) -> delete.
func TestColorStyleCRUD(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)

	var list wire.ColorStylesResult
	call(t, r, s, "styles.list", `{}`, &list)
	if len(list.Styles) < 3 {
		t.Fatalf("built-in styles = %d, want >= 3", len(list.Styles))
	}

	glass := wire.ColorStyleView{Name: "Glass", Diffuse: types.NewColor(200, 220, 255), Opacity: 0.3, Location: types.LocalStyleLocation}
	var added wire.ColorStyleView
	call(t, r, s, "styles.set", mustJSON(t, glass), &added)
	if added.Name != "Glass" || added.Opacity != 0.3 {
		t.Errorf("added = %+v, want Glass opacity 0.3", added)
	}

	var got wire.ColorStyleView
	call(t, r, s, "styles.get", mustJSON(t, wire.GetStyleArgs{Name: "Glass"}), &got)
	if got.Diffuse.B != 255 {
		t.Errorf("Glass diffuse.B = %d, want 255", got.Diffuse.B)
	}

	glass.Opacity = 0.6
	call(t, r, s, "styles.set", mustJSON(t, glass), &wire.ColorStyleView{})
	call(t, r, s, "styles.get", mustJSON(t, wire.GetStyleArgs{Name: "Glass"}), &got)
	if got.Opacity != 0.6 {
		t.Errorf("Glass opacity after update = %v, want 0.6", got.Opacity)
	}

	call(t, r, s, "styles.delete", mustJSON(t, wire.GetStyleArgs{Name: "Glass"}), &wire.OKResult{})
	if err := tryCall(t, r, s, "styles.get", `{"name":"Glass"}`); err == nil {
		t.Error("getting a deleted style should error")
	}
}

// TestImportStyleLibrary writes a JSON library file, imports it, and checks it joins the
// cascade and merges a new style.
func TestImportStyleLibrary(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	path := filepath.Join(t.TempDir(), "metals.json")
	body := `{"name":"Metals","styles":[{"name":"Titanium","diffuse":{"r":180,"g":184,"b":190,"opacity":1,"source":79105},"opacity":1,"shininess":0.6}]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var libs wire.StyleLibrariesResult
	call(t, r, s, "styles.importLibrary", mustJSON(t, wire.ImportStyleLibraryArgs{Path: path}), &libs)
	if len(libs.Libraries) != 1 || libs.Libraries[0].Name != "Metals" {
		t.Fatalf("libraries = %+v, want one named Metals", libs.Libraries)
	}

	var got wire.ColorStyleView
	call(t, r, s, "styles.get", mustJSON(t, wire.GetStyleArgs{Name: "Titanium"}), &got)
	if got.Location != types.LibraryStyleLocation {
		t.Errorf("Titanium location = %v, want Library", got.Location)
	}

	if err := tryCall(t, r, s, "styles.importLibrary", `{"path":"/no/such/file.json"}`); err == nil {
		t.Error("importing a missing file should error")
	}
}
