// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// The session seams behind the file-UI event set (M04-F05, Oblikovati#613):
// the head consults the Hook* methods before presenting its file dialogs, the
// recent-files menu opens through OpenDocumentFromMRU, and the save paths
// collect metadata. Each seam emits Before (handlers answer or veto) and After
// (observers see the outcome) on the session bus.

// maxRecentDocuments bounds the recent-files list (the File ▸ Open Recent menu).
const maxRecentDocuments = 10

// HookFileOpenDialog announces File ▸ Open is about to present. A handler that
// supplies a path replaces the dialog: the head opens the returned path
// directly instead of showing it.
func (s *Session) HookFileOpenDialog() (string, bool) {
	return s.runDialogHook(FileOpenDialog{answer: new(string)})
}

// HookFileSaveAsDialog announces File ▸ Save As is about to present; a supplied
// path replaces the dialog.
func (s *Session) HookFileSaveAsDialog(saveCopyAs bool) (string, bool) {
	return s.runDialogHook(FileSaveAsDialog{SaveCopyAs: saveCopyAs, answer: new(string)})
}

// HookFileNewDialog announces a new-document template chooser is about to
// present; a supplied template replaces the dialog.
func (s *Session) HookFileNewDialog() (string, bool) {
	return s.runDialogHook(FileNewDialog{answer: new(string)})
}

// dialogHook is the shape the three dialog hooks share: an answer slot readable
// after emission.
type dialogHook interface {
	event.Event
	Supplied() string
}

// runDialogHook emits one dialog hook in both phases and returns the answer. A
// generic method (Go 1.27): the bus dispatches on the event's static type, so it
// needs its own type parameter rather than accepting the dialogHook interface.
func (s *Session) runDialogHook[E dialogHook](ev E) (string, bool) {
	s.bus.Emit(event.Before, ev)
	s.bus.Emit(event.After, ev)
	return ev.Supplied(), ev.Supplied() != ""
}

// RecentDocuments returns the recently opened/saved paths, most recent first —
// the File ▸ Open Recent menu.
func (s *Session) RecentDocuments() []string { return s.recentDocuments }

// OpenDocumentFromMRU opens a path picked from the recent-files menu, behind
// the vetoable FileOpenFromMRU hook (a vault add-in knows the file moved).
func (s *Session) OpenDocumentFromMRU(path string) (*doc.Document, error) {
	ev := FileOpenFromMRU{FullDocumentName: path}
	if out := event.Emit(s.bus, event.Before, ev); out.Vetoed() {
		s.notice = out.Reason
		return nil, &doc.VetoError{Operation: "open recent", Reason: out.Reason}
	}
	d, err := s.OpenDocument(path)
	if err != nil {
		return nil, err
	}
	event.Emit(s.bus, event.After, ev)
	return d, nil
}

// rememberRecentDocument moves path to the front of the recent list, bounded —
// called by the open/save-as paths so the menu tracks real file activity.
func (s *Session) rememberRecentDocument(path string) {
	out := []string{path}
	for _, p := range s.recentDocuments {
		if p != path && len(out) < maxRecentDocuments {
			out = append(out, p)
		}
	}
	s.recentDocuments = out
}

// collectFileMetadata runs the PopulateFileMetadata hook for a document about
// to save and remembers what handlers contributed.
func (s *Session) collectFileMetadata(d *doc.Document) {
	ev := PopulateFileMetadata{FullDocumentName: d.FullDocumentName(), entries: &[]FileMetadataValue{}}
	event.Emit(s.bus, event.Before, ev)
	event.Emit(s.bus, event.After, ev)
	if s.fileMetadata == nil {
		s.fileMetadata = map[doc.ID][]FileMetadataValue{}
	}
	s.fileMetadata[d.ID()] = ev.Entries()
}

// FileMetadata returns the entries the PopulateFileMetadata hook collected at
// the document's last save (nil when it never saved this session).
func (s *Session) FileMetadata(id doc.ID) []FileMetadataValue { return s.fileMetadata[id] }
