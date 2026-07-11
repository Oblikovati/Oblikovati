//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import "oblikovati.org/head/internal/native"

// drawTangentChainRow renders the "Tangent chain" selection toggle shared by the Fillet and Chamfer
// panels (#1947): when on — the default, matching Inventor's tangent propagation — a plain edge
// click selects the whole tangent loop through it; when off, one edge per click. Shift+click always
// selects the chain regardless. idScope keeps the imgui widget id unique per panel; set is the
// tool's SetTangentChain, called only on an actual toggle so it never fights the seeded state.
func drawTangentChainRow(idScope string, state *bool, set func(bool)) {
	propertyRow("")
	if native.Checkbox("Tangent chain##"+idScope, state) {
		set(*state)
	}
	native.SetItemTooltip("On: a click selects the whole connected (tangent) loop of edges — e.g. a rounded rim all around. Off: one edge per click. Shift+click always selects the loop.")
}
