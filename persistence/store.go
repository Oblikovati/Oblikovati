// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"fmt"
	"os"

	"github.com/Oblikovati/oblikovati/model/doc"
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

// Save writes the document's manifest into a fresh package and saves it atomically
// at the document's file name.
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
	return doc.Restore(doc.DocumentType(manifest.DocumentType), fullDocumentName, manifest.DisplayName)
}

// Exists reports whether a package file is present at fullDocumentName.
func (s *PackageStore) Exists(fullDocumentName string) bool {
	info, err := os.Stat(fullDocumentName)
	return err == nil && !info.IsDir()
}
