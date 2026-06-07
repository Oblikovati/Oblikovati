// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"fmt"
	"os"

	"oblikovati/api/types"
	"oblikovati/math"
	"oblikovati/model/doc"
	"oblikovati/persistence/yamlcodec"
)

// PackageStore is the [doc.Store] backed by .obk packages on disk. Injected into a
// doc.Workspace, it makes save/open real file operations: each document is one
// package whose manifest carries the document kind and display name. The richer
// model streams (parameters, features, sketches) join the manifest once that data
// exists on the content objects (M07+); today the manifest is the persisted recipe.
type PackageStore struct{}

// NewPackageStore returns a store that reads and writes .obk packages.
func NewPackageStore() *PackageStore {
	return &PackageStore{}
}

var _ doc.Store = (*PackageStore)(nil)

// Save writes the document's manifest and — when its content carries a recipe — the
// model recipe into a fresh package, and saves it atomically at the document's file
// name. The realized geometry is never written; it is recomputed on open (ADR-0020).
func (s *PackageStore) Save(d *doc.Document) error {
	pkg := NewPackage()
	manifest := Manifest{
		SchemaVersion: CurrentSchemaVersion,
		DocumentType:  uint32(d.DocumentType()),
		DisplayName:   d.DisplayName(),
	}
	if err := pkg.SetManifest(manifest); err != nil {
		return err
	}
	if rc, ok := d.Content().(doc.RecipeContent); ok {
		model, err := rc.MarshalRecipe()
		if err != nil {
			return fmt.Errorf("persistence: save %q: marshal recipe: %w", d.FullFileName(), err)
		}
		pkg.SetModelYAML(model)
	}
	pkg.SetViews(viewsSection(d))
	return pkg.Save(d.FullFileName())
}

// Load opens the package at fullDocumentName, migrates it, and reconstructs the
// document from its manifest.
func (s *PackageStore) Load(fullDocumentName string) (*doc.Document, error) {
	pkg, err := OpenPackage(fullDocumentName)
	if err != nil {
		return nil, err
	}
	manifest, err := pkg.Manifest()
	if err != nil {
		return nil, fmt.Errorf("persistence: load %q: %w", fullDocumentName, err)
	}
	d, err := doc.Restore(doc.DocumentType(manifest.DocumentType), fullDocumentName, manifest.DisplayName)
	if err != nil {
		return nil, err
	}
	if model, ok := pkg.ModelYAML(); ok {
		rc, ok := d.Content().(doc.RecipeContent)
		if !ok {
			return nil, fmt.Errorf("persistence: load %q: file has a model recipe but %v content cannot restore it (is its package imported?)", fullDocumentName, d.DocumentType())
		}
		if err := rc.ApplyRecipe(model); err != nil {
			return nil, fmt.Errorf("persistence: load %q: %w", fullDocumentName, err)
		}
	}
	restoreViews(d, pkg.Views())
	return d, nil
}

// viewsSection projects a document's view collection (cameras) onto the persisted section.
func viewsSection(d *doc.Document) *yamlcodec.ViewsSection {
	vs := d.Views()
	sec := &yamlcodec.ViewsSection{Active: vs.ActiveIndex(), Layout: int32(vs.Layout())}
	for _, v := range vs.All() {
		sec.Views = append(sec.Views, yamlcodec.ViewFrame{
			Name:   v.Name,
			Eye:    [3]float64{v.Eye.X, v.Eye.Y, v.Eye.Z},
			Target: [3]float64{v.Target.X, v.Target.Y, v.Target.Z},
			Up:     [3]float64{v.Up.X, v.Up.Y, v.Up.Z},
			FOV:    v.FOV,
		})
	}
	return sec
}

// restoreViews rebuilds a document's view collection from the persisted section (each
// loaded view is framed). A nil/empty section leaves the lazily-seeded default view.
func restoreViews(d *doc.Document, sec *yamlcodec.ViewsSection) {
	if sec == nil || len(sec.Views) == 0 {
		return
	}
	views := make([]*doc.View, len(sec.Views))
	for i, f := range sec.Views {
		views[i] = &doc.View{
			Name:   f.Name,
			Eye:    math.P3(f.Eye[0], f.Eye[1], f.Eye[2]),
			Target: math.P3(f.Target[0], f.Target[1], f.Target[2]),
			Up:     math.V3(f.Up[0], f.Up[1], f.Up[2]),
			FOV:    f.FOV,
			Framed: true,
		}
	}
	d.RestoreViews(views, sec.Active, types.ViewLayout(sec.Layout))
}

// Exists reports whether a package file is present at fullDocumentName.
func (s *PackageStore) Exists(fullDocumentName string) bool {
	info, err := os.Stat(fullDocumentName)
	return err == nil && !info.IsDir()
}
