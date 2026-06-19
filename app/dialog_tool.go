// SPDX-License-Identifier: GPL-2.0-only

package app

// dialogTool supplies the no-op viewport-interaction hooks (Start/Pick/Cancel) shared by
// dialog-driven tools — tools that gather all their input from the Params dialog, so they
// capture no activation state, pick nothing in the 3D view, and keep nothing to discard on
// cancel. Embed it and override only the hooks a tool actually needs (commonly Start, to
// capture the base-view list). It mirrors the no-op hooks derivedViewTool already provides.
type dialogTool struct{}

func (dialogTool) Start(*Session) {
	// no-op: a dialog-driven tool captures no state when it is activated
}

func (dialogTool) Pick(*Session, Selectable) {
	// no-op: a dialog-driven tool takes its input from the Params dialog, not viewport picks
}

func (dialogTool) Cancel(*Session) {
	// no-op: a dialog-driven tool keeps no in-progress state to discard
}
