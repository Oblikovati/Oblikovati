// SPDX-License-Identifier: GPL-2.0-only

// Package colorscheme is the application's named color palettes (M16-F06, #642): the
// viewport background and the highlight/selection colors the renderer and selection pipeline
// traffic in. It is pure data — the app layer applies the active scheme to the live viewport;
// the head/renderer never imports this package directly.
package colorscheme

import "oblikovati.org/api/types"

// Scheme is one named palette. Background colors honor BackgroundType: Screen for a solid
// background, Top/Bottom for a gradient. Highlight/PrimarySelect/SecondarySelect feed the
// selection pipeline. The fields mirror the api/contract ColorScheme getters.
type Scheme struct {
	Name           string
	BackgroundType types.BackgroundTypeEnum
	Screen         types.Color
	Top            types.Color
	Bottom         types.Color
	Highlight      types.Color
	PrimarySelect  types.Color
	SecondSelect   types.Color
}

// builtinSchemes is the out-of-the-box gallery, in picker order. The first entry is the
// default active scheme. Colors mirror the renderer's existing defaults so activating
// "Default" is a no-op visual change.
func builtinSchemes() []Scheme {
	return []Scheme{
		{
			Name: "Default", BackgroundType: types.GradientBackground,
			Screen: types.NewColor(45, 48, 54), Top: types.NewColor(60, 64, 72), Bottom: types.NewColor(30, 32, 36),
			Highlight: types.NewColor(255, 196, 0), PrimarySelect: types.NewColor(60, 160, 255), SecondSelect: types.NewColor(0, 200, 160),
		},
		{
			Name: "Presentation", BackgroundType: types.GradientBackground,
			Screen: types.NewColor(235, 238, 242), Top: types.NewColor(245, 247, 250), Bottom: types.NewColor(205, 212, 222),
			Highlight: types.NewColor(255, 140, 0), PrimarySelect: types.NewColor(0, 120, 215), SecondSelect: types.NewColor(0, 160, 120),
		},
		{
			Name: "High Contrast", BackgroundType: types.OneColorBackground,
			Screen: types.NewColor(0, 0, 0), Top: types.NewColor(0, 0, 0), Bottom: types.NewColor(0, 0, 0),
			Highlight: types.NewColor(255, 255, 0), PrimarySelect: types.NewColor(0, 255, 255), SecondSelect: types.NewColor(255, 0, 255),
		},
		{
			Name: "Sky", BackgroundType: types.GradientBackground,
			Screen: types.NewColor(120, 160, 210), Top: types.NewColor(150, 190, 235), Bottom: types.NewColor(225, 235, 245),
			Highlight: types.NewColor(255, 170, 0), PrimarySelect: types.NewColor(20, 110, 200), SecondSelect: types.NewColor(0, 150, 110),
		},
	}
}
