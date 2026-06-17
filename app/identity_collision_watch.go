// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// Open-time identity-collision notice: two open files must never share an identity GUID
// (references key on it, and the document tabs are keyed by it). When a file is opened whose
// persisted GUID clashes with an already-open document, the workspace mints the newcomer a
// fresh GUID; this surfaces that to the user so they know the on-disk file's id will change on
// its next save. A status notice, never fatal.

// watchDocumentIdentityCollisions subscribes the open-time identity-reassignment notice.
func (s *Session) watchDocumentIdentityCollisions() {
	event.Subscribe(s.workspace.Events(), event.After,
		func(_ event.Context, e doc.DocumentIdentityReassigned) event.Outcome {
			s.notice = identityCollisionNotice(e)
			return event.Continue()
		})
}

// identityCollisionNotice is the user-facing message for a reassigned identity: the opened
// file shared an id with an open document and was given a new one, saved on its next save.
func identityCollisionNotice(e doc.DocumentIdentityReassigned) string {
	return fmt.Sprintf(
		"%s shared a file identity with an already-open document; it was assigned a new id, which will be written on its next save.",
		e.Document.DisplayName())
}
