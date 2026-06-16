// SPDX-License-Identifier: GPL-2.0-only

// Package style is the document's color styles and the style-library cascade (M16-F02,
// #403/#408): named visual styles (ambient/diffuse/specular/emissive + shininess/opacity) a
// body or face references, plus loaded style libraries. It is pure data — the app fires the
// style-change events and loads library files; this package never touches the filesystem.
package style

import "oblikovati.org/api/types"

// ColorStyle is one named color style. The color components are full value objects; Shininess
// and Opacity are in [0,1]. Location records where it sits in the cascade (a Local style
// overrides the Library style of the same name).
type ColorStyle struct {
	Name      string
	Diffuse   types.Color
	Ambient   types.Color
	Specular  types.Color
	Emissive  types.Color
	Shininess float64
	Opacity   float64
	Location  types.StyleLocationEnum
}

// Library is a loaded style library: its name, the file it came from, its cascade position
// (lower Order shadows higher Order for a same-named style), and the styles it carries.
type Library struct {
	Name   string
	Path   string
	Order  int
	Styles []ColorStyle
}

// builtinStyles is the out-of-the-box local gallery — a neutral default plus a couple of
// common material looks. Local-located so they sit at the top of the cascade.
func builtinStyles() []ColorStyle {
	black := types.NewColor(0, 0, 0)
	return []ColorStyle{
		{Name: "Default", Diffuse: types.NewColor(200, 200, 205), Ambient: types.NewColor(60, 60, 64),
			Specular: types.NewColor(230, 230, 235), Emissive: black, Shininess: 0.3, Opacity: 1, Location: types.LocalStyleLocation},
		{Name: "Steel", Diffuse: types.NewColor(170, 174, 182), Ambient: types.NewColor(50, 52, 56),
			Specular: types.NewColor(245, 247, 250), Emissive: black, Shininess: 0.6, Opacity: 1, Location: types.LocalStyleLocation},
		{Name: "Brass", Diffuse: types.NewColor(196, 160, 72), Ambient: types.NewColor(64, 52, 24),
			Specular: types.NewColor(250, 235, 180), Emissive: black, Shininess: 0.5, Opacity: 1, Location: types.LocalStyleLocation},
	}
}
