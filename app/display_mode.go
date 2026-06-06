// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati/api/types"

	"oblikovati/renderer"
)

// DisplayModeEnum is the public viewport display mode (Inventor's DisplayModeEnum), defined
// once in the Apache-2.0 contract and aliased here so call sites read app.DisplayModeEnum
// (ADR-0018). It maps one-to-one onto the renderer's internal VisualStyle.
type DisplayModeEnum = types.DisplayModeEnum

// displayModeToStyle / styleToDisplayMode are the bijection between the public display mode and
// the renderer's VisualStyle. The 8707 alias (HiddenEdgeRendering ==
// ShadedWithHiddenEdgesRendering) collapses to a single key/value.
var displayModeToStyle = map[types.DisplayModeEnum]renderer.VisualStyle{
	types.WireframeRendering:                renderer.Wireframe,
	types.ShadedWithHiddenEdgesRendering:    renderer.ShadedWithHiddenEdges, // 8707
	types.ShadedRendering:                   renderer.Shaded,
	types.RealisticRendering:                renderer.Realistic,
	types.ShadedWithEdgesRendering:          renderer.ShadedWithEdges,
	types.WireframeNoHiddenEdges:            renderer.WireframeVisibleOnly,
	types.WireframeWithHiddenEdgesRendering: renderer.WireframeWithHiddenEdges,
	types.MonochromeRendering:               renderer.Monochrome,
	types.WatercolorRendering:               renderer.Watercolor,
	types.IllustrationRendering:             renderer.Illustration,
	types.TechnicalIllustrationRendering:    renderer.TechnicalIllustration,
}

var styleToDisplayMode = map[renderer.VisualStyle]types.DisplayModeEnum{
	renderer.Wireframe:                types.WireframeRendering,
	renderer.ShadedWithHiddenEdges:    types.ShadedWithHiddenEdgesRendering,
	renderer.Shaded:                   types.ShadedRendering,
	renderer.Realistic:                types.RealisticRendering,
	renderer.ShadedWithEdges:          types.ShadedWithEdgesRendering,
	renderer.WireframeVisibleOnly:     types.WireframeNoHiddenEdges,
	renderer.WireframeWithHiddenEdges: types.WireframeWithHiddenEdgesRendering,
	renderer.Monochrome:               types.MonochromeRendering,
	renderer.Watercolor:               types.WatercolorRendering,
	renderer.Illustration:             types.IllustrationRendering,
	renderer.TechnicalIllustration:    types.TechnicalIllustrationRendering,
}

// DisplayMode returns the public display mode for the session's active visual style.
func (s *Session) DisplayMode() DisplayModeEnum {
	return styleToDisplayMode[s.visualStyle]
}

// SetDisplayMode switches the viewport to a public display mode, erroring on an unknown value
// (so a bad add-in request is rejected, not silently mapped to a default).
func (s *Session) SetDisplayMode(m DisplayModeEnum) error {
	style, ok := displayModeToStyle[m]
	if !ok {
		return fmt.Errorf("app: unknown display mode %d (%s)", int32(m), m)
	}
	s.SetVisualStyle(style)
	return nil
}
