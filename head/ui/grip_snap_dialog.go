//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// gripSnapHost is the slim session surface the Grip Snap panel consumes (audit I5, the
// arrowSession pattern): the running Grip Snap tool plus the shared commit/cancel
// controls. *app.Session satisfies it implicitly.
type gripSnapHost interface {
	ActiveGripSnap() *app.GripSnapTool
	commitCancelHost
}

var _ gripSnapHost = (*app.Session)(nil)

// drawGripSnapDialog is the registry-facing adapter; the panel itself consumes gripSnapHost.
func drawGripSnapDialog(s *app.Session) { drawGripSnap(s) }

// drawGripSnap shows the Grip Snap Move-Options panel while the tool is active: the Constraint
// override (Auto/Mate/Flush/Insert/Tangent) that the snap will create, and a HUD line reporting the
// constraint inferred on the last snap. The two face picks happen in the viewport (see the tool's
// prompt in the command window).
func drawGripSnap(h gripSnapHost) {
	g := h.ActiveGripSnap()
	if g == nil {
		return
	}
	dialogSizeOnce(320, 180)
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
		drawCommitCancelButtons(h, g.CanCommit())
	}
	native.End()
}
