// SPDX-License-Identifier: GPL-2.0-only

package doc

// The file as an object distinct from the documents it contains (M03-F07,
// #608). Today one .obk hosts exactly one document, so a File is a view over
// that document's identity and reference records; the shape already allows
// several contained documents (model states and library members later).

// File is one document file: identity, load state, and its reference records.
type File struct {
	d *Document
}

// FileByName returns the file view of an open (or stub-registered) document
// file, ok=false when no document with that file name is in the workspace.
func (ws *Workspace) FileByName(fullFileName string) (*File, bool) {
	d, ok := ws.byName[fullFileName]
	if !ok {
		return nil, false
	}
	return &File{d: d}, true
}

// File returns the file hosting this document.
func (d *Document) File() *File { return &File{d: d} }

// FullFileName returns the file's on-disk name.
func (f *File) FullFileName() string { return f.d.FullFileName() }

// InternalName returns the stable identity GUID minted at creation.
func (f *File) InternalName() string { return f.d.identity.InternalName }

// RevisionID returns the GUID stamping the content as of the last save.
func (f *File) RevisionID() string { return f.d.identity.RevisionID }

// DatabaseRevisionID returns the GUID stamping only the model content.
func (f *File) DatabaseRevisionID() string { return f.d.identity.DatabaseRevisionID }

// FileSaveCounter returns how many times the file has been saved.
func (f *File) FileSaveCounter() int { return f.d.identity.SaveCounter }

// VersionCreated returns the software version that created the file.
func (f *File) VersionCreated() string { return f.d.identity.VersionCreated }

// VersionSaved returns the software version that last saved the file, "" for
// a never-saved file.
func (f *File) VersionSaved() string { return f.d.identity.VersionSaved }

// Loaded reports whether any document within this file is paged in.
func (f *File) Loaded() bool { return f.d.Open() }

// Referenced reports whether any other in-memory file references this one.
func (f *File) Referenced() bool { return f.d.Referenced() }

// Documents returns the contained documents that are available — today the
// hosting document alone; model states and members widen this later.
func (f *File) Documents() []*Document { return []*Document{f.d} }

// References returns the file's persisted file-to-file reference records.
func (f *File) References() []*FileReference { return f.d.FileReferences() }
