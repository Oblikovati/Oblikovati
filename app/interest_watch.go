// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// The host behavior hook of the interest registry (M03-F10, #611): when a
// document opens carrying interests from clients that are not present this
// session, the user is warned — their add-in data is in the file but nothing
// loaded can service it. A warning in the status bar, never fatal.

// watchDocumentInterests subscribes the open-time absent-client check.
func (s *Session) watchDocumentInterests() {
	event.Subscribe(s.workspace.Events(), event.After, func(_ event.Context, e doc.DocumentOpened) event.Outcome {
		if d, ok := s.workspace.ByName(e.FullDocumentName); ok {
			if notice := s.absentInterestNotice(d); notice != "" {
				s.notice = notice
			}
		}
		return event.Continue()
	})
}

// absentInterestNotice names the interested clients missing from this
// session, "" when everyone is present.
func (s *Session) absentInterestNotice(d *doc.Document) string {
	var absent []string
	for _, rec := range d.InterestRecords() {
		if rec.InterestType != types.Interested {
			continue
		}
		if !s.interestClientPresent(rec.ClientID) {
			absent = append(absent, rec.ClientID)
		}
	}
	if len(absent) == 0 {
		return ""
	}
	return fmt.Sprintf("%s has add-in data from unavailable client(s): %s",
		d.DisplayName(), strings.Join(absent, ", "))
}

// interestClientPresent reports whether clientID matches a registered add-in
// or a connected client application.
func (s *Session) interestClientPresent(clientID string) bool {
	for _, id := range s.addins.Registered() {
		if id == clientID {
			return true
		}
	}
	for _, info := range s.clientApps.List() {
		if info.Name == clientID {
			return true
		}
	}
	return false
}
