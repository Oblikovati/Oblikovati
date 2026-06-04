// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/Oblikovati/oblikovati/persistence/yamlcodec"
)

// catalogFiles holds the shipped, read-only material catalog: one YAML file per category
// (metals, steels, aluminium, plastics, woods, composites), numbered so they load in a
// stable display order. Keeping the catalog as data (ADR-0024) rather than Go literals
// lets the ~100-entry library stay reviewable and under the file-size limit.
//
//go:embed catalog/*.yaml
var catalogFiles embed.FS

// loadCatalog parses every embedded catalog file in sorted order and folds its
// appearances and materials into l as built-in (read-only) assets. The files ship with
// the binary, so a parse error or malformed colour is a build defect; it is returned for
// the caller (seedBuiltins) to turn into a panic the package tests catch — never a
// runtime condition.
func loadCatalog(l *Library) error {
	names, err := fs.Glob(catalogFiles, "catalog/*.yaml")
	if err != nil {
		return fmt.Errorf("material: list catalog: %w", err)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := loadCatalogFile(l, name); err != nil {
			return err
		}
	}
	return nil
}

// loadCatalogFile reads one catalog file and adds its appearances and materials to l as
// built-ins, naming the offending file and asset id on any malformed colour.
func loadCatalogFile(l *Library, name string) error {
	data, err := catalogFiles.ReadFile(name)
	if err != nil {
		return fmt.Errorf("material: read catalog %q: %w", name, err)
	}
	var rd RecipeData
	if err := yamlcodec.Unmarshal(data, &rd); err != nil {
		return fmt.Errorf("material: parse catalog %q: %w", name, err)
	}
	for _, ar := range rd.Appearances {
		a, err := recipeToAppearance(ar, SourceBuiltin)
		if err != nil {
			return fmt.Errorf("material: catalog %q appearance %q: %w", name, ar.ID, err)
		}
		l.AddAppearance(a)
	}
	for _, mr := range rd.Materials {
		l.AddMaterial(recipeToMaterial(mr, SourceBuiltin))
	}
	return nil
}
