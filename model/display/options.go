// SPDX-License-Identifier: GPL-2.0-only

// Package display is the display options that parameterize the M23 display modes (M16-F07,
// #643): the application-level DisplayOptions and the per-document DisplaySettings (background,
// edges, ground plane, shadows). It is pure data — the renderer and persistence layers consume
// it; this package never imports them.
package display

import "oblikovati.org/api/types"

// ShadedOptions are the shaded-mode display sub-options (edge overlay, transparency rendering).
type ShadedOptions struct {
	EdgeDisplay      bool
	EdgeColor        types.Color
	Silhouettes      bool
	TransparencyType types.TransparencyTypeEnum
}

// WireframeOptions are the wireframe-mode display sub-options.
type WireframeOptions struct {
	DepthDimming      bool
	Silhouettes       bool
	DimmedHiddenEdges bool
}

// Options are the application-level display options — the preferences that parameterize the
// display modes (Inventor's DisplayOptions).
type Options struct {
	DisplayQuality           types.DisplayQualityEnum
	ViewTransitionTime       float64
	MinimumFrameRate         float64
	HiddenLineDimmingPercent int
	EdgeColor                types.Color
	NewWindowDisplayMode     types.DisplayModeEnum
	NewWindowProjection      types.ProjectionTypeEnum
	BackFaceCulling          types.BackFaceCullingEnum
	UseRayTracing            bool
	RayTracingQuality        types.RayTracingQualityEnum
	Shaded                   ShadedOptions
	Wireframe                WireframeOptions
}

// DefaultOptions are the out-of-the-box application display options.
func DefaultOptions() Options {
	edge := types.NewColor(30, 30, 34)
	return Options{
		DisplayQuality:           types.SmoothDisplayQuality,
		ViewTransitionTime:       0.4,
		MinimumFrameRate:         15,
		HiddenLineDimmingPercent: 50,
		EdgeColor:                edge,
		NewWindowDisplayMode:     types.ShadedWithEdgesRendering,
		NewWindowProjection:      types.OrthographicProjection,
		BackFaceCulling:          types.CullNone,
		UseRayTracing:            false,
		RayTracingQuality:        types.InteractiveRayTracingQuality,
		Shaded:                   ShadedOptions{EdgeDisplay: true, EdgeColor: edge, Silhouettes: false, TransparencyType: types.BlendingTransparency},
		Wireframe:                WireframeOptions{DepthDimming: true, Silhouettes: false, DimmedHiddenEdges: true},
	}
}
