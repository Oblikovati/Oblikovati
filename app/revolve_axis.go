// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// The Revolve tool's axis of revolution, as a SELECTION (#2018).
//
// Before this, the panel offered an origin-axis combo and nothing else, so the axis was the one
// input the user could not point at. Worse, it LIED: a pre-selected sketch centerline silently
// outranks the combo in addRevolve, so the panel read "Y Axis" while the feature spun about the
// centerline. Gathering every axis source into one value is what lets the panel name the axis the
// feature will actually use, and lets a pick replace it.

// revolveAxisChoice is the axis a revolve will spin about. At most one selection is live at a
// time — a picked sketch line, a picked work axis, or "the profile sketch's own centerline" (the
// auto mode a revolve saved before #2018 carries) — and with none the origin axis named by ref
// applies, which is why ref always holds a valid reference.
type revolveAxisChoice struct {
	ref    feature.WorkRef   // origin-axis fallback; the panel's quick-pick writes it
	work   *feature.WorkAxis // a work axis picked in the viewport or the browser
	line   *sketch.Line      // a picked / pre-selected sketch line (usually a centerline)
	lineSk *sketch.Sketch    // that line's sketch, for its plane
	auto   bool              // resolve the profile sketch's lone centerline at recompute
}

// pickLine selects a sketch line (centerline or not — the model revolves about either) as the
// axis, replacing whatever was selected before.
func (a *revolveAxisChoice) pickLine(line *sketch.Line, sk *sketch.Sketch) {
	a.work, a.auto = nil, false
	a.line, a.lineSk = line, sk
}

// pickWork selects a work axis as the axis, replacing whatever was selected before.
func (a *revolveAxisChoice) pickWork(w *feature.WorkAxis) {
	a.line, a.lineSk, a.auto = nil, nil, false
	a.work = w
}

// setAuto selects the profile sketch's own centerline, resolved every recompute.
func (a *revolveAxisChoice) setAuto() {
	a.work, a.line, a.lineSk = nil, nil, nil
	a.auto = true
}

// clear drops the selection, returning the revolve to the origin axis named by ref.
func (a *revolveAxisChoice) clear() {
	a.work, a.line, a.lineSk, a.auto = nil, nil, nil, false
}

// selected reports whether something was picked, i.e. whether ref is being overridden.
func (a *revolveAxisChoice) selected() bool { return a.line != nil || a.work != nil || a.auto }

// name is the chip caption: what the revolve will spin about, in the user's words.
func (a *revolveAxisChoice) name() string {
	switch {
	case a.line != nil && a.line.IsCenterline():
		return "Centerline"
	case a.line != nil:
		return "Sketch Line"
	case a.work != nil:
		return a.work.Name()
	case a.auto:
		return "Sketch Centerline"
	}
	return originAxisLabel(a.ref)
}

// OriginAxisChoice is one entry of the axis quick-pick offered by the Revolve and Coil panels:
// an origin axis and the caption to show for it.
type OriginAxisChoice struct {
	Label string
	Ref   feature.WorkRef
}

// originAxisChoices is the quick-pick list, shared by the panels and by the axis chip's caption
// so the same axis is never named two different things.
var originAxisChoices = []OriginAxisChoice{
	{"X Axis", feature.OriginXAxis},
	{"Y Axis", feature.OriginYAxis},
	{"Z Axis", feature.OriginZAxis},
}

// OriginAxisChoices returns the origin axes a feature panel offers as a quick-pick, in X/Y/Z
// order.
//
//	for _, a := range app.OriginAxisChoices() { … } // "X Axis", "Y Axis", "Z Axis"
func OriginAxisChoices() []OriginAxisChoice { return originAxisChoices }

// originAxisLabel names an origin axis, falling back to the raw reference for a work axis that
// carries a key but is not one of the three origin axes.
func originAxisLabel(ref feature.WorkRef) string {
	for _, a := range originAxisChoices {
		if a.Ref == ref {
			return a.Label
		}
	}
	return string(ref)
}

// resolve turns the choice into the work axis a revolve definition needs, or reports that the
// selection is a sketch line / the auto centerline (which the definition carries in its own
// fields instead). ok is false when the origin-axis fallback names an axis the part lacks.
func (a *revolveAxisChoice) resolve(part axisOwner) (*feature.WorkAxis, bool) {
	if a.work != nil {
		return a.work, true
	}
	return part.WorkGeometry().AxisByRef(a.ref)
}

// axisOwner is the slice of a part component definition the axis choice needs — resolving a
// reference key — so the choice does not depend on the whole definition.
type axisOwner interface {
	WorkGeometry() *feature.WorkGeometry
}
