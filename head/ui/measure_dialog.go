//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// measureHost is the slim session surface the Measure readout panel consumes (audit I5,
// the arrowSession pattern): the running Measure tool plus the shared commit/cancel
// controls — not the whole session. *app.Session satisfies it implicitly, so drawMeasure
// is unit-testable against a small fake host.
type measureHost interface {
	ActiveMeasure() *app.MeasureTool
	commitCancelHost
}

var _ measureHost = (*app.Session)(nil)

// drawMeasureDialog is the registry-facing adapter; the panel itself consumes only
// measureHost.
func drawMeasureDialog(s *app.Session) { drawMeasure(s) }

// drawMeasure shows the Measure readout panel while the tool is active: the running
// measurement for the picked faces/edges/vertices. Picking happens in the viewport (see the tool's
// prompt in the command window); the panel echoes the result and offers Close.
func drawMeasure(h measureHost) {
	m := h.ActiveMeasure()
	if m == nil {
		return
	}
	dialogSizeOnce(360, 150)
	if native.Begin("Measure") {
		drawFeatureBreadcrumb("Measure", "")
		if propertySection("Measurement") {
			propertyRow("Result")
			readout := m.Readout()
			if readout == "" {
				readout = "pick a face, edge or vertex"
			}
			native.Text(readout)
		}
		native.Separator()
		drawCommitCancelButtons(h, m.CanCommit())
	}
	native.End()
}
