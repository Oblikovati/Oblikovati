//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// drawMeasureDialog shows the Measure readout panel while the tool is active: the running
// measurement for the picked faces/edges/vertices. Picking happens in the viewport (see the tool's
// prompt in the command window); the panel echoes the result and offers Close.
func drawMeasureDialog(s *app.Session) {
	m := s.ActiveMeasure()
	if m == nil {
		return
	}
	native.SetNextWindowSizeOnce(360, 150)
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
		drawCommitCancelButtons(s, m.CanCommit())
	}
	native.End()
}
