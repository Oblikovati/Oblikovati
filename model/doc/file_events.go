// SPDX-License-Identifier: GPL-2.0-only

package doc

import "oblikovati.org/event"

// The file-access event set (M04-F05, Oblikovati#613) on the workspace bus:
// reference resolution and dirty transitions. TypeIDs continue the document
// block; never renumber.
const (
	tidFileResolution event.TypeID = 0x0406
	tidFileDirty      event.TypeID = 0x0407
)

// FileResolution fires (Before) when a full document name fails to load — the
// hook that lets a subscriber supply a substitute path for a moved or renamed
// reference. The first handler to call [FileResolution.Resolve] wins; the
// workspace retries the load with the supplied name. After the open concludes
// the event fires again (After) carrying the outcome, for passive observers.
type FileResolution struct {
	RequestedName string
	// resolved is shared by every handler's copy of the event, so a handler's
	// Resolve is visible to the emitting workspace (events travel by value).
	resolved *string
}

// newFileResolution builds the event with its answer slot allocated.
func newFileResolution(requestedName string) FileResolution {
	return FileResolution{RequestedName: requestedName, resolved: new(string)}
}

// Resolve supplies the substitute full document name to load instead. The first
// non-empty answer sticks; later calls are ignored.
func (e FileResolution) Resolve(fullDocumentName string) {
	if *e.resolved == "" {
		*e.resolved = fullDocumentName
	}
}

// Resolved returns the substitute a handler supplied, "" when unanswered.
func (e FileResolution) Resolved() string { return *e.resolved }

// EventID implements event.Event.
func (FileResolution) EventID() event.TypeID { return tidFileResolution }

// FileDirty fires (After only) on a document's clean→dirty transition — the
// first unsaved change since open/save; further edits while already dirty do
// not re-fire. Subscribers use it to track which open files need attention
// (e.g. an autosave or session-recovery add-in).
type FileDirty struct{ Document *Document }

// EventID implements event.Event.
func (FileDirty) EventID() event.TypeID { return tidFileDirty }
