// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"oblikovati.org/api/types"
	"oblikovati.org/model/display"
	"oblikovati.org/yamlcodec"
)

// toCodecColor / fromCodecColor convert a color value object to and from its on-disk record.
func toCodecColor(c types.Color) yamlcodec.ColorRecord {
	return yamlcodec.ColorRecord{R: c.R, G: c.G, B: c.B, Opacity: c.Opacity, Source: int32(c.Source)}
}

func fromCodecColor(c yamlcodec.ColorRecord) types.Color {
	return types.Color{R: c.R, G: c.G, B: c.B, Opacity: c.Opacity, Source: types.ColorSourceTypeEnum(c.Source)}
}

// toCodecDisplaySettings converts the model display settings into the on-disk record (M16-F07
// #643) so they round-trip in the .obk; the enums become their frozen integer ids.
func toCodecDisplaySettings(s display.Settings) *yamlcodec.DisplaySettingsRecord {
	return &yamlcodec.DisplaySettingsRecord{
		BackgroundType:           int32(s.BackgroundType),
		EdgeColor:                toCodecColor(s.EdgeColor),
		DepthDimming:             s.DepthDimming,
		DisplaySilhouettes:       s.DisplaySilhouettes,
		HiddenLineDimmingPercent: s.HiddenLineDimmingPercent,
		NewWindowDisplayMode:     int32(s.NewWindowDisplayMode),
		DisplayModeSource:        int32(s.DisplayModeSource),
		NewWindowProjection:      int32(s.NewWindowProjection),
		GroundPlane: yamlcodec.GroundPlaneRecord{
			Visible:                    s.GroundPlane.Visible,
			Color:                      toCodecColor(s.GroundPlane.Color),
			HeightOffset:               s.GroundPlane.HeightOffset,
			DisplayGridLines:           s.GroundPlane.DisplayGridLines,
			MinorGridLineSpacing:       s.GroundPlane.MinorGridLineSpacing,
			MinorLinesPerMajorGridLine: s.GroundPlane.MinorLinesPerMajorGridLine,
			Opacity:                    s.GroundPlane.Opacity,
			Reflectivity:               s.GroundPlane.Reflectivity,
		},
		GroundShadow:          int32(s.GroundShadow),
		ShadowDirection:       int32(s.ShadowDirection),
		ShowGroundReflections: s.ShowGroundReflections,
		ShowObjectShadows:     s.ShowObjectShadows,
		ShowAmbientShadows:    s.ShowAmbientShadows,
		TexturesOn:            s.TexturesOn,
	}
}

// fromCodecDisplaySettings rebuilds the model display settings from the on-disk record.
func fromCodecDisplaySettings(r *yamlcodec.DisplaySettingsRecord) display.Settings {
	return display.Settings{
		BackgroundType:           types.BackgroundTypeEnum(r.BackgroundType),
		EdgeColor:                fromCodecColor(r.EdgeColor),
		DepthDimming:             r.DepthDimming,
		DisplaySilhouettes:       r.DisplaySilhouettes,
		HiddenLineDimmingPercent: r.HiddenLineDimmingPercent,
		NewWindowDisplayMode:     types.DisplayModeEnum(r.NewWindowDisplayMode),
		DisplayModeSource:        types.DisplayModeSourceTypeEnum(r.DisplayModeSource),
		NewWindowProjection:      types.ProjectionTypeEnum(r.NewWindowProjection),
		GroundPlane: display.GroundPlaneSettings{
			Visible:                    r.GroundPlane.Visible,
			Color:                      fromCodecColor(r.GroundPlane.Color),
			HeightOffset:               r.GroundPlane.HeightOffset,
			DisplayGridLines:           r.GroundPlane.DisplayGridLines,
			MinorGridLineSpacing:       r.GroundPlane.MinorGridLineSpacing,
			MinorLinesPerMajorGridLine: r.GroundPlane.MinorLinesPerMajorGridLine,
			Opacity:                    r.GroundPlane.Opacity,
			Reflectivity:               r.GroundPlane.Reflectivity,
		},
		GroundShadow:          types.GroundShadowEnum(r.GroundShadow),
		ShadowDirection:       types.ShadowDirectionEnum(r.ShadowDirection),
		ShowGroundReflections: r.ShowGroundReflections,
		ShowObjectShadows:     r.ShowObjectShadows,
		ShowAmbientShadows:    r.ShowAmbientShadows,
		TexturesOn:            r.TexturesOn,
	}
}
