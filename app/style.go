// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/event"
	"oblikovati.org/model/style"
	"oblikovati.org/renderer"
)

// StyleManager exposes the document's style registry through the public contract (M16-F02,
// #403/#408): the color styles, the lighting styles, and the style-library cascade.
func (s *Session) StyleManager() contract.StyleManager { return styleManagerView{s} }

// ColorStyles returns the document's color styles as plain model data (the head/UI reads it).
func (s *Session) ColorStyles() []style.ColorStyle { return s.styles.Styles() }

// ColorStyle returns the named color style and whether it exists.
func (s *Session) ColorStyle(name string) (style.ColorStyle, bool) { return s.styles.ByName(name) }

// SetColorStyle creates or updates a color style and fires the matching style event so every
// consumer re-resolves its styling.
func (s *Session) SetColorStyle(cs style.ColorStyle) error {
	added, err := s.styles.Set(cs)
	if err != nil {
		return err
	}
	kind := StyleEdited
	if added {
		kind = StyleAdded
	}
	event.Emit(s.bus, event.After, StyleChanged{Name: cs.Name, Kind: kind})
	return nil
}

// DeleteColorStyle removes a color style and fires the delete event.
func (s *Session) DeleteColorStyle(name string) error {
	if err := s.styles.Delete(name); err != nil {
		return err
	}
	event.Emit(s.bus, event.After, StyleChanged{Name: name, Kind: StyleDeleted})
	return nil
}

// StyleLibraries returns the loaded style libraries in cascade order.
func (s *Session) StyleLibraries() []style.Library { return s.styles.Libraries() }

// ImportStyleLibrary folds a parsed style library into the cascade and fires an add event for
// each newly merged style. File parsing happens in the caller (the DI rule keeps file I/O out
// of the model and app-core); this takes the already-loaded library.
func (s *Session) ImportStyleLibrary(lib style.Library) {
	for _, name := range s.styles.Import(lib) {
		event.Emit(s.bus, event.After, StyleChanged{Name: name, Kind: StyleAdded})
	}
}

// styleManagerView adapts the session's registry to the api/contract StyleManager interface.
type styleManagerView struct{ s *Session }

var _ contract.StyleManager = styleManagerView{}

func (v styleManagerView) ColorStyles() contract.ColorStyles { return colorStylesView{v.s} }

func (v styleManagerView) LightingStyles() []contract.LightingStyle {
	gallery := renderer.LightingStyleGallery()
	out := make([]contract.LightingStyle, len(gallery))
	for i, opt := range gallery {
		out[i] = LightingStyleOf(opt.Name, renderer.SceneLightingFor(opt.Style))
	}
	return out
}

func (v styleManagerView) LibraryNames() []string {
	libs := v.s.styles.Libraries()
	names := make([]string, len(libs))
	for i, l := range libs {
		names[i] = l.Name
	}
	return names
}

func (v styleManagerView) ImportLibrary(path string) error {
	lib, err := loadStyleLibrary(path)
	if err != nil {
		return err
	}
	v.s.ImportStyleLibrary(lib)
	return nil
}

// colorStylesView adapts the registry's color styles to contract.ColorStyles.
type colorStylesView struct{ s *Session }

var _ contract.ColorStyles = colorStylesView{}

func (v colorStylesView) Count() int { return len(v.s.styles.Styles()) }

func (v colorStylesView) Item(i int) contract.ColorStyle {
	styles := v.s.styles.Styles()
	if i < 0 || i >= len(styles) {
		return nil
	}
	return colorStyleView{styles[i]}
}

func (v colorStylesView) ByName(name string) contract.ColorStyle {
	cs, ok := v.s.styles.ByName(name)
	if !ok {
		return nil
	}
	return colorStyleView{cs}
}

// colorStyleView adapts one style.ColorStyle to contract.ColorStyle.
type colorStyleView struct{ cs style.ColorStyle }

var _ contract.ColorStyle = colorStyleView{}

func (v colorStyleView) Name() string                      { return v.cs.Name }
func (v colorStyleView) DiffuseColor() types.Color         { return v.cs.Diffuse }
func (v colorStyleView) AmbientColor() types.Color         { return v.cs.Ambient }
func (v colorStyleView) SpecularColor() types.Color        { return v.cs.Specular }
func (v colorStyleView) EmissiveColor() types.Color        { return v.cs.Emissive }
func (v colorStyleView) Shininess() float64                { return v.cs.Shininess }
func (v colorStyleView) Opacity() float64                  { return v.cs.Opacity }
func (v colorStyleView) Location() types.StyleLocationEnum { return v.cs.Location }
