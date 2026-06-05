//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
)

// drawSplitDialog shows the Split options while the Split tool is active: the user picks a work
// plane in the view, chooses split-vs-trim, and OKs. The keep choice is driven through the
// tool's convenience setters so the head needs no model enum.
func drawSplitDialog(s *app.Session) {
	t := s.ActiveSplit()
	if t == nil {
		return
	}
	native.SetNextWindowSize(300, 170)
	if native.Begin("Split") {
		if _, ok := t.PickedPlane(); !ok {
			native.Text("Select a work plane to cut with")
		} else {
			native.Text("Mode: " + t.KeepLabel())
		}
		if native.Button("Split into two") {
			t.SetKeepBoth()
		}
		if native.Button("Trim — keep front side") {
			t.SetKeepPositive()
		}
		if native.Button("Trim — keep back side") {
			t.SetKeepNegative()
		}
		native.BeginDisabled(!t.CanCommit())
		if native.Button("OK") {
			_ = s.OK()
		}
		native.EndDisabled()
		native.SameLine()
		if native.Button("Cancel") {
			s.CancelTool()
		}
	}
	native.End()
}
