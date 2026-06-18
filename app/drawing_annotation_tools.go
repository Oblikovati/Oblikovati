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

// RevisionTableTool drops a revision table seeded with one revision row (revision/date/description)
// at the cursor; the API/bridge can place multi-row tables.
type RevisionTableTool struct {
	revision, date, description string
	centerX, centerY            float64
}

// NewRevisionTableTool seeds the first revision row (rev A).
func NewRevisionTableTool() *RevisionTableTool {
	return &RevisionTableTool{revision: "A", description: "Initial release", centerX: 250, centerY: 60}
}

func (t *RevisionTableTool) Name() string              { return "Revision Table" }
func (t *RevisionTableTool) Start(*Session)            {}
func (t *RevisionTableTool) Pick(*Session, Selectable) {}
func (t *RevisionTableTool) CanCommit() bool           { return t.revision != "" }
func (t *RevisionTableTool) Cancel(*Session)           {}
func (t *RevisionTableTool) SetPlacement(x, y float64) { t.centerX, t.centerY = x, y }

// Commit drops the one-row revision table at the placed point.
func (t *RevisionTableTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	rows := []drawing.RevisionRow{{Revision: t.revision, Date: t.date, Description: t.description}}
	if _, err := c.Sheets().Active().Annotations().AddRevisionTable("", t.centerX, t.centerY, rows); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the first revision row's fields.
func (t *RevisionTableTool) Params() ToolParams {
	return ToolParams{Texts: []TextParam{
		{Label: "Revision", Get: func() string { return t.revision }, Set: func(v string) { t.revision = v }},
		{Label: "Date", Get: func() string { return t.date }, Set: func(v string) { t.date = v }},
		{Label: "Description", Get: func() string { return t.description }, Set: func(v string) { t.description = v }},
	}}
}

// RevisionTagTool drops a revision tag (a triangle holding a revision letter) at the cursor.
type RevisionTagTool struct {
	revision         string
	centerX, centerY float64
}

// NewRevisionTagTool starts on revision A.
func NewRevisionTagTool() *RevisionTagTool {
	return &RevisionTagTool{revision: "A", centerX: 150, centerY: 150}
}

func (t *RevisionTagTool) Name() string              { return "Revision Tag" }
func (t *RevisionTagTool) Start(*Session)            {}
func (t *RevisionTagTool) Pick(*Session, Selectable) {}
func (t *RevisionTagTool) CanCommit() bool           { return t.revision != "" }
func (t *RevisionTagTool) Cancel(*Session)           {}
func (t *RevisionTagTool) SetPlacement(x, y float64) { t.centerX, t.centerY = x, y }

// Commit drops the revision tag at the placed point.
func (t *RevisionTagTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	if _, err := c.Sheets().Active().Annotations().AddRevisionTag("", t.centerX, t.centerY, t.revision); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the revision letter input.
func (t *RevisionTagTool) Params() ToolParams {
	return ToolParams{Texts: []TextParam{
		{Label: "Revision", Get: func() string { return t.revision }, Set: func(v string) { t.revision = v }},
	}}
}

// DatumFeatureTool drops a GD&T datum feature symbol (a lettered box + datum triangle) at the
// cursor.
type DatumFeatureTool struct {
	letter           string
	centerX, centerY float64
}

// NewDatumFeatureTool starts on datum letter A.
func NewDatumFeatureTool() *DatumFeatureTool {
	return &DatumFeatureTool{letter: "A", centerX: 150, centerY: 150}
}

func (t *DatumFeatureTool) Name() string              { return "Datum Feature" }
func (t *DatumFeatureTool) Start(*Session)            {}
func (t *DatumFeatureTool) Pick(*Session, Selectable) {}
func (t *DatumFeatureTool) CanCommit() bool           { return t.letter != "" }
func (t *DatumFeatureTool) Cancel(*Session)           {}
func (t *DatumFeatureTool) SetPlacement(x, y float64) { t.centerX, t.centerY = x, y }

// Commit drops the datum symbol at the placed point.
func (t *DatumFeatureTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	if _, err := c.Sheets().Active().Annotations().AddDatumFeature("", t.centerX, t.centerY, t.letter); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the datum letter input.
func (t *DatumFeatureTool) Params() ToolParams {
	return ToolParams{Texts: []TextParam{
		{Label: "Datum letter", Get: func() string { return t.letter }, Set: func(v string) { t.letter = v }},
	}}
}

// surfaceVariants indexes the surface-texture Material removal dropdown.
var surfaceVariants = []struct {
	label string
	value types.MaterialRemoval
}{
	{"Any", types.MaterialRemovalAny},
	{"Required (machined)", types.MaterialRemovalRequired},
	{"Prohibited (as-cast)", types.MaterialRemovalProhibited},
}

// SurfaceTextureTool drops an ISO 1302 surface texture symbol at the cursor with a roughness value
// and material-removal variant.
type SurfaceTextureTool struct {
	roughness        string
	variantIndex     int
	centerX, centerY float64
}

// NewSurfaceTextureTool starts on a 1.6 roughness, any material removal.
func NewSurfaceTextureTool() *SurfaceTextureTool {
	return &SurfaceTextureTool{roughness: "1.6", centerX: 150, centerY: 150}
}

func (t *SurfaceTextureTool) Name() string              { return "Surface Texture" }
func (t *SurfaceTextureTool) Start(*Session)            {}
func (t *SurfaceTextureTool) Pick(*Session, Selectable) {}
func (t *SurfaceTextureTool) CanCommit() bool           { return true }
func (t *SurfaceTextureTool) Cancel(*Session)           {}
func (t *SurfaceTextureTool) SetPlacement(x, y float64) { t.centerX, t.centerY = x, y }

// Commit drops the surface texture symbol at the placed point.
func (t *SurfaceTextureTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	variant := surfaceVariants[clampIndex(t.variantIndex, len(surfaceVariants))].value
	if _, err := c.Sheets().Active().Annotations().AddSurfaceTexture("", t.centerX, t.centerY, t.roughness, variant); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the roughness value and material-removal variant.
func (t *SurfaceTextureTool) Params() ToolParams {
	labels := make([]string, len(surfaceVariants))
	for i, v := range surfaceVariants {
		labels[i] = v.label
	}
	return ToolParams{
		Texts:   []TextParam{{Label: "Roughness", Get: func() string { return t.roughness }, Set: func(v string) { t.roughness = v }}},
		Choices: []ChoiceParam{{Label: "Material removal", Options: labels, Get: func() int { return t.variantIndex }, Set: func(i int) { t.variantIndex = i }}},
	}
}

// PartsListTool drops a parts list table (sourced from the referenced assembly's BOM) at the
// cursor (the table's top-left corner).
type PartsListTool struct {
	centerX, centerY float64
}

// NewPartsListTool starts placing near the top-left of the sheet.
func NewPartsListTool() *PartsListTool { return &PartsListTool{centerX: 40, centerY: 260} }

func (t *PartsListTool) Name() string              { return "Parts List" }
func (t *PartsListTool) Start(*Session)            {}
func (t *PartsListTool) Pick(*Session, Selectable) {}
func (t *PartsListTool) CanCommit() bool           { return true }
func (t *PartsListTool) Cancel(*Session)           {}
func (t *PartsListTool) SetPlacement(x, y float64) { t.centerX, t.centerY = x, y }

// Commit drops the parts list at the placed point.
func (t *PartsListTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	if _, err := c.Sheets().Active().Annotations().AddPartsList("", t.centerX, t.centerY); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// BalloonTool drops a balloon (a circled parts-list item number) at the cursor, with an optional
// leader to a previously-picked point.
type BalloonTool struct {
	item             int
	centerX, centerY float64
	leaderX, leaderY float64
}

// NewBalloonTool starts on item 1, no leader.
func NewBalloonTool() *BalloonTool { return &BalloonTool{item: 1, centerX: 150, centerY: 150} }

func (t *BalloonTool) Name() string              { return "Balloon" }
func (t *BalloonTool) Start(*Session)            {}
func (t *BalloonTool) Pick(*Session, Selectable) {}
func (t *BalloonTool) CanCommit() bool           { return t.item > 0 }
func (t *BalloonTool) Cancel(*Session)           {}
func (t *BalloonTool) SetPlacement(x, y float64) { t.centerX, t.centerY = x, y }

// Commit drops the balloon at the placed point.
func (t *BalloonTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	if _, err := c.Sheets().Active().Annotations().AddBalloon("", t.centerX, t.centerY, t.item, t.leaderX, t.leaderY); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the item number and leader-target inputs.
func (t *BalloonTool) Params() ToolParams {
	return ToolParams{
		Ints: []IntParam{{Label: "Item", Get: func() int { return t.item }, Set: func(v int) { t.item = v }}},
		Floats: []FloatParam{
			{"Leader X (mm)", func() float64 { return t.leaderX }, func(v float64) { t.leaderX = v }},
			{"Leader Y (mm)", func() float64 { return t.leaderY }, func(v float64) { t.leaderY = v }},
		},
	}
}

// HoleTableTool drops a hole table for a chosen base view at the cursor (the table's top-left
// corner), listing the view's circular edges with X/Y from a datum and diameter.
type HoleTableTool struct {
	views            []string
	viewIndex        int
	centerX, centerY float64
}

// NewHoleTableTool creates the tool; its base-view list is captured on Start.
func NewHoleTableTool() *HoleTableTool { return &HoleTableTool{centerX: 250, centerY: 260} }

func (t *HoleTableTool) Name() string              { return "Hole Table" }
func (t *HoleTableTool) Start(s *Session)          { t.views = baseViewNames(s) }
func (t *HoleTableTool) Pick(*Session, Selectable) {}
func (t *HoleTableTool) CanCommit() bool           { return len(t.views) > 0 }
func (t *HoleTableTool) Cancel(*Session)           {}
func (t *HoleTableTool) SetPlacement(x, y float64) { t.centerX, t.centerY = x, y }

// Commit drops the hole table for the selected view at the placed point.
func (t *HoleTableTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	if len(t.views) == 0 {
		return fmt.Errorf("drawing: no base view for a hole table — add a base view first")
	}
	if _, err := c.Sheets().Active().Annotations().AddHoleTable("", t.views[clampIndex(t.viewIndex, len(t.views))], t.centerX, t.centerY); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the base-view choice for the property dialog.
func (t *HoleTableTool) Params() ToolParams {
	return ToolParams{Choices: []ChoiceParam{
		{Label: "View", Options: t.views, Get: func() int { return t.viewIndex }, Set: func(i int) { t.viewIndex = i }},
	}}
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
