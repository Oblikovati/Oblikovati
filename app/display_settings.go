// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/model/display"
	"oblikovati.org/model/doc"
)

// DocumentDisplaySettings returns a document's display settings as the public contract
// (Document 0 ⇒ active). M16-F07 (#643).
func (s *Session) DocumentDisplaySettings(id doc.ID) contract.DisplaySettings {
	return settingsView{s.DisplaySettings(id)}
}

// settingsView adapts a display.Settings value to contract.DisplaySettings.
type settingsView struct{ set display.Settings }

var _ contract.DisplaySettings = settingsView{}

// The capability families DisplaySettings embeds (I9, api v0.104.1). settingsView
// satisfies each as part of the union; asserting the window-mode and ground-shadow
// slices explicitly lets a caller depend on just one and keeps the #1619 guard honest.
var (
	_ contract.BackgroundDisplaySettings = settingsView{}
	_ contract.EdgeDisplaySettings       = settingsView{}
	_ contract.WindowDisplaySettings     = settingsView{}
	_ contract.GroundShadowSettings      = settingsView{}
)

func (v settingsView) BackgroundType() types.BackgroundTypeEnum { return v.set.BackgroundType }
func (v settingsView) EdgeColor() types.Color                   { return v.set.EdgeColor }
func (v settingsView) DepthDimming() bool                       { return v.set.DepthDimming }
func (v settingsView) DisplaySilhouettes() bool                 { return v.set.DisplaySilhouettes }
func (v settingsView) HiddenLineDimmingPercent() int            { return v.set.HiddenLineDimmingPercent }
func (v settingsView) NewWindowDisplayMode() types.DisplayModeEnum {
	return v.set.NewWindowDisplayMode
}
func (v settingsView) DisplayModeSource() types.DisplayModeSourceTypeEnum {
	return v.set.DisplayModeSource
}
func (v settingsView) NewWindowProjection() types.ProjectionTypeEnum {
	return v.set.NewWindowProjection
}
func (v settingsView) GroundPlane() contract.GroundPlaneSettings {
	return groundPlaneView{v.set.GroundPlane}
}
func (v settingsView) GroundShadow() types.GroundShadowEnum       { return v.set.GroundShadow }
func (v settingsView) ShadowDirection() types.ShadowDirectionEnum { return v.set.ShadowDirection }
func (v settingsView) ShowGroundReflections() bool                { return v.set.ShowGroundReflections }
func (v settingsView) ShowObjectShadows() bool                    { return v.set.ShowObjectShadows }
func (v settingsView) ShowAmbientShadows() bool                   { return v.set.ShowAmbientShadows }
func (v settingsView) TexturesOn() bool                           { return v.set.TexturesOn }

// groundPlaneView adapts a display.GroundPlaneSettings value to contract.GroundPlaneSettings.
type groundPlaneView struct{ g display.GroundPlaneSettings }

var _ contract.GroundPlaneSettings = groundPlaneView{}

func (v groundPlaneView) Visible() bool                 { return v.g.Visible }
func (v groundPlaneView) Color() types.Color            { return v.g.Color }
func (v groundPlaneView) HeightOffset() float64         { return v.g.HeightOffset }
func (v groundPlaneView) DisplayGridLines() bool        { return v.g.DisplayGridLines }
func (v groundPlaneView) MinorGridLineSpacing() float64 { return v.g.MinorGridLineSpacing }
func (v groundPlaneView) MinorLinesPerMajorGridLine() int {
	return v.g.MinorLinesPerMajorGridLine
}
func (v groundPlaneView) Opacity() float64      { return v.g.Opacity }
func (v groundPlaneView) Reflectivity() float64 { return v.g.Reflectivity }
