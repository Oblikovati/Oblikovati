// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/model/collview"
	"oblikovati.org/model/colorscheme"
)

// ColorSchemes is the application's color-scheme registry as the public read/write contract
// (what the wire router serves and add-ins drive). M16-F06 (#642).
func (s *Session) ColorSchemes() contract.ColorSchemes { return colorSchemesAdapter{s} }

// ActiveColorScheme is the active scheme as plain model data — the head reads it each frame to
// paint the viewport background and the selection colors.
func (s *Session) ActiveColorScheme() colorscheme.Scheme { return s.colorSchemes.Active() }

// SetColorScheme activates the named scheme and bumps the color-scheme revision so the head
// re-applies the viewport colors next frame. Activating a scheme is an explicit choice of
// viewport background, so it turns off the environment-image (sky) background that would
// otherwise paint over the scheme's screen color — the IBL still lights the model, and the
// sky can be turned back on via the View ▸ Environment controls. It surfaces the registry's
// "no such scheme" error.
func (s *Session) SetColorScheme(name string) error {
	if err := s.colorSchemes.SetActive(name); err != nil {
		return err
	}
	s.lighting.Environment.ShowImage = false
	s.colorSchemeRev++
	return nil
}

// SetColorSchemeBackgroundType overrides the application-wide viewport background type and
// bumps the revision. It surfaces the registry's validation error.
func (s *Session) SetColorSchemeBackgroundType(t types.BackgroundTypeEnum) error {
	if err := s.colorSchemes.SetBackgroundType(t); err != nil {
		return err
	}
	s.colorSchemeRev++
	return nil
}

// ColorSchemeRevision is the change counter the head compares across frames to decide whether
// to re-apply the active scheme's colors to the viewport (the live-preview hook).
func (s *Session) ColorSchemeRevision() uint64 { return s.colorSchemeRev }

// colorSchemesAdapter exposes the session's registry through the api/contract interfaces so the
// public surface stays decoupled from the concrete colorscheme package.
type colorSchemesAdapter struct{ s *Session }

var _ contract.ColorSchemes = colorSchemesAdapter{}

func (a colorSchemesAdapter) Count() int { return len(a.s.colorSchemes.Schemes()) }

func (a colorSchemesAdapter) Item(i int) contract.ColorScheme {
	return collview.ItemAs(a.s.colorSchemes.Schemes(), i, func(sc colorscheme.Scheme) contract.ColorScheme { return schemeView{sc} })
}

func (a colorSchemesAdapter) Active() contract.ColorScheme {
	return schemeView{a.s.colorSchemes.Active()}
}

func (a colorSchemesAdapter) SetActive(name string) error { return a.s.SetColorScheme(name) }

func (a colorSchemesAdapter) BackgroundType() types.BackgroundTypeEnum {
	return a.s.colorSchemes.BackgroundType()
}

func (a colorSchemesAdapter) SetBackgroundType(t types.BackgroundTypeEnum) error {
	return a.s.SetColorSchemeBackgroundType(t)
}

// schemeView adapts one colorscheme.Scheme to the api/contract ColorScheme getters.
type schemeView struct{ s colorscheme.Scheme }

var _ contract.ColorScheme = schemeView{}

func (v schemeView) Name() string                      { return v.s.Name }
func (v schemeView) ScreenColor() types.Color          { return v.s.Screen }
func (v schemeView) TopScreenColor() types.Color       { return v.s.Top }
func (v schemeView) BottomScreenColor() types.Color    { return v.s.Bottom }
func (v schemeView) HighlightColor() types.Color       { return v.s.Highlight }
func (v schemeView) PrimarySelectColor() types.Color   { return v.s.PrimarySelect }
func (v schemeView) SecondarySelectColor() types.Color { return v.s.SecondSelect }
