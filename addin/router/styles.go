// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/style"
)

// registerStyleHandlers wires the color-style + style-library methods (M16-F02, #403/#408).
func (r *Router) registerStyleHandlers() {
	r.readOnly(wire.MethodStylesList, listColorStyles)
	r.readOnly(wire.MethodStylesGet, typed(getColorStyle))
	r.readOnly(wire.MethodStylesSet, typed(setColorStyle))
	r.readOnly(wire.MethodStylesDelete, typed(deleteColorStyle))
	r.readOnly(wire.MethodStylesListLibraries, listStyleLibraries)
	r.readOnly(wire.MethodStylesImportLibrary, typed(importStyleLibrary))
}

// listColorStyles returns every color style in the document (wire.MethodStylesList).
func listColorStyles(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	styles := s.ColorStyles()
	out := make([]wire.ColorStyleView, len(styles))
	for i, cs := range styles {
		out[i] = colorStyleView(cs)
	}
	return json.Marshal(wire.ColorStylesResult{Styles: out})
}

// getColorStyle returns one color style by name (wire.MethodStylesGet).
func getColorStyle(s *app.Session, in wire.GetStyleArgs) (wire.ColorStyleView, error) {
	cs, ok := s.ColorStyle(in.Name)
	if !ok {
		return wire.ColorStyleView{}, fmt.Errorf("getColorStyle: no color style named %q", in.Name)
	}
	return colorStyleView(cs), nil
}

// setColorStyle creates or updates a color style and echoes it (wire.MethodStylesSet).
func setColorStyle(s *app.Session, in wire.ColorStyleView) (wire.ColorStyleView, error) {
	if err := s.SetColorStyle(colorStyleOf(in)); err != nil {
		return wire.ColorStyleView{}, err
	}
	cs, _ := s.ColorStyle(in.Name)
	return colorStyleView(cs), nil
}

// deleteColorStyle removes a color style by name (wire.MethodStylesDelete).
func deleteColorStyle(s *app.Session, in wire.GetStyleArgs) (wire.OKResult, error) {
	if err := s.DeleteColorStyle(in.Name); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// listStyleLibraries returns the loaded style libraries in cascade order
// (wire.MethodStylesListLibraries).
func listStyleLibraries(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(styleLibrariesResult(s))
}

// styleLibrariesResult snapshots the loaded style libraries in cascade order, shared by the
// list handler and importStyleLibrary (which echoes the updated cascade).
func styleLibrariesResult(s *app.Session) wire.StyleLibrariesResult {
	libs := s.StyleLibraries()
	out := make([]wire.StyleLibraryInfo, len(libs))
	for i, l := range libs {
		out[i] = wire.StyleLibraryInfo{Name: l.Name, Path: l.Path, Order: l.Order}
	}
	return wire.StyleLibrariesResult{Libraries: out}
}

// importStyleLibrary loads a style-library file into the cascade and returns the updated list
// (wire.MethodStylesImportLibrary).
func importStyleLibrary(s *app.Session, in wire.ImportStyleLibraryArgs) (wire.StyleLibrariesResult, error) {
	if err := s.StyleManager().ImportLibrary(in.Path); err != nil {
		return wire.StyleLibrariesResult{}, err
	}
	return styleLibrariesResult(s), nil
}

// colorStyleView projects one model color style into its wire shape.
func colorStyleView(cs style.ColorStyle) wire.ColorStyleView {
	return wire.ColorStyleView{
		Name: cs.Name, Diffuse: cs.Diffuse, Ambient: cs.Ambient, Specular: cs.Specular,
		Emissive: cs.Emissive, Shininess: cs.Shininess, Opacity: cs.Opacity, Location: cs.Location,
	}
}

// colorStyleOf builds a model color style from its wire shape.
func colorStyleOf(v wire.ColorStyleView) style.ColorStyle {
	return style.ColorStyle{
		Name: v.Name, Diffuse: v.Diffuse, Ambient: v.Ambient, Specular: v.Specular,
		Emissive: v.Emissive, Shininess: v.Shininess, Opacity: v.Opacity, Location: v.Location,
	}
}
