// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"log/slog"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// MessageCenter aggregates errors/warnings/notices into a reviewable, sectioned tree
// — the ErrorManager + MessageSection equivalent (M05-F09, #616). A command's
// sub-operations report under one section (sections nest); the head shows the tree in
// the Messages panel. Every entry is also slog'd (structured JSON per house rules),
// so the operation trace carries the same record.
type MessageCenter struct {
	root        wire.MessageSectionView
	openPath    []int // indexes into the nested Sections, innermost last
	nextSection int
	sectionIDs  map[int][]int // section id → its path, while open
	hasErrors   bool
	hasWarnings bool
	lastMessage string
	sink        func(text string, severity types.MessageSeverity) // M26 F03: mirror to the command line
}

// NewMessageCenter returns an empty message center.
func NewMessageCenter() *MessageCenter {
	return &MessageCenter{nextSection: 1, sectionIDs: map[int][]int{}}
}

// BeginSection opens a titled section under the innermost open one; messages added
// until EndSection land inside it. Returns the section's id.
func (m *MessageCenter) BeginSection(title string) int {
	parent := m.sectionAt(m.openPath)
	parent.Sections = append(parent.Sections, wire.MessageSectionView{Title: title})
	m.openPath = append(m.openPath, len(parent.Sections)-1)
	id := m.nextSection
	m.nextSection++
	m.sectionIDs[id] = append([]int(nil), m.openPath...)
	return id
}

// EndSection closes an open section; it must be the innermost one (sections nest
// like brackets — closing an outer section with an inner one open is a bug).
func (m *MessageCenter) EndSection(id int) error {
	path, ok := m.sectionIDs[id]
	if !ok {
		return fmt.Errorf("app: no open message section %d", id)
	}
	if len(path) != len(m.openPath) {
		return fmt.Errorf("app: message section %d is not the innermost open section", id)
	}
	delete(m.sectionIDs, id)
	m.openPath = m.openPath[:len(m.openPath)-1]
	return nil
}

// AddMessage reports one entry under the innermost open section, updates the
// aggregate flags, and logs it (structured).
func (m *MessageCenter) AddMessage(text string, severity types.MessageSeverity) {
	section := m.sectionAt(m.openPath)
	section.Messages = append(section.Messages, wire.MessageEntry{Text: text, Severity: severity})
	m.lastMessage = text
	switch severity {
	case types.SeverityError:
		m.hasErrors = true
		slog.Error(logMessageCenter, "text", text)
	case types.SeverityWarning:
		m.hasWarnings = true
		slog.Warn(logMessageCenter, "text", text)
	default:
		slog.Info(logMessageCenter, "text", text)
	}
	if m.sink != nil { // M26 F03: also surface it in the Command Window
		m.sink(text, severity)
	}
}

// HasErrors / HasWarnings / LastMessage are the aggregate ErrorManager reads.
func (m *MessageCenter) HasErrors() bool     { return m.hasErrors }
func (m *MessageCenter) HasWarnings() bool   { return m.hasWarnings }
func (m *MessageCenter) LastMessage() string { return m.lastMessage }

// Clear empties the tree and resets the flags (open sections are abandoned).
func (m *MessageCenter) Clear() {
	m.root = wire.MessageSectionView{}
	m.openPath = nil
	m.sectionIDs = map[int][]int{}
	m.hasErrors, m.hasWarnings, m.lastMessage = false, false, ""
}

// View returns the current tree for serialization/rendering (a deep-shared copy is
// unnecessary: callers render or marshal it immediately on the session goroutine).
func (m *MessageCenter) View() wire.MessageSectionView { return m.root }

// sectionAt walks the nested sections to the given path.
func (m *MessageCenter) sectionAt(path []int) *wire.MessageSectionView {
	section := &m.root
	for _, i := range path {
		section = &section.Sections[i]
	}
	return section
}

// Messages returns the session's message center.
func (s *Session) Messages() *MessageCenter { return s.messageCenter }

// MessageCenterOpen reports / SetMessageCenterOpen sets whether the head shows the
// Messages panel (errors.show and the status-bar indicator toggle it).
func (s *Session) MessageCenterOpen() bool        { return s.messageCenterOpen }
func (s *Session) SetMessageCenterOpen(open bool) { s.messageCenterOpen = open }
