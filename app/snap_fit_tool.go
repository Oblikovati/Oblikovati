// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// SnapFitTool is the interactive Snap Fit command (#1076, plastic features): a cantilever hook
// sized purely by its beam and catch dimensions, added at the active sketch origin. It needs no
// picking, so it commits straight from its parameters, and exposes them through the generic
// property dialog.
type SnapFitTool struct {
	dialogTool
	length      float64
	width       float64
	thickness   float64
	catchLength float64
	catchHeight float64
	added       *feature.PartFeature
}

// NewSnapFitTool returns a snap-fit tool with a default cantilever hook (beam 5×2×1, catch 1×1).
func NewSnapFitTool() *SnapFitTool {
	return &SnapFitTool{length: 5, width: 2, thickness: 1, catchLength: 1, catchHeight: 1}
}

// Name implements [Tool].
func (t *SnapFitTool) Name() string { return "Snap Fit" }

// The beam and catch dimensions the property window drives (database units, all positive).
func (t *SnapFitTool) SetLength(v float64) { t.length = posOrKeep(v, t.length) }
func (t *SnapFitTool) Length() float64     { return t.length }
func (t *SnapFitTool) SetWidth(v float64)  { t.width = posOrKeep(v, t.width) }
func (t *SnapFitTool) Width() float64      { return t.width }
func (t *SnapFitTool) SetThickness(v float64) {
	t.thickness = posOrKeep(v, t.thickness)
}
func (t *SnapFitTool) Thickness() float64       { return t.thickness }
func (t *SnapFitTool) SetCatchLength(v float64) { t.catchLength = posOrKeep(v, t.catchLength) }
func (t *SnapFitTool) CatchLength() float64     { return t.catchLength }
func (t *SnapFitTool) SetCatchHeight(v float64) { t.catchHeight = posOrKeep(v, t.catchHeight) }
func (t *SnapFitTool) CatchHeight() float64     { return t.catchHeight }

// Params exposes the five snap-fit dimensions for the generic property dialog.
func (t *SnapFitTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{
		{Label: "Beam Length", Get: t.Length, Set: t.SetLength},
		{Label: "Beam Width", Get: t.Width, Set: t.SetWidth},
		{Label: "Beam Thickness", Get: t.Thickness, Set: t.SetThickness},
		{Label: "Catch Length", Get: t.CatchLength, Set: t.SetCatchLength},
		{Label: "Catch Height", Get: t.CatchHeight, Set: t.SetCatchHeight},
	}}
}

// CanCommit reports whether every dimension is positive.
func (t *SnapFitTool) CanCommit() bool {
	return t.length > 0 && t.width > 0 && t.thickness > 0 && t.catchLength > 0 && t.catchHeight > 0
}

// Commit adds the snap-fit hook to the active part and recomputes; a sick feature keeps the
// tool open by returning an error.
func (t *SnapFitTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	l, w, th, cl, ch := t.length, t.width, t.thickness, t.catchLength, t.catchHeight
	t.added = feature.NewPlasticFeatures(part.Features()).AddCantileverSnapFit(
		func() float64 { return l }, func() float64 { return w }, func() float64 { return th },
		func() float64 { return cl }, func() float64 { return ch })
	part.Recompute()
	s.recordEdit(part, "Snap Fit")
	if !t.added.Health().OK() {
		return errors.New("snap fit: " + t.added.Health().Reason)
	}
	return nil
}

// DraftFeature returns the unattached hook the viewport previews before commit.
func (t *SnapFitTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	l, w, th, cl, ch := t.length, t.width, t.thickness, t.catchLength, t.catchHeight
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return feature.NewPlasticFeatures(fs).AddCantileverSnapFit(
			func() float64 { return l }, func() float64 { return w }, func() float64 { return th },
			func() float64 { return cl }, func() float64 { return ch }), nil
	})
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *SnapFitTool) AddedFeature() *feature.PartFeature { return t.added }

// posOrKeep accepts v only when positive, otherwise keeps the prior value — guarding the
// dimension setters against a zero/negative entry that would make an invalid hook.
func posOrKeep(v, prior float64) float64 {
	if v > 0 {
		return v
	}
	return prior
}
