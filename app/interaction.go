// SPDX-License-Identifier: GPL-2.0-only

package app

// Interaction input — the device events the viewport feeds the session, modeled as
// plain data so tests inject them directly (the "auto-click" harness). Mouse/key
// behavior follows Autodesk Inventor: LMB selects, RMB opens the marking menu, MMB
// drag orbits (Shift+MMB pans), the wheel zooms.

// PointerButton identifies a mouse button.
type PointerButton uint8

const (
	LeftButton PointerButton = iota
	RightButton
	MiddleButton
)

// Modifier is a bitmask of held modifier keys.
type Modifier uint8

const (
	ShiftMod Modifier = 1 << iota
	CtrlMod
	AltMod
)

// Has reports whether a modifier is held.
func (m Modifier) Has(mod Modifier) bool { return m&mod != 0 }

// PointerEvent is a click/drag/move at a viewport coordinate.
type PointerEvent struct {
	X, Y   float64
	Button PointerButton
	Mods   Modifier
}

// KeyEvent is a key press, with any held modifiers.
type KeyEvent struct {
	Key  string
	Mods Modifier
}

// Picker maps a viewport coordinate to the front-most selectable honoring the active
// filter — Inventor's hit-test/pick. The real implementation reads the viewport
// ID-buffer (core/08); tests supply a fake that returns a known entity for a
// coordinate, which is how a test "clicks on" geometry headlessly.
type Picker interface {
	Pick(x, y float64, filter *SelectionFilter) (Selectable, bool)
}
