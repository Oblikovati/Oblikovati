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
	r.readOnly(wire.MethodStylesGet, getColorStyle)
	r.readOnly(wire.MethodStylesSet, setColorStyle)
	r.readOnly(wire.MethodStylesDelete, deleteColorStyle)
	r.readOnly(wire.MethodStylesListLibraries, listStyleLibraries)
	r.readOnly(wire.MethodStylesImportLibrary, importStyleLibrary)
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
func getColorStyle(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.GetStyleArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	cs, ok := s.ColorStyle(a.Name)
	if !ok {
		return nil, fmt.Errorf("getColorStyle: no color style named %q", a.Name)
	}
	return json.Marshal(colorStyleView(cs))
}

// setColorStyle creates or updates a color style and echoes it (wire.MethodStylesSet).
func setColorStyle(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var v wire.ColorStyleView
	if err := decode(args, &v); err != nil {
		return nil, err
	}
	if err := s.SetColorStyle(colorStyleOf(v)); err != nil {
		return nil, err
	}
	cs, _ := s.ColorStyle(v.Name)
	return json.Marshal(colorStyleView(cs))
}

// deleteColorStyle removes a color style by name (wire.MethodStylesDelete).
func deleteColorStyle(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.GetStyleArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if err := s.DeleteColorStyle(a.Name); err != nil {
		return nil, err
	}
	return json.Marshal(wire.OKResult{OK: true})
}

// listStyleLibraries returns the loaded style libraries in cascade order
// (wire.MethodStylesListLibraries).
func listStyleLibraries(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	libs := s.StyleLibraries()
	out := make([]wire.StyleLibraryInfo, len(libs))
	for i, l := range libs {
		out[i] = wire.StyleLibraryInfo{Name: l.Name, Path: l.Path, Order: l.Order}
	}
	return json.Marshal(wire.StyleLibrariesResult{Libraries: out})
}

// importStyleLibrary loads a style-library file into the cascade and returns the updated list
// (wire.MethodStylesImportLibrary).
func importStyleLibrary(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.ImportStyleLibraryArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if err := s.StyleManager().ImportLibrary(a.Path); err != nil {
		return nil, err
	}
	return listStyleLibraries(s, nil)
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
