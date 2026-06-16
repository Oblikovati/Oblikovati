//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// drawGripSnapDialog shows the Grip Snap Move-Options panel while the tool is active: the Constraint
// override (Auto/Mate/Flush/Insert/Tangent) that the snap will create, and a HUD line reporting the
// constraint inferred on the last snap. The two face picks happen in the viewport (see the tool's
// prompt in the command window).
func drawGripSnapDialog(s *app.Session) {
	g := s.ActiveGripSnap()
	if g == nil {
		return
	}
	native.SetNextWindowSizeOnce(320, 180)
	if native.Begin("Grip Snap") {
		drawFeatureBreadcrumb("Grip Snap", "")
		if propertySection("Move Options") {
			if i := propertyComboRow("Constraint", "grip-snap-constraint", app.GripSnapPreferOptions(), g.PreferIndex()); i >= 0 {
				g.SetPreferIndex(i)
			}
			if inferred := g.Inferred(); inferred != "" {
				propertyRow("Created")
				native.Text(inferred) // HUD: the kind created on the last snap
			}
		}
		native.Separator()
		drawCommitCancelButtons(s, g.CanCommit())
	}
	native.End()
}
