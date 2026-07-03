// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/doc"
)

// The file surface (M03-F07, #608): identity, the persisted file-to-file
// reference records, and reference repair, served from the doc.File view.

// fileInfo marshals a file view into the wire DTO.
func fileInfo(f *doc.File) wire.FileInfo {
	docs := f.Documents()
	ids := make([]uint64, len(docs))
	for i, d := range docs {
		ids[i] = uint64(d.ID())
	}
	return wire.FileInfo{
		FullFileName: f.FullFileName(), InternalName: f.InternalName(),
		RevisionID: f.RevisionID(), DatabaseRevisionID: f.DatabaseRevisionID(),
		SaveCounter: f.FileSaveCounter(), VersionCreated: f.VersionCreated(),
		VersionSaved: f.VersionSaved(), Loaded: f.Loaded(), Referenced: f.Referenced(),
		Documents: ids,
	}
}

// fileReferenceInfo marshals one reference record into the file-side wire DTO.
func fileReferenceInfo(r *doc.FileReference) wire.FileReferenceInfo {
	return wire.FileReferenceInfo{
		FullFileName: r.FullFileName(), RelativeFileName: r.RelativeFileName(),
		LibraryName: r.LibraryName(), LocationType: r.LocationType(),
		ResolvedFileName: r.ResolvedFileName(), ReferencedInternalName: r.ReferencedFileInternalName(),
		SaveCounter: r.FileSaveCounter(), Status: r.Status(),
		Missing: r.ReferenceMissing(), Replaced: r.ReferenceReplaced(),
		LocationDifferent:     r.ReferenceLocationDifferent(),
		InternalNameDifferent: r.ReferenceInternalNameDifferent(),
	}
}

// documentByID resolves a wire document session id, shared by the document-
// scoped M03 handlers (file references, attachments, interests, save).
func documentByID(s *app.Session, id uint64) (*doc.Document, error) {
	for _, d := range s.Workspace().Documents() {
		if uint64(d.ID()) == id {
			return d, nil
		}
	}
	return nil, fmt.Errorf("router: no open document with id %d", id)
}

// openFile resolves the wire file name to the workspace's file view.
func openFile(s *app.Session, fullFileName string) (*doc.File, error) {
	f, ok := s.Workspace().FileByName(fullFileName)
	if !ok {
		return nil, fmt.Errorf("router: no open file named %q", fullFileName)
	}
	return f, nil
}

// getFile returns one open file's identity (wire files.get).
func getFile(s *app.Session, in wire.GetFileArgs) (wire.FileInfo, error) {
	f, err := openFile(s, in.FullFileName)
	if err != nil {
		return wire.FileInfo{}, err
	}
	return fileInfo(f), nil
}

// listFileReferences returns a file's persisted reference records (wire
// files.listReferences).
func listFileReferences(s *app.Session, in wire.GetFileArgs) (wire.ListFileReferencesResult, error) {
	f, err := openFile(s, in.FullFileName)
	if err != nil {
		return wire.ListFileReferencesResult{}, err
	}
	refs := f.References()
	out := make([]wire.FileReferenceInfo, len(refs))
	for i, r := range refs {
		out[i] = fileReferenceInfo(r)
	}
	return wire.ListFileReferencesResult{References: out}, nil
}

// replaceFileReference re-points one reference record at a new target (wire
// files.replaceReference) and returns the updated record.
func replaceFileReference(s *app.Session, in wire.ReplaceFileReferenceArgs) (wire.FileReferenceInfo, error) {
	f, err := openFile(s, in.FullFileName)
	if err != nil {
		return wire.FileReferenceInfo{}, err
	}
	for _, r := range f.References() {
		if r.FullFileName() != in.RequestedName {
			continue
		}
		if err := r.ReplaceReference(in.NewFileName); err != nil {
			return wire.FileReferenceInfo{}, err
		}
		return fileReferenceInfo(r), nil
	}
	return wire.FileReferenceInfo{}, fmt.Errorf("router: file %q holds no reference named %q", in.FullFileName, in.RequestedName)
}

// listDocumentFileReferences returns the document-side reference views (wire
// documents.listFileReferences).
func listDocumentFileReferences(s *app.Session, in wire.ListDocumentFileReferencesArgs) (wire.ListDocumentFileReferencesResult, error) {
	d, err := documentByID(s, in.Document)
	if err != nil {
		return wire.ListDocumentFileReferencesResult{}, err
	}
	refs := d.FileReferences()
	out := make([]wire.DocumentFileReferenceInfo, len(refs))
	for i, r := range refs {
		out[i] = wire.DocumentFileReferenceInfo{
			DisplayName: r.DisplayName(), FullFileName: r.FullFileName(),
			Status: r.Status(), DocumentFound: r.DocumentFound(),
			DifferentDocument: r.DifferentDocument(),
		}
	}
	return wire.ListDocumentFileReferencesResult{References: out}, nil
}
