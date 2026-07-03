// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/event"
	"oblikovati.org/math"
	"oblikovati.org/model/doc"

	"oblikovati.org/model/style"
	"oblikovati.org/renderer"
)

// BodyColorStyleChanged fires (After) when a body's color-style assignment is set or cleared (M16-F02
// #403/#408, S5 #1640). Style is the new style name, empty when the assignment was cleared; add-ins
// (renderers, exporters) re-resolve the body's appearance from it. It is the granular appearance
// notification the assignment used to skip entirely — the map lived only in session memory, so no
// event fired and no observer could react.
type BodyColorStyleChanged struct {
	Document doc.ID
	BodyKey  string
	Style    string
}

// EventID implements event.Event.
func (BodyColorStyleChanged) EventID() event.TypeID { return tidBodyColorStyleChanged }

// AssignColorStyleToBody assigns a named color style to a body (by its reference key) on the active
// document so the body renders in that style's color (M16-F02 #403/#408). The assignment is document
// data: it persists in the .obk, is undoable, and fires [BodyColorStyleChanged] (S5 #1640). It errors
// when the style is unknown or no document is active.
func (s *Session) AssignColorStyleToBody(bodyKey, styleName string) error {
	if _, ok := s.styles.ByName(styleName); !ok {
		return fmt.Errorf("assignColorStyle: no color style named %q", styleName)
	}
	return s.setBodyColorStyle(bodyKey, styleName, "Assign Color Style")
}

// ClearBodyColorStyle removes a body's color-style assignment (reverting to its appearance),
// recording it as an undo step and firing [BodyColorStyleChanged].
func (s *Session) ClearBodyColorStyle(bodyKey string) {
	_ = s.setBodyColorStyle(bodyKey, "", "Clear Color Style")
}

// setBodyColorStyle writes (or, with an empty name, clears) the active document's body color style,
// capturing the pre-edit baseline so the change is one undo step, then emitting the change event.
func (s *Session) setBodyColorStyle(bodyKey, styleName, label string) error {
	d := s.ActiveDocument()
	if d == nil {
		return errNoActiveDocument
	}
	s.beginMetadataEdit(d)
	d.SetBodyColorStyle(bodyKey, styleName)
	s.recordMetadataEdit(d, label)
	event.Emit(s.bus, event.After, BodyColorStyleChanged{Document: d.ID(), BodyKey: bodyKey, Style: styleName})
	return nil
}

// BodyColorStyle returns the color style assigned to a body on the active document, and whether one
// is assigned.
func (s *Session) BodyColorStyle(bodyKey string) (string, bool) {
	d := s.ActiveDocument()
	if d == nil {
		return "", false
	}
	return d.BodyColorStyle(bodyKey)
}

// SelectedBodyKey returns the reference key of the currently selected body, and whether a body
// is selected — the target the Color Styles panel applies a style to.
func (s *Session) SelectedBodyKey() (string, bool) {
	if h, ok := s.Selection().First().(BodyHandle); ok && h.Body != nil {
		return string(h.Body.ReferenceKey()), true
	}
	return "", false
}

// OpenColorStylesPanel opens the Color Styles panel (M16-F02 #403/#408).
func (s *Session) OpenColorStylesPanel() { s.colorStylesPanelOpen = true }

// CloseColorStylesPanel closes the Color Styles panel.
func (s *Session) CloseColorStylesPanel() { s.colorStylesPanelOpen = false }

// ColorStylesPanelOpen reports whether the Color Styles panel is open.
func (s *Session) ColorStylesPanelOpen() bool { return s.colorStylesPanelOpen }

// styleSurface converts a color style into the renderer's PBR surface: the diffuse drives the
// albedo, shininess maps to (1-roughness), and the specular's brightness hints the metalness.
func styleSurface(cs style.ColorStyle) renderer.Surface {
	em := cs.Emissive.Rgba()
	return renderer.Surface{
		Albedo:    cs.Diffuse.Rgba().Array(),
		Metallic:  0,
		Roughness: float32(math.Clamp01(1 - cs.Shininess)),
		Emissive:  [3]float32{em.R, em.G, em.B},
		Opacity:   float32(math.Clamp01(cs.Opacity)),
	}
}
