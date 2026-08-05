// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/renderer"
)

// Driving the session the way a user's mouse and keyboard do — the entry point behind
// wire.MethodViewportClick and wire.MethodViewportKey.
//
// Everything an add-in could reach before edited the model directly, which cannot exercise the
// behaviour that only exists on the way IN: the constraints a tool infers from where the click
// landed, what a command previews between clicks, how a multi-click chain accumulates. Those bugs
// are invisible to a client that creates geometry by calling the model, because such a client
// never takes the path they live on (#2032).

// ProjectToViewport maps a model-space point to its viewport pixel, reporting false when the point
// falls outside the view frustum (behind the camera, or clipped).
//
//	x, y, ok := s.ProjectToViewport(sk.Plane().ToModel(math.P2(6, 0)))
func (s *Session) ProjectToViewport(p math.Point3) (x, y float64, ok bool) {
	return renderer.Project(s.camera, regionNear, regionFar, p)
}

// ClickPointer delivers a synthetic pointer click at a viewport pixel and reports the tool still
// running afterwards — empty once the click finished a command, which is how a caller tells "the
// shape was created" from "another click is expected".
func (s *Session) ClickPointer(x, y float64, button PointerButton, mods Modifier) string {
	s.Pointer(PointerEvent{X: x, Y: y, Button: button, Mods: mods})
	return s.ActiveToolName()
}

// PressKeyNamed delivers a key by name to the running command and reports the tool still running
// afterwards. It is [Session.PressKey] with the name resolved, so a caller need not build a
// KeyEvent.
func (s *Session) PressKeyNamed(key string, mods Modifier) (string, error) {
	if key == "" {
		return "", fmt.Errorf("app: no key given (want a key name such as \"Escape\", \"Enter\" or \"Delete\")")
	}
	if s.placementEditKey(key) {
		return s.ActiveToolName(), nil
	}
	if err := s.PressKey(KeyEvent{Key: key, Mods: mods}); err != nil {
		return s.ActiveToolName(), err
	}
	return s.ActiveToolName(), nil
}

// ActiveToolName is the running command's name, empty when none is running.
func (s *Session) ActiveToolName() string {
	if t := s.ActiveTool(); t != nil {
		return t.Name()
	}
	return ""
}

// PointerButtonNamed resolves a button name to its button, defaulting to the left button for an
// empty name — the overwhelmingly common case, and the only one that reaches a tool.
func PointerButtonNamed(name string) (PointerButton, error) {
	switch name {
	case "", "left":
		return LeftButton, nil
	case "right":
		return RightButton, nil
	case "middle":
		return MiddleButton, nil
	}
	return 0, fmt.Errorf("app: %q is not a pointer button (want \"left\", \"right\" or \"middle\")", name)
}

// ModifierFor packs the held modifier keys into the mask the selection and snapping paths read.
func ModifierFor(shift, ctrl, alt bool) Modifier {
	var m Modifier
	if shift {
		m |= ShiftMod
	}
	if ctrl {
		m |= CtrlMod
	}
	if alt {
		m |= AltMod
	}
	return m
}

// placementEditKey applies an editing key to the in-place dimension boxes while a shape is being
// placed, reporting whether it consumed the key.
//
// The head reaches the boxes through its own per-frame editing-key read, so this is the wire's
// equivalent — without it a client could type a value (which BeginCommandTyping hands to the
// boxes) but never LOCK it, since Tab resolves as a chord and dies unbound (#2034). It lives on
// the wire-only entry point deliberately: routing it inside PressKey would double-apply against
// the head, which already reads Tab itself.
func (s *Session) placementEditKey(key string) bool {
	if !s.PlacingGeometry() {
		return false
	}
	switch key {
	case "Tab":
		s.PlacementFieldTab()
	case "Backspace":
		s.PlacementFieldBackspace()
	default:
		return false
	}
	return true
}
