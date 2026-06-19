// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/drawing"

// DrawingViewEditTool re-opens an existing view's settings in the property dialog and applies
// them on OK (mirrors the part feature edit-in-place flow). It edits working copies of the
// view's fields and writes them back on Commit, so Cancel leaves the view untouched. A base
// view edits orientation/style/scale/centre; a projected view edits direction/centre (its
// orientation/scale/style follow its base).
type DrawingViewEditTool struct {
	dialogTool
	views     *drawing.DrawingViews
	view      *drawing.DrawingView
	projected bool
	// working copies
	orientation int
	style       int
	direction   int
	scale       float64
	centerX     float64
	centerY     float64
}

func newDrawingViewEditTool(views *drawing.DrawingViews, view *drawing.DrawingView) *DrawingViewEditTool {
	x, y := view.CenterMM()
	return &DrawingViewEditTool{
		views: views, view: view, projected: view.IsProjected(),
		orientation: baseViewOrientationIndex(view.Orientation()),
		style:       viewStyleIndex(view.Style()),
		direction:   projectionDirectionIndex(view.Direction()),
		scale:       view.Scale(), centerX: x, centerY: y,
	}
}

func (t *DrawingViewEditTool) Name() string    { return "Edit View" }
func (t *DrawingViewEditTool) CanCommit() bool { return true }

// Commit writes the edited settings back to the view and re-projects it.
func (t *DrawingViewEditTool) Commit(s *Session) error {
	var err error
	if t.projected {
		err = t.views.EditProjected(t.view.Name(),
			projectionDirections[clampIndex(t.direction, len(projectionDirections))].dir, t.centerX, t.centerY)
	} else {
		err = t.views.EditBase(t.view.Name(),
			baseViewOrientations[clampIndex(t.orientation, len(baseViewOrientations))].orientation,
			viewStyleChoices[clampIndex(t.style, len(viewStyleChoices))].style, t.scale, t.centerX, t.centerY)
	}
	if err != nil {
		return err
	}
	if d := s.ActiveDocument(); d != nil {
		d.MarkDirty()
	}
	return nil
}

// Params exposes the editable fields; a projected view shows its direction, a base view its
// orientation/style/scale. Both expose the sheet position.
func (t *DrawingViewEditTool) Params() ToolParams {
	pos := []FloatParam{
		{"Center X (mm)", func() float64 { return t.centerX }, func(v float64) { t.centerX = v }},
		{"Center Y (mm)", func() float64 { return t.centerY }, func(v float64) { t.centerY = v }},
	}
	if t.projected {
		return ToolParams{
			Choices: []ChoiceParam{{Label: "Direction", Options: labelsOf(len(projectionDirections), func(i int) string { return projectionDirections[i].label }),
				Get: func() int { return t.direction }, Set: func(i int) { t.direction = i }}},
			Floats: pos,
		}
	}
	return ToolParams{
		Choices: []ChoiceParam{
			{Label: "Orientation", Options: labelsOf(len(baseViewOrientations), func(i int) string { return baseViewOrientations[i].label }),
				Get: func() int { return t.orientation }, Set: func(i int) { t.orientation = i }},
			{Label: "Style", Options: labelsOf(len(viewStyleChoices), func(i int) string { return viewStyleChoices[i].label }),
				Get: func() int { return t.style }, Set: func(i int) { t.style = i }},
		},
		Floats: append([]FloatParam{{"Scale", func() float64 { return t.scale }, func(v float64) { t.scale = v }}}, pos...),
	}
}

// index helpers map a model enum value back to its dropdown index.
func baseViewOrientationIndex(o interface{ String() string }) int {
	for i, c := range baseViewOrientations {
		if c.orientation.String() == o.String() {
			return i
		}
	}
	return 0
}

func viewStyleIndex(st interface{ String() string }) int {
	for i, c := range viewStyleChoices {
		if c.style.String() == st.String() {
			return i
		}
	}
	return 0
}

func projectionDirectionIndex(d interface{ String() string }) int {
	for i, c := range projectionDirections {
		if c.dir.String() == d.String() {
			return i
		}
	}
	return 0
}
