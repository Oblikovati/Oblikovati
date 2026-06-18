// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"strings"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/drawing"
)

// Drawing-annotation tools (M14-F02 #813): place a centre-of-gravity marker on a base view and a
// revision cloud on the sheet. The CoG tool is dialog-only (pick the view; the marker positions
// itself at the model's centre of mass); the revision-cloud tool drops a sized region at the
// cursor.

// CoGMarkerTool places a centre-of-gravity marker on a chosen base view.
type CoGMarkerTool struct {
	views     []string
	viewIndex int
}

// NewCoGMarkerTool creates the tool; its view list is captured on Start.
func NewCoGMarkerTool() *CoGMarkerTool { return &CoGMarkerTool{} }

func (t *CoGMarkerTool) Name() string              { return "Center of Gravity" }
func (t *CoGMarkerTool) Start(s *Session)          { t.views = baseViewNames(s) }
func (t *CoGMarkerTool) Pick(*Session, Selectable) {}
func (t *CoGMarkerTool) CanCommit() bool           { return len(t.views) > 0 }
func (t *CoGMarkerTool) Cancel(*Session)           {}

// Commit adds the marker on the selected view.
func (t *CoGMarkerTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	if len(t.views) == 0 {
		return fmt.Errorf("drawing: no view for a centre-of-gravity marker — add a base view first")
	}
	if _, err := c.Sheets().Active().Annotations().AddCoGMarker("", t.views[clampIndex(t.viewIndex, len(t.views))]); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the view choice for the property dialog.
func (t *CoGMarkerTool) Params() ToolParams {
	return ToolParams{Choices: []ChoiceParam{
		{Label: "View", Options: t.views, Get: func() int { return t.viewIndex }, Set: func(i int) { t.viewIndex = i }},
	}}
}

// CenterMarkTool places a centre mark (crosshair) at every circular edge's centre in a chosen base
// view — the auto centre-mark-all-holes action.
type CenterMarkTool struct {
	views     []string
	viewIndex int
}

// NewCenterMarkTool creates the tool; its view list is captured on Start.
func NewCenterMarkTool() *CenterMarkTool { return &CenterMarkTool{} }

func (t *CenterMarkTool) Name() string              { return "Center Mark" }
func (t *CenterMarkTool) Start(s *Session)          { t.views = baseViewNames(s) }
func (t *CenterMarkTool) Pick(*Session, Selectable) {}
func (t *CenterMarkTool) CanCommit() bool           { return len(t.views) > 0 }
func (t *CenterMarkTool) Cancel(*Session)           {}

// Commit centre-marks every circular edge in the selected view.
func (t *CenterMarkTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	if len(t.views) == 0 {
		return fmt.Errorf("drawing: no view for centre marks — add a base view first")
	}
	if _, err := c.Sheets().Active().Annotations().AddCenterMarks(t.views[clampIndex(t.viewIndex, len(t.views))]); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the view choice for the property dialog.
func (t *CenterMarkTool) Params() ToolParams {
	return ToolParams{Choices: []ChoiceParam{
		{Label: "View", Options: t.views, Get: func() int { return t.viewIndex }, Set: func(i int) { t.viewIndex = i }},
	}}
}

// CenterlineTool adds the horizontal+vertical dash-dot symmetry centerlines through a chosen base
// view's centre.
type CenterlineTool struct {
	views     []string
	viewIndex int
}

// NewCenterlineTool creates the tool; its view list is captured on Start.
func NewCenterlineTool() *CenterlineTool { return &CenterlineTool{} }

func (t *CenterlineTool) Name() string              { return "Centerline" }
func (t *CenterlineTool) Start(s *Session)          { t.views = baseViewNames(s) }
func (t *CenterlineTool) Pick(*Session, Selectable) {}
func (t *CenterlineTool) CanCommit() bool           { return len(t.views) > 0 }
func (t *CenterlineTool) Cancel(*Session)           {}

// Commit adds the centerlines on the selected view.
func (t *CenterlineTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	if len(t.views) == 0 {
		return fmt.Errorf("drawing: no view for centerlines — add a base view first")
	}
	if _, err := c.Sheets().Active().Annotations().AddCenterlines("", t.views[clampIndex(t.viewIndex, len(t.views))]); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the view choice for the property dialog.
func (t *CenterlineTool) Params() ToolParams {
	return ToolParams{Choices: []ChoiceParam{
		{Label: "View", Options: t.views, Get: func() int { return t.viewIndex }, Set: func(i int) { t.viewIndex = i }},
	}}
}

// fcfCharacteristics indexes the feature control frame's Characteristic dropdown.
var fcfCharacteristics = []struct {
	label string
	value types.GeometricCharacteristic
}{
	{"Position", types.CharacteristicPosition},
	{"Flatness", types.CharacteristicFlatness},
	{"Perpendicularity", types.CharacteristicPerpendicularity},
	{"Parallelism", types.CharacteristicParallelism},
	{"Straightness", types.CharacteristicStraightness},
	{"Circularity", types.CharacteristicCircularity},
	{"Concentricity", types.CharacteristicConcentricity},
	{"Angularity", types.CharacteristicAngularity},
}

// FeatureControlFrameTool drops a GD&T feature control frame at the cursor with a chosen
// characteristic, tolerance value and comma-separated datum references.
type FeatureControlFrameTool struct {
	charIndex        int
	tolerance        string
	datums           string
	centerX, centerY float64
}

// NewFeatureControlFrameTool starts on a position tolerance of 0.5.
func NewFeatureControlFrameTool() *FeatureControlFrameTool {
	return &FeatureControlFrameTool{tolerance: "0.5", centerX: 150, centerY: 150}
}

func (t *FeatureControlFrameTool) Name() string              { return "Feature Control Frame" }
func (t *FeatureControlFrameTool) Start(*Session)            {}
func (t *FeatureControlFrameTool) Pick(*Session, Selectable) {}
func (t *FeatureControlFrameTool) CanCommit() bool           { return t.tolerance != "" }
func (t *FeatureControlFrameTool) Cancel(*Session)           {}
func (t *FeatureControlFrameTool) SetPlacement(x, y float64) { t.centerX, t.centerY = x, y }

// Commit drops the frame at the placed point.
func (t *FeatureControlFrameTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	ch := fcfCharacteristics[clampIndex(t.charIndex, len(fcfCharacteristics))].value
	_, err = c.Sheets().Active().Annotations().AddFeatureControlFrame("", t.centerX, t.centerY, ch, t.tolerance, splitDatums(t.datums))
	if err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the characteristic, tolerance and datum-reference inputs.
func (t *FeatureControlFrameTool) Params() ToolParams {
	labels := make([]string, len(fcfCharacteristics))
	for i, c := range fcfCharacteristics {
		labels[i] = c.label
	}
	return ToolParams{
		Choices: []ChoiceParam{{Label: "Characteristic", Options: labels, Get: func() int { return t.charIndex }, Set: func(i int) { t.charIndex = i }}},
		Texts: []TextParam{
			{Label: "Tolerance", Get: func() string { return t.tolerance }, Set: func(v string) { t.tolerance = v }},
			{Label: "Datums (A,B,C)", Get: func() string { return t.datums }, Set: func(v string) { t.datums = v }},
		},
	}
}

// splitDatums turns a comma-separated datum string ("A, B,C") into trimmed, non-empty references.
func splitDatums(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if d := strings.TrimSpace(part); d != "" {
			out = append(out, d)
		}
	}
	return out
}

// RevisionCloudTool drops a scalloped revision cloud of a chosen size at the cursor.
type RevisionCloudTool struct {
	width, height    float64
	centerX, centerY float64
	preview          []drawing.DrawingCurve
}

// NewRevisionCloudTool starts on a 60×40 mm cloud.
func NewRevisionCloudTool() *RevisionCloudTool {
	return &RevisionCloudTool{width: 60, height: 40, centerX: 150, centerY: 150}
}

func (t *RevisionCloudTool) Name() string              { return "Revision Cloud" }
func (t *RevisionCloudTool) Start(*Session)            {}
func (t *RevisionCloudTool) Pick(*Session, Selectable) {}
func (t *RevisionCloudTool) CanCommit() bool           { return true }
func (t *RevisionCloudTool) Cancel(*Session)           {}
func (t *RevisionCloudTool) SetPlacement(x, y float64) { t.centerX, t.centerY = x, y }

// PreviewCurves outlines the cloud region (a plain rectangle) at the origin to follow the cursor;
// the committed cloud is scalloped.
func (t *RevisionCloudTool) PreviewCurves(*Session) []drawing.DrawingCurve {
	w, h := gmath.Scalar(t.width/2), gmath.Scalar(t.height/2)
	corners := [4]gmath.Point2{gmath.P2(-w, -h), gmath.P2(w, -h), gmath.P2(w, h), gmath.P2(-w, h)}
	t.preview = t.preview[:0]
	for i := 0; i < 4; i++ {
		t.preview = append(t.preview, drawing.DrawingCurve{A: corners[i], B: corners[(i+1)%4], Visible: true})
	}
	return t.preview
}

// Commit drops the cloud over the placed region.
func (t *RevisionCloudTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	if _, err := c.Sheets().Active().Annotations().AddRevisionCloud("", t.centerX-t.width/2, t.centerY-t.height/2, t.width, t.height, ""); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the cloud width/height.
func (t *RevisionCloudTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{
		{"Width (mm)", func() float64 { return t.width }, func(v float64) { t.width = v }},
		{"Height (mm)", func() float64 { return t.height }, func(v float64) { t.height = v }},
	}}
}
