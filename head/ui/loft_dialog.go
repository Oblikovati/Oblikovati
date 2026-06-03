//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
)

// The Loft flow in the head: while the Loft tool runs, a modeless options window shows
// the picked cross-sections, the output operation and a closed-loop toggle, then
// OK/Cancel. The picked sections are outlined by the tool's preview.
var loftUI = struct{ open bool }{}

// drawLoftDialog shows the loft options window while the Loft tool is active.
func drawLoftDialog(s *app.Session) {
	l := s.ActiveLoft()
	if l == nil {
		loftUI.open = false
		return
	}
	loftUI.open = true
	native.SetNextWindowSize(300, 200)
	if native.Begin("Loft") {
		native.Text("Sections: " + strconv.Itoa(len(l.Sections())) + " (click regions in order)")
		loftOperationCombo(l)
		closed := l.Closed()
		if native.Checkbox("Closed loop", &closed) {
			l.SetClosed(closed)
		}
		drawLoftButtons(s, l)
	}
	native.End()
}

func loftOperationCombo(l *app.LoftTool) {
	preview := "New Solid"
	for _, o := range extrudeOperations {
		if o.op == l.Operation() {
			preview = o.label
		}
	}
	if native.BeginCombo("Output", preview) {
		for _, o := range extrudeOperations {
			if native.Selectable(o.label, o.op == l.Operation()) {
				l.SetOperation(o.op)
			}
		}
		native.EndCombo()
	}
}

func drawLoftButtons(s *app.Session, l *app.LoftTool) {
	native.BeginDisabled(!l.CanCommit())
	if native.Button("OK") {
		_ = s.OK()
	}
	native.EndDisabled()
	native.SameLine()
	if native.Button("Cancel") {
		s.CancelTool()
	}
}
