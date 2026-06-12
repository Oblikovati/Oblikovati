// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// The file-UI event set (M04-F05, Oblikovati#613) on the session bus: the
// new/open/save-as flows and their dialogs, so an add-in can veto, observe, or
// pre-seed them (a PDM add-in replaces the file dialogs with its vault picker).
// Stable TypeIDs continue the session-bus block; never renumber.
const (
	tidFileNew              event.TypeID = 0x0518
	tidFileNewDialog        event.TypeID = 0x0519
	tidFileOpenDialog       event.TypeID = 0x051a
	tidFileSaveAsDialog     event.TypeID = 0x051b
	tidFileOpenFromMRU      event.TypeID = 0x051c
	tidPopulateFileMetadata event.TypeID = 0x051d
)

// FileNew fires around creating a new document; a Before handler may veto
// (e.g. enforce a vault checkout-first policy).
type FileNew struct{ DocumentType doc.DocumentType }

// EventID implements event.Event.
func (FileNew) EventID() event.TypeID { return tidFileNew }

// FileOpenDialog fires (Before) when the head is about to present File ▸ Open;
// a handler that supplies a path replaces the dialog — the head opens the
// supplied file directly. The After phase reports the outcome to observers.
type FileOpenDialog struct{ answer *string }

// EventID implements event.Event.
func (FileOpenDialog) EventID() event.TypeID { return tidFileOpenDialog }

// Supply answers the hook with the path to use instead of showing the dialog.
// The first non-empty answer sticks (events travel by value; the slot is shared).
func (e FileOpenDialog) Supply(path string) { supplyAnswer(e.answer, path) }

// Supplied returns the path a handler supplied, "" when none did.
func (e FileOpenDialog) Supplied() string { return *e.answer }

// FileSaveAsDialog fires (Before) when the head is about to present File ▸
// Save As; a handler that supplies a path replaces the dialog. SaveCopyAs
// distinguishes the save-a-copy variant when it arrives.
type FileSaveAsDialog struct {
	SaveCopyAs bool
	answer     *string
}

// EventID implements event.Event.
func (FileSaveAsDialog) EventID() event.TypeID { return tidFileSaveAsDialog }

// Supply answers the hook with the destination path to use instead of asking.
func (e FileSaveAsDialog) Supply(path string) { supplyAnswer(e.answer, path) }

// Supplied returns the path a handler supplied, "" when none did.
func (e FileSaveAsDialog) Supplied() string { return *e.answer }

// FileNewDialog fires (Before) when the head is about to present a new-document
// template chooser; a handler may supply the template to use instead. The head
// has no template dialog yet — the seam exists so add-ins written against it
// keep working when one arrives, and wire observers see the flow today.
type FileNewDialog struct{ answer *string }

// EventID implements event.Event.
func (FileNewDialog) EventID() event.TypeID { return tidFileNewDialog }

// Supply answers the hook with the template file to use instead of asking.
func (e FileNewDialog) Supply(templateFile string) { supplyAnswer(e.answer, templateFile) }

// Supplied returns the template a handler supplied, "" when none did.
func (e FileNewDialog) Supplied() string { return *e.answer }

// FileOpenFromMRU fires around opening a document from the recent-files list;
// a Before handler may veto (e.g. the vault knows the file moved).
type FileOpenFromMRU struct{ FullDocumentName string }

// EventID implements event.Event.
func (FileOpenFromMRU) EventID() event.TypeID { return tidFileOpenFromMRU }

// FileMetadataValue is one name/value pair collected around a save.
type FileMetadataValue struct{ Name, Value string }

// PopulateFileMetadata fires (Before) as a document is about to save, letting
// handlers contribute file properties (author, revision, vault state…); the
// collected entries are queryable via [Session.FileMetadata].
type PopulateFileMetadata struct {
	FullDocumentName string
	entries          *[]FileMetadataValue
}

// EventID implements event.Event.
func (PopulateFileMetadata) EventID() event.TypeID { return tidPopulateFileMetadata }

// Add contributes one metadata entry (every handler's copy shares the slice).
func (e PopulateFileMetadata) Add(name, value string) {
	*e.entries = append(*e.entries, FileMetadataValue{Name: name, Value: value})
}

// Entries returns the collected metadata so far.
func (e PopulateFileMetadata) Entries() []FileMetadataValue { return *e.entries }

// supplyAnswer writes a hook's first non-empty answer; later calls are ignored.
func supplyAnswer(slot *string, answer string) {
	if *slot == "" {
		*slot = answer
	}
}
