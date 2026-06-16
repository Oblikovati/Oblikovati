// SPDX-License-Identifier: GPL-2.0-only

package display

import "oblikovati.org/api/types"

// GroundPlaneSettings is a document's ground plane — the receiver the renderer drops shadows
// and reflections onto. Lengths are cm; Opacity/Reflectivity in [0,1].
type GroundPlaneSettings struct {
	Visible                    bool
	Color                      types.Color
	HeightOffset               float64
	DisplayGridLines           bool
	MinorGridLineSpacing       float64
	MinorLinesPerMajorGridLine int
	Opacity                    float64
	Reflectivity               float64
}

// Settings are a document's per-document display settings — the home for background, edge
// color, ground-plane, and shadow state (Inventor's DisplaySettings).
type Settings struct {
	BackgroundType           types.BackgroundTypeEnum
	EdgeColor                types.Color
	DepthDimming             bool
	DisplaySilhouettes       bool
	HiddenLineDimmingPercent int
	NewWindowDisplayMode     types.DisplayModeEnum
	DisplayModeSource        types.DisplayModeSourceTypeEnum
	NewWindowProjection      types.ProjectionTypeEnum
	GroundPlane              GroundPlaneSettings
	GroundShadow             types.GroundShadowEnum
	ShadowDirection          types.ShadowDirectionEnum
	ShowGroundReflections    bool
	ShowObjectShadows        bool
	ShowAmbientShadows       bool
	TexturesOn               bool
}

// DefaultSettings are the out-of-the-box per-document display settings.
func DefaultSettings() Settings {
	return Settings{
		BackgroundType:           types.GradientBackground,
		EdgeColor:                types.NewColor(30, 30, 34),
		DepthDimming:             false,
		DisplaySilhouettes:       false,
		HiddenLineDimmingPercent: 50,
		NewWindowDisplayMode:     types.ShadedWithEdgesRendering,
		DisplayModeSource:        types.DefaultDisplayModeSource,
		NewWindowProjection:      types.OrthographicProjection,
		GroundPlane: GroundPlaneSettings{
			Visible: true, Color: types.NewColor(120, 120, 128), HeightOffset: 0,
			DisplayGridLines: true, MinorGridLineSpacing: 1, MinorLinesPerMajorGridLine: 10,
			Opacity: 0.6, Reflectivity: 0.2,
		},
		GroundShadow:          types.GroundShadow,
		ShadowDirection:       types.AboveShadow,
		ShowGroundReflections: false,
		ShowObjectShadows:     true,
		ShowAmbientShadows:    true,
		TexturesOn:            true,
	}
}
