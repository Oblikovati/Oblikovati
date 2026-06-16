// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/model/display"
	"oblikovati.org/model/doc"
)

// DisplayOptions returns the application-level display options as the public contract (M16-F07,
// #643).
func (s *Session) DisplayOptions() contract.DisplayOptions { return displayOptionsView{s} }

// DisplayOptionsData returns the app display options as plain model data (head/UI reads it).
func (s *Session) DisplayOptionsData() display.Options { return s.displayOptions }

// SetDisplayOptions replaces the application-level display options.
func (s *Session) SetDisplayOptions(o display.Options) { s.displayOptions = o }

// DisplaySettings returns a document's per-document display settings, seeding the defaults the
// first time a document is asked for. Document 0 selects the active document.
func (s *Session) DisplaySettings(id doc.ID) display.Settings {
	id = s.resolveDocID(id)
	if set, ok := s.docDisplay[id]; ok {
		return set
	}
	def := display.DefaultSettings()
	s.docDisplay[id] = def
	return def
}

// SetDisplaySettings stores a document's per-document display settings (Document 0 ⇒ active).
func (s *Session) SetDisplaySettings(id doc.ID, set display.Settings) {
	s.docDisplay[s.resolveDocID(id)] = set
}

// GroundPlaneVisible reports whether the active document's display settings keep the ground
// plane visible (M16-F07 #643).
func (s *Session) GroundPlaneVisible() bool { return s.DisplaySettings(0).GroundPlane.Visible }

// SetGroundPlaneVisible toggles the active document's ground-plane visibility.
func (s *Session) SetGroundPlaneVisible(v bool) {
	set := s.DisplaySettings(0)
	set.GroundPlane.Visible = v
	s.SetDisplaySettings(0, set)
}

// resolveDocID maps a 0 document id to the active document's id (or leaves it unchanged).
func (s *Session) resolveDocID(id doc.ID) doc.ID {
	if id == 0 {
		if d := s.ActiveDocument(); d != nil {
			return d.ID()
		}
	}
	return id
}

// displayOptionsView adapts the session's app display options to contract.DisplayOptions.
type displayOptionsView struct{ s *Session }

var _ contract.DisplayOptions = displayOptionsView{}

func (v displayOptionsView) DisplayQuality() types.DisplayQualityEnum {
	return v.s.displayOptions.DisplayQuality
}
func (v displayOptionsView) ViewTransitionTime() float64 {
	return v.s.displayOptions.ViewTransitionTime
}
func (v displayOptionsView) MinimumFrameRate() float64 { return v.s.displayOptions.MinimumFrameRate }
func (v displayOptionsView) HiddenLineDimmingPercent() int {
	return v.s.displayOptions.HiddenLineDimmingPercent
}
func (v displayOptionsView) EdgeColor() types.Color { return v.s.displayOptions.EdgeColor }
func (v displayOptionsView) NewWindowDisplayMode() types.DisplayModeEnum {
	return v.s.displayOptions.NewWindowDisplayMode
}
func (v displayOptionsView) NewWindowProjection() types.ProjectionTypeEnum {
	return v.s.displayOptions.NewWindowProjection
}
func (v displayOptionsView) BackFaceCulling() types.BackFaceCullingEnum {
	return v.s.displayOptions.BackFaceCulling
}
func (v displayOptionsView) UseRayTracing() bool { return v.s.displayOptions.UseRayTracing }
func (v displayOptionsView) RayTracingQuality() types.RayTracingQualityEnum {
	return v.s.displayOptions.RayTracingQuality
}
func (v displayOptionsView) Shaded() contract.ShadedDisplayOptions {
	return shadedOptionsView{v.s.displayOptions.Shaded}
}
func (v displayOptionsView) Wireframe() contract.WireframeDisplayOptions {
	return wireframeOptionsView{v.s.displayOptions.Wireframe}
}

// shadedOptionsView / wireframeOptionsView adapt the display sub-options.
type shadedOptionsView struct{ o display.ShadedOptions }

var _ contract.ShadedDisplayOptions = shadedOptionsView{}

func (v shadedOptionsView) EdgeDisplay() bool      { return v.o.EdgeDisplay }
func (v shadedOptionsView) EdgeColor() types.Color { return v.o.EdgeColor }
func (v shadedOptionsView) Silhouettes() bool      { return v.o.Silhouettes }
func (v shadedOptionsView) TransparencyType() types.TransparencyTypeEnum {
	return v.o.TransparencyType
}

type wireframeOptionsView struct{ o display.WireframeOptions }

var _ contract.WireframeDisplayOptions = wireframeOptionsView{}

func (v wireframeOptionsView) DepthDimming() bool      { return v.o.DepthDimming }
func (v wireframeOptionsView) Silhouettes() bool       { return v.o.Silhouettes }
func (v wireframeOptionsView) DimmedHiddenEdges() bool { return v.o.DimmedHiddenEdges }
