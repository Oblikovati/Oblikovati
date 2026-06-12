// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"encoding/base64"
	"fmt"
	"os"

	"oblikovati.org/model/doc"
	"oblikovati.org/persistence/yamlcodec"
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
		SubType:       string(d.SubType()),
		DisplayName:   d.DisplayName(),
	}
	if err := pkg.SetManifest(manifest); err != nil {
		return err
	}
	pkg.SetIdentity(toCodecIdentity(d.FileIdentity()))
	pkg.SetFileReferences(toCodecFileReferences(d.FileReferenceRecords()))
	pkg.SetAttachments(toCodecAttachments(d.AttachmentRecords()))
	if rc, ok := d.Content().(doc.RecipeContent); ok {
		model, err := rc.MarshalRecipe()
		if err != nil {
			return fmt.Errorf("persistence: save %q: marshal recipe: %w", d.FullFileName(), err)
		}
		pkg.SetModelYAML(model)
	}
	if rb, ok := d.Content().(doc.ResourceBearer); ok {
		pkg.SetResources(toCodecResources(rb.Resources()))
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
	d, err := doc.Restore(doc.DocumentType(manifest.DocumentType), fullDocumentName, manifest.DisplayName)
	if err != nil {
		return nil, err
	}
	d.SetSubType(doc.SubTypeID(manifest.SubType)) // restore the flavor (M05-F15)
	if err := restoreIdentity(d, pkg); err != nil {
		return nil, fmt.Errorf("persistence: load %q: %w", fullDocumentName, err)
	}
	// Resources must be restored BEFORE the recipe is applied, so a feature that re-derives
	// geometry from an embedded resource (e.g. an imported body) can read its bytes (ADR-0031).
	if rb, ok := d.Content().(doc.ResourceBearer); ok {
		rb.SetResources(fromCodecResources(pkg.Resources()))
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
	return d, nil
}

// restoreIdentity puts the persisted identity block, reference records and
// attachments back on the document (M03-F07/F08). A pre-identity file keeps
// the freshly minted identity, persisting it on its next save.
func restoreIdentity(d *doc.Document, pkg *Package) error {
	if id := pkg.Identity(); id != nil {
		d.SetFileIdentity(doc.FileIdentity{
			InternalName: id.InternalName, RevisionID: id.RevisionID,
			DatabaseRevisionID: id.DatabaseRevisionID, SaveCounter: id.SaveCounter,
			VersionCreated: id.VersionCreated, VersionSaved: id.VersionSaved,
			ModelDigest: id.ModelDigest,
		})
	}
	d.SetFileReferenceRecords(fromCodecFileReferences(pkg.FileReferences()))
	recs, err := fromCodecAttachments(pkg.Attachments())
	if err != nil {
		return err
	}
	d.SetAttachmentRecords(recs)
	return nil
}

// toCodecAttachments / fromCodecAttachments bridge attachment records across
// the doc/persistence seam; the embedded payload travels base64 on disk (the
// same concession as data sections, ADR-0020).
func toCodecAttachments(in []doc.FileAttachmentRecord) []yamlcodec.AttachmentRecord {
	out := make([]yamlcodec.AttachmentRecord, len(in))
	for i, a := range in {
		out[i] = yamlcodec.AttachmentRecord{
			Name: a.Name, Kind: a.Kind, FullFileName: a.FullFileName,
			ResourceID: a.ResourceID, Payload: base64.StdEncoding.EncodeToString(a.Payload),
			LastKnownFileTime: a.LastKnownFileTime, BrowserVisible: a.BrowserVisible,
		}
	}
	return out
}

func fromCodecAttachments(in []yamlcodec.AttachmentRecord) ([]doc.FileAttachmentRecord, error) {
	out := make([]doc.FileAttachmentRecord, len(in))
	for i, a := range in {
		payload, err := base64.StdEncoding.DecodeString(a.Payload)
		if err != nil {
			return nil, fmt.Errorf("persistence: attachment %q payload is not valid base64: %w", a.Name, err)
		}
		out[i] = doc.FileAttachmentRecord{
			Name: a.Name, Kind: a.Kind, FullFileName: a.FullFileName,
			ResourceID: a.ResourceID, Payload: payload,
			LastKnownFileTime: a.LastKnownFileTime, BrowserVisible: a.BrowserVisible,
		}
	}
	return out, nil
}

// toCodecIdentity renders the document identity for the on-disk shape.
func toCodecIdentity(id doc.FileIdentity) *yamlcodec.FileIdentityRecord {
	return &yamlcodec.FileIdentityRecord{
		InternalName: id.InternalName, RevisionID: id.RevisionID,
		DatabaseRevisionID: id.DatabaseRevisionID, SaveCounter: id.SaveCounter,
		VersionCreated: id.VersionCreated, VersionSaved: id.VersionSaved,
		ModelDigest: id.ModelDigest,
	}
}

// toCodecFileReferences / fromCodecFileReferences bridge the reference records
// across the doc/persistence seam (fields identical, layers decoupled).
func toCodecFileReferences(in []doc.FileReferenceRecord) []yamlcodec.FileReferenceRecord {
	out := make([]yamlcodec.FileReferenceRecord, len(in))
	for i, r := range in {
		out[i] = yamlcodec.FileReferenceRecord{
			FullFileName: r.FullFileName, RelativeFileName: r.RelativeFileName,
			LibraryName: r.LibraryName, LocationType: r.LocationType,
			ReferencedInternalName: r.ReferencedInternalName, SaveCounter: r.SaveCounter,
		}
	}
	return out
}

func fromCodecFileReferences(in []yamlcodec.FileReferenceRecord) []doc.FileReferenceRecord {
	out := make([]doc.FileReferenceRecord, len(in))
	for i, r := range in {
		out[i] = doc.FileReferenceRecord{
			FullFileName: r.FullFileName, RelativeFileName: r.RelativeFileName,
			LibraryName: r.LibraryName, LocationType: r.LocationType,
			ReferencedInternalName: r.ReferencedInternalName, SaveCounter: r.SaveCounter,
		}
	}
	return out
}

// toCodecResources / fromCodecResources bridge the document-layer resource table (doc.Resource)
// and the on-disk serialization type (yamlcodec.Resource); the fields are identical, this keeps
// the layers decoupled (persistence is the only package that knows both).
func toCodecResources(in map[string]doc.Resource) map[string]yamlcodec.Resource {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]yamlcodec.Resource, len(in))
	for id, r := range in {
		out[id] = yamlcodec.Resource{Type: r.Type, Encoding: r.Encoding, Value: r.Value, Origin: r.Origin}
	}
	return out
}

func fromCodecResources(in map[string]yamlcodec.Resource) map[string]doc.Resource {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]doc.Resource, len(in))
	for id, r := range in {
		out[id] = doc.Resource{Type: r.Type, Encoding: r.Encoding, Value: r.Value, Origin: r.Origin}
	}
	return out
}

// Exists reports whether a package file is present at fullDocumentName.
func (s *PackageStore) Exists(fullDocumentName string) bool {
	info, err := os.Stat(fullDocumentName)
	return err == nil && !info.IsDir()
}
