// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"strings"

	"oblikovati.org/model/doc"
)

// Document-scoped save operations (M03-F09, #610): the generalized save flow
// behind File ▸ Save, the documents.save/saveAs/saveCopyAs wire methods and
// batch save. The active-document wrappers in session.go delegate here.

// SaveDocument writes any open document back to its path, honoring the save
// policy: with SaveDependents on, its dirty referenced documents save first.
func (s *Session) SaveDocument(d *doc.Document) error {
	if d == nil {
		return ErrNoActiveDoc
	}
	if !strings.HasSuffix(d.FullFileName(), doc.PackageExtension) {
		return ErrNeedsPath
	}
	if s.appOptions.Save.SaveDependents {
		if err := s.saveDirtyDependents(d); err != nil {
			return err
		}
	}
	s.collectFileMetadata(d) // the PopulateFileMetadata hook gathers file properties around the save
	if err := s.workspace.Save(d); err != nil {
		return err
	}
	s.saveViewState(d) // persist this user's camera/view layout alongside (not inside) the document
	return nil
}

// saveDirtyDependents saves d's dirty referenced documents before d itself, so
// one Save leaves the whole tree consistent (the SaveDependents policy).
func (s *Session) saveDirtyDependents(d *doc.Document) error {
	for _, dep := range d.AllReferencedDocuments() {
		if !dep.Dirty() || !strings.HasSuffix(dep.FullFileName(), doc.PackageExtension) {
			continue
		}
		if err := s.workspace.Save(dep); err != nil {
			return fmt.Errorf("app: save dependent %q: %w", dep.FullFileName(), err)
		}
	}
	return nil
}

// SaveDocumentAs writes any open document under a new full document name,
// which becomes its identity.
func (s *Session) SaveDocumentAs(d *doc.Document, path string) error {
	if d == nil {
		return ErrNoActiveDoc
	}
	s.collectFileMetadata(d)
	if err := s.workspace.SaveAs(d, path); err != nil {
		return err
	}
	s.saveViewState(d)
	s.rememberRecentDocument(path)
	return nil
}

// SaveDocumentCopyAs writes a copy of any open document to targetFileName
// without retargeting it — the document keeps its binding and dirty state.
func (s *Session) SaveDocumentCopyAs(d *doc.Document, targetFileName string, meta doc.CopyMetadata) error {
	if d == nil {
		return ErrNoActiveDoc
	}
	return s.workspace.SaveCopy(d, targetFileName, meta)
}
