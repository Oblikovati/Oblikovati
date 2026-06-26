// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"strconv"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/display"
	"oblikovati.org/model/doc"
)

// registerDisplayHandlers wires the application display options and per-document display
// settings methods (M16-F07, #643).
func (r *Router) registerDisplayHandlers() {
	r.readOnly(wire.MethodDisplayGetOptions, getDisplayOptions)
	r.readOnly(wire.MethodDisplaySetOptions, setDisplayOptions)
	r.readOnly(wire.MethodDocumentGetDisplaySettings, getDisplaySettings)
	r.readOnly(wire.MethodDocumentSetDisplaySettings, setDisplaySettings)
}

// getDisplayOptions returns the application-level display options (wire.MethodDisplayGetOptions).
func getDisplayOptions(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(displayModeOptionsView(s.DisplayOptionsData()))
}

// setDisplayOptions applies the application-level display options (wire.MethodDisplaySetOptions).
func setDisplayOptions(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var v wire.DisplayModeOptionsView
	if err := decode(args, &v); err != nil {
		return nil, err
	}
	s.SetDisplayOptions(displayModeOptionsOf(v))
	return json.Marshal(displayModeOptionsView(s.DisplayOptionsData()))
}

// getDisplaySettings returns a document's per-document display settings
// (wire.MethodDocumentGetDisplaySettings).
func getDisplaySettings(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.GetDisplaySettingsArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	id, err := docIDFromString(a.Document)
	if err != nil {
		return nil, err
	}
	return json.Marshal(displaySettingsView(s.DisplaySettings(id)))
}

// setDisplaySettings applies a document's per-document display settings
// (wire.MethodDocumentSetDisplaySettings).
func setDisplaySettings(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetDisplaySettingsArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	id, err := docIDFromString(a.Document)
	if err != nil {
		return nil, err
	}
	s.SetDisplaySettings(id, displaySettingsOf(a.Settings))
	return json.Marshal(displaySettingsView(s.DisplaySettings(id)))
}

// displayModeOptionsView / displayModeOptionsOf map between the app display options and their wire DTO.
func displayModeOptionsView(o display.Options) wire.DisplayModeOptionsView {
	return wire.DisplayModeOptionsView{
		DisplayQuality: o.DisplayQuality, ViewTransitionTime: o.ViewTransitionTime,
		MinimumFrameRate: o.MinimumFrameRate, HiddenLineDimmingPercent: o.HiddenLineDimmingPercent,
		EdgeColor: o.EdgeColor, NewWindowDisplayMode: o.NewWindowDisplayMode,
		NewWindowProjection: o.NewWindowProjection, BackFaceCulling: o.BackFaceCulling,
		UseRayTracing: o.UseRayTracing, RayTracingQuality: o.RayTracingQuality,
		Shaded: wire.ShadedDisplayModeOptionsView{
			EdgeDisplay: o.Shaded.EdgeDisplay, EdgeColor: o.Shaded.EdgeColor,
			Silhouettes: o.Shaded.Silhouettes, TransparencyType: o.Shaded.TransparencyType,
		},
		Wireframe: wire.WireframeDisplayModeOptionsView{
			DepthDimming: o.Wireframe.DepthDimming, Silhouettes: o.Wireframe.Silhouettes,
			DimmedHiddenEdges: o.Wireframe.DimmedHiddenEdges,
		},
	}
}

func displayModeOptionsOf(v wire.DisplayModeOptionsView) display.Options {
	return display.Options{
		DisplayQuality: v.DisplayQuality, ViewTransitionTime: v.ViewTransitionTime,
		MinimumFrameRate: v.MinimumFrameRate, HiddenLineDimmingPercent: v.HiddenLineDimmingPercent,
		EdgeColor: v.EdgeColor, NewWindowDisplayMode: v.NewWindowDisplayMode,
		NewWindowProjection: v.NewWindowProjection, BackFaceCulling: v.BackFaceCulling,
		UseRayTracing: v.UseRayTracing, RayTracingQuality: v.RayTracingQuality,
		Shaded: display.ShadedOptions{
			EdgeDisplay: v.Shaded.EdgeDisplay, EdgeColor: v.Shaded.EdgeColor,
			Silhouettes: v.Shaded.Silhouettes, TransparencyType: v.Shaded.TransparencyType,
		},
		Wireframe: display.WireframeOptions{
			DepthDimming: v.Wireframe.DepthDimming, Silhouettes: v.Wireframe.Silhouettes,
			DimmedHiddenEdges: v.Wireframe.DimmedHiddenEdges,
		},
	}
}

// displaySettingsView / displaySettingsOf map between the per-document settings and their DTO.
func displaySettingsView(set display.Settings) wire.DisplaySettingsView {
	return wire.DisplaySettingsView{
		BackgroundType: set.BackgroundType, EdgeColor: set.EdgeColor, DepthDimming: set.DepthDimming,
		DisplaySilhouettes: set.DisplaySilhouettes, HiddenLineDimmingPercent: set.HiddenLineDimmingPercent,
		NewWindowDisplayMode: set.NewWindowDisplayMode, DisplayModeSource: set.DisplayModeSource,
		NewWindowProjection: set.NewWindowProjection, GroundPlane: groundPlaneViewOf(set.GroundPlane),
		GroundShadow: set.GroundShadow, ShadowDirection: set.ShadowDirection,
		ShowGroundReflections: set.ShowGroundReflections, ShowObjectShadows: set.ShowObjectShadows,
		ShowAmbientShadows: set.ShowAmbientShadows, TexturesOn: set.TexturesOn,
	}
}

func displaySettingsOf(v wire.DisplaySettingsView) display.Settings {
	return display.Settings{
		BackgroundType: v.BackgroundType, EdgeColor: v.EdgeColor, DepthDimming: v.DepthDimming,
		DisplaySilhouettes: v.DisplaySilhouettes, HiddenLineDimmingPercent: v.HiddenLineDimmingPercent,
		NewWindowDisplayMode: v.NewWindowDisplayMode, DisplayModeSource: v.DisplayModeSource,
		NewWindowProjection: v.NewWindowProjection, GroundPlane: groundPlaneOf(v.GroundPlane),
		GroundShadow: v.GroundShadow, ShadowDirection: v.ShadowDirection,
		ShowGroundReflections: v.ShowGroundReflections, ShowObjectShadows: v.ShowObjectShadows,
		ShowAmbientShadows: v.ShowAmbientShadows, TexturesOn: v.TexturesOn,
	}
}

func groundPlaneViewOf(g display.GroundPlaneSettings) wire.GroundPlaneView {
	return wire.GroundPlaneView{
		Visible: g.Visible, Color: g.Color, HeightOffset: g.HeightOffset,
		DisplayGridLines: g.DisplayGridLines, MinorGridLineSpacing: g.MinorGridLineSpacing,
		MinorLinesPerMajorGridLine: g.MinorLinesPerMajorGridLine, Opacity: g.Opacity, Reflectivity: g.Reflectivity,
	}
}

func groundPlaneOf(v wire.GroundPlaneView) display.GroundPlaneSettings {
	return display.GroundPlaneSettings{
		Visible: v.Visible, Color: v.Color, HeightOffset: v.HeightOffset,
		DisplayGridLines: v.DisplayGridLines, MinorGridLineSpacing: v.MinorGridLineSpacing,
		MinorLinesPerMajorGridLine: v.MinorLinesPerMajorGridLine, Opacity: v.Opacity, Reflectivity: v.Reflectivity,
	}
}

// docIDFromString parses an optional document id (empty ⇒ 0, the active document).
func docIDFromString(s string) (doc.ID, error) {
	if s == "" {
		return 0, nil
	}
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return doc.ID(id), nil
}
