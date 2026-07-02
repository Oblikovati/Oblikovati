// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/math"

	"oblikovati.org/model/style"
	"oblikovati.org/renderer"
)

// AssignColorStyleToBody assigns a named color style to a body (by its reference key) so the
// body renders in that style's color (M16-F02 #403/#408). It errors when the style is unknown.
func (s *Session) AssignColorStyleToBody(bodyKey, styleName string) error {
	if _, ok := s.styles.ByName(styleName); !ok {
		return fmt.Errorf("assignColorStyle: no color style named %q", styleName)
	}
	s.bodyColorStyles[bodyKey] = styleName
	return nil
}

// ClearBodyColorStyle removes a body's color-style assignment (reverting to its appearance).
func (s *Session) ClearBodyColorStyle(bodyKey string) { delete(s.bodyColorStyles, bodyKey) }

// BodyColorStyle returns the color style assigned to a body, and whether one is assigned.
func (s *Session) BodyColorStyle(bodyKey string) (string, bool) {
	name, ok := s.bodyColorStyles[bodyKey]
	return name, ok
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
