// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"math"

	"oblikovati.org/api/types"
	"oblikovati.org/model/drawing"
)

// baseViewOrientations is the orientation dropdown for the Base View tool (index → orientation).
var baseViewOrientations = []struct {
	label       string
	orientation types.BaseViewOrientation
}{
	{"Front", types.BaseViewFront}, {"Top", types.BaseViewTop}, {"Right", types.BaseViewRight},
	{"Back", types.BaseViewBack}, {"Left", types.BaseViewLeft}, {"Bottom", types.BaseViewBottom},
	{"Isometric", types.BaseViewIso},
}

var viewStyleChoices = []struct {
	label string
	style types.DrawingViewStyle
}{
	{"Hidden Line", types.HiddenLineViewStyle}, {"Wireframe", types.WireframeViewStyle},
}

// DrawingPlacementTool is a tool that drops a drawing view where the user clicks on the sheet,
// with a cursor-follow preview. The head canvas, each frame, sets the placement to the cursor
// and draws PreviewCurves there; a left-click commits the tool (s.OK) at that position.
type DrawingPlacementTool interface {
	Tool
	// PreviewCurves returns the view's curves centred at the origin (to be drawn at the cursor).
	PreviewCurves(s *Session) []drawing.DrawingCurve
	// SetPlacement records the sheet position (millimetres) the view will be centred on.
	SetPlacement(xMM, yMM float64)
}

// BaseViewTool places a base view of the drawing's referenced model. The user picks
// orientation/style/scale in the dialog and clicks the sheet to drop the view; the preview
// follows the cursor.
type BaseViewTool struct {
	orientation int
	style       int
	scale       float64
	centerX     float64
	centerY     float64
	preview     []drawing.DrawingCurve
	previewKey  string // orientation/style/scale the cached preview was built for
}

// NewBaseViewTool starts on a front, hidden-line, 1:1 view.
func NewBaseViewTool() *BaseViewTool {
	return &BaseViewTool{scale: 1, centerX: 150, centerY: 150}
}

func (t *BaseViewTool) Name() string              { return "Base View" }
func (t *BaseViewTool) Start(*Session)            {}
func (t *BaseViewTool) Pick(*Session, Selectable) {}
func (t *BaseViewTool) CanCommit() bool           { return true }
func (t *BaseViewTool) Cancel(*Session)           {}

// SetPlacement records the cursor sheet position the view will be centred on.
func (t *BaseViewTool) SetPlacement(x, y float64) { t.centerX, t.centerY = x, y }

// PreviewCurves projects the chosen orientation/style/scale at the origin, caching until those
// options change, so the head can draw a cursor-follow preview without re-projecting per frame.
func (t *BaseViewTool) PreviewCurves(s *Session) []drawing.DrawingCurve {
	c, err := ActiveDrawing(s)
	if err != nil {
		return nil
	}
	key := fmt.Sprintf("%d/%d/%g", t.orientation, t.style, t.scale)
	if key != t.previewKey {
		t.preview, _ = c.Sheets().Active().Views().PreviewBase(
			baseViewOrientations[clampIndex(t.orientation, len(baseViewOrientations))].orientation,
			viewStyleChoices[clampIndex(t.style, len(viewStyleChoices))].style, t.scale)
		t.previewKey = key
	}
	return t.preview
}

// Commit projects the configured base view onto the active sheet.
func (t *BaseViewTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	_, err = c.Sheets().Active().Views().AddBase(drawing.BaseViewSpec{
		Orientation: baseViewOrientations[clampIndex(t.orientation, len(baseViewOrientations))].orientation,
		Style:       viewStyleChoices[clampIndex(t.style, len(viewStyleChoices))].style,
		Scale:       t.scale, CenterX: t.centerX, CenterY: t.centerY,
	})
	if err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the orientation/style/scale fields for the property dialog (the sheet
// position comes from the placement click, not a field).
func (t *BaseViewTool) Params() ToolParams {
	return ToolParams{
		Choices: []ChoiceParam{
			{Label: "Orientation", Options: labelsOf(len(baseViewOrientations), func(i int) string { return baseViewOrientations[i].label }),
				Get: func() int { return t.orientation }, Set: func(i int) { t.orientation = i }},
			{Label: "Style", Options: labelsOf(len(viewStyleChoices), func(i int) string { return viewStyleChoices[i].label }),
				Get: func() int { return t.style }, Set: func(i int) { t.style = i }},
		},
		Floats: []FloatParam{
			{"Scale", func() float64 { return t.scale }, func(v float64) { t.scale = v }},
		},
	}
}

// derivedViewTool is the shared state of the tools that place a view derived from a base view
// (projected/auxiliary/section): the candidate base views, the chosen one, the cursor placement
// and the cached cursor-follow preview. The concrete tools embed it and add their own option
// (direction / fold angle / cut orientation) plus PreviewCurves/Commit/Params.
type derivedViewTool struct {
	bases      []string
	baseIndex  int
	centerX    float64
	centerY    float64
	preview    []drawing.DrawingCurve
	previewKey string
}

// Start captures the base views the derived view can be built from.
func (t *derivedViewTool) Start(s *Session)          { t.bases = baseViewNames(s) }
func (t *derivedViewTool) Pick(*Session, Selectable) {}
func (t *derivedViewTool) Cancel(*Session)           {}

// CanCommit requires at least one base view to derive from.
func (t *derivedViewTool) CanCommit() bool { return len(t.bases) > 0 }

// SetPlacement records the cursor sheet position the derived view will be centred on.
func (t *derivedViewTool) SetPlacement(x, y float64) { t.centerX, t.centerY = x, y }

// parent returns the selected base view's name ("" when none are available).
func (t *derivedViewTool) parent() string {
	if len(t.bases) == 0 {
		return ""
	}
	return t.bases[clampIndex(t.baseIndex, len(t.bases))]
}

// baseChoice is the base-view dropdown the derived-view tools share in Params.
func (t *derivedViewTool) baseChoice(label string) ChoiceParam {
	return ChoiceParam{Label: label, Options: t.bases, Get: func() int { return t.baseIndex }, Set: func(i int) { t.baseIndex = i }}
}

var projectionDirections = []struct {
	label string
	dir   types.ProjectionDirection
}{
	{"Right", types.ProjectRight}, {"Left", types.ProjectLeft}, {"Up", types.ProjectUp}, {"Down", types.ProjectDown},
}

// ProjectedViewTool places a view projected from a base view in a chosen direction.
type ProjectedViewTool struct {
	derivedViewTool
	direction int
}

// NewProjectedViewTool creates the tool; its base-view list is captured on Start.
func NewProjectedViewTool() *ProjectedViewTool {
	return &ProjectedViewTool{derivedViewTool: derivedViewTool{centerX: 250, centerY: 150}}
}

func (t *ProjectedViewTool) Name() string { return "Projected View" }

func (t *ProjectedViewTool) dir() types.ProjectionDirection {
	return projectionDirections[clampIndex(t.direction, len(projectionDirections))].dir
}

// PreviewCurves projects the chosen base+direction at the origin, cached until they change.
func (t *ProjectedViewTool) PreviewCurves(s *Session) []drawing.DrawingCurve {
	parent := t.parent()
	if parent == "" {
		return nil
	}
	c, err := ActiveDrawing(s)
	if err != nil {
		return nil
	}
	key := fmt.Sprintf("%s/%d", parent, t.direction)
	if key != t.previewKey {
		t.preview, _ = c.Sheets().Active().Views().PreviewProjected(parent, t.dir())
		t.previewKey = key
	}
	return t.preview
}

// Commit projects from the selected base view in the chosen direction.
func (t *ProjectedViewTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	if t.parent() == "" {
		return fmt.Errorf("drawing: no base view to project from — add a base view first")
	}
	_, err = c.Sheets().Active().Views().AddProjected(drawing.ProjectedViewSpec{
		BaseView: t.parent(), Direction: t.dir(), CenterX: t.centerX, CenterY: t.centerY,
	})
	if err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the base-view and direction choices for the property dialog.
func (t *ProjectedViewTool) Params() ToolParams {
	return ToolParams{Choices: []ChoiceParam{
		t.baseChoice("Base View"),
		{Label: "Direction", Options: labelsOf(len(projectionDirections), func(i int) string { return projectionDirections[i].label }),
			Get: func() int { return t.direction }, Set: func(i int) { t.direction = i }},
	}}
}

// AuxiliaryViewTool places a view folded off a base view about a fold line at a chosen angle —
// it mirrors ProjectedViewTool but takes a fold angle (degrees) instead of a discrete direction.
type AuxiliaryViewTool struct {
	derivedViewTool
	foldDeg float64
}

// NewAuxiliaryViewTool creates the tool; its parent-view list is captured on Start.
func NewAuxiliaryViewTool() *AuxiliaryViewTool {
	return &AuxiliaryViewTool{derivedViewTool: derivedViewTool{centerX: 250, centerY: 250}}
}

func (t *AuxiliaryViewTool) Name() string { return "Auxiliary View" }

// PreviewCurves folds the chosen parent at the chosen angle at the origin, cached until they change.
func (t *AuxiliaryViewTool) PreviewCurves(s *Session) []drawing.DrawingCurve {
	parent := t.parent()
	if parent == "" {
		return nil
	}
	c, err := ActiveDrawing(s)
	if err != nil {
		return nil
	}
	key := fmt.Sprintf("%s/%g", parent, t.foldDeg)
	if key != t.previewKey {
		t.preview, _ = c.Sheets().Active().Views().PreviewAuxiliary(parent, t.foldDeg*math.Pi/180)
		t.previewKey = key
	}
	return t.preview
}

// Commit folds an auxiliary view off the selected parent at the chosen angle.
func (t *AuxiliaryViewTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	if t.parent() == "" {
		return fmt.Errorf("drawing: no base view to fold from — add a base view first")
	}
	_, err = c.Sheets().Active().Views().AddAuxiliary(drawing.AuxiliaryViewSpec{
		ParentView: t.parent(), FoldAngleRad: t.foldDeg * math.Pi / 180, CenterX: t.centerX, CenterY: t.centerY,
	})
	if err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the parent-view choice and fold angle for the property dialog.
func (t *AuxiliaryViewTool) Params() ToolParams {
	return ToolParams{
		Choices: []ChoiceParam{t.baseChoice("Parent View")},
		Floats:  []FloatParam{{"Fold Angle (deg)", func() float64 { return t.foldDeg }, func(v float64) { t.foldDeg = v }}},
	}
}

// sectionOrientations is the cut-line orientation choice for the Section View tool.
var sectionOrientations = []string{"Horizontal", "Vertical"}

// SectionViewTool cuts a section view through a base view's centre, along a horizontal or
// vertical section line; the preview follows the cursor and a click drops the section view.
// (Free-hand section lines are a follow-up; a centreline cut is the common case.)
type SectionViewTool struct {
	derivedViewTool
	orient int
}

// NewSectionViewTool creates the tool; its base-view list is captured on Start.
func NewSectionViewTool() *SectionViewTool {
	return &SectionViewTool{derivedViewTool: derivedViewTool{centerX: 150, centerY: 250}}
}

func (t *SectionViewTool) Name() string { return "Section View" }

// sectionLineOn returns the cut line (sheet mm) through the named base view's centre, spanning
// its bounds along the chosen orientation; ok is false when the view has no geometry yet.
func (t *SectionViewTool) sectionLineOn(s *Session, parent string) (x1, y1, x2, y2 float64, ok bool) {
	c, err := ActiveDrawing(s)
	if err != nil {
		return 0, 0, 0, 0, false
	}
	v, found := c.Sheets().Active().Views().ByName(parent)
	if !found {
		return 0, 0, 0, 0, false
	}
	minX, minY, maxX, maxY, has := v.BoundsMM()
	if !has {
		return 0, 0, 0, 0, false
	}
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	if t.orient == 1 { // vertical
		return cx, minY - 5, cx, maxY + 5, true
	}
	return minX - 5, cy, maxX + 5, cy, true // horizontal
}

// PreviewCurves projects the section through the chosen base+orientation, cached until they change.
func (t *SectionViewTool) PreviewCurves(s *Session) []drawing.DrawingCurve {
	parent := t.parent()
	if parent == "" {
		return nil
	}
	key := fmt.Sprintf("%s/%d", parent, t.orient)
	if key != t.previewKey {
		t.preview = nil
		if x1, y1, x2, y2, ok := t.sectionLineOn(s, parent); ok {
			c, _ := ActiveDrawing(s)
			t.preview, _ = c.Sheets().Active().Views().PreviewSection(parent, x1, y1, x2, y2)
		}
		t.previewKey = key
	}
	return t.preview
}

// Commit cuts the section view through the selected base view.
func (t *SectionViewTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	parent := t.parent()
	if parent == "" {
		return fmt.Errorf("drawing: no base view to section — add a base view first")
	}
	x1, y1, x2, y2, ok := t.sectionLineOn(s, parent)
	if !ok {
		return fmt.Errorf("drawing: base view %q has no geometry to section", parent)
	}
	if _, err := c.Sheets().Active().Views().AddSection(drawing.SectionViewSpec{
		ParentView: parent, X1: x1, Y1: y1, X2: x2, Y2: y2, CenterX: t.centerX, CenterY: t.centerY,
	}); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the base-view and cut-orientation choices for the property dialog.
func (t *SectionViewTool) Params() ToolParams {
	return ToolParams{Choices: []ChoiceParam{
		t.baseChoice("Base View"),
		{Label: "Cut", Options: sectionOrientations, Get: func() int { return t.orient }, Set: func(i int) { t.orient = i }},
	}}
}

// DetailViewTool magnifies a circular region (centred on the parent view, radius a fraction of
// its size) at a chosen scale; the preview follows the cursor and a click drops the detail view.
// (A free-placed boundary is a follow-up; a centred region is the common case.)
type DetailViewTool struct {
	derivedViewTool
	scale float64
}

// NewDetailViewTool starts at 2× magnification.
func NewDetailViewTool() *DetailViewTool {
	return &DetailViewTool{derivedViewTool: derivedViewTool{centerX: 150, centerY: 250}, scale: 2}
}

func (t *DetailViewTool) Name() string { return "Detail View" }

// boundaryOn returns the detail circle (sheet mm) centred on the named parent view, with a
// radius covering ~40% of its larger dimension; ok is false when the view has no geometry yet.
func (t *DetailViewTool) boundaryOn(s *Session, parent string) (cx, cy, r float64, ok bool) {
	c, err := ActiveDrawing(s)
	if err != nil {
		return 0, 0, 0, false
	}
	v, found := c.Sheets().Active().Views().ByName(parent)
	if !found {
		return 0, 0, 0, false
	}
	minX, minY, maxX, maxY, has := v.BoundsMM()
	if !has {
		return 0, 0, 0, false
	}
	return (minX + maxX) / 2, (minY + maxY) / 2, 0.4 * maxf64(maxX-minX, maxY-minY), true
}

// PreviewCurves magnifies the chosen parent region at the origin, cached until parent/scale change.
func (t *DetailViewTool) PreviewCurves(s *Session) []drawing.DrawingCurve {
	parent := t.parent()
	if parent == "" {
		return nil
	}
	key := fmt.Sprintf("%s/%g", parent, t.scale)
	if key != t.previewKey {
		t.preview = nil
		if cx, cy, r, ok := t.boundaryOn(s, parent); ok {
			c, _ := ActiveDrawing(s)
			t.preview, _ = c.Sheets().Active().Views().PreviewDetail(parent, cx, cy, r, t.scale)
		}
		t.previewKey = key
	}
	return t.preview
}

// Commit magnifies the selected parent's centred region at the chosen scale.
func (t *DetailViewTool) Commit(s *Session) error {
	c, err := ActiveDrawing(s)
	if err != nil {
		return err
	}
	parent := t.parent()
	if parent == "" {
		return fmt.Errorf("drawing: no base view to detail — add a base view first")
	}
	cx, cy, r, ok := t.boundaryOn(s, parent)
	if !ok {
		return fmt.Errorf("drawing: base view %q has no geometry to detail", parent)
	}
	if _, err := c.Sheets().Active().Views().AddDetail(drawing.DetailViewSpec{
		ParentView: parent, BoundaryX: cx, BoundaryY: cy, RadiusMM: r, Scale: t.scale, CenterX: t.centerX, CenterY: t.centerY,
	}); err != nil {
		return err
	}
	s.ActiveDocument().MarkDirty()
	return nil
}

// Params exposes the base-view choice and detail scale for the property dialog.
func (t *DetailViewTool) Params() ToolParams {
	return ToolParams{
		Choices: []ChoiceParam{t.baseChoice("Base View")},
		Floats:  []FloatParam{{"Scale", func() float64 { return t.scale }, func(v float64) { t.scale = v }}},
	}
}

func maxf64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// baseViewNames lists the active sheet's base views (the candidates a projected view derives
// from); empty when no drawing is active.
func baseViewNames(s *Session) []string {
	c, err := ActiveDrawing(s)
	if err != nil {
		return nil
	}
	views := c.Sheets().Active().Views()
	var names []string
	for i := 0; i < views.Count(); i++ {
		if v := views.Item(i); !v.IsProjected() {
			names = append(names, v.Name())
		}
	}
	return names
}

func labelsOf(n int, at func(int) string) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = at(i)
	}
	return out
}

func clampIndex(i, n int) int {
	if i < 0 || i >= n {
		return 0
	}
	return i
}
