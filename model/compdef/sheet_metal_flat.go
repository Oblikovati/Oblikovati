// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// Unfold develops the folded sheet-metal part into its flat pattern (M13-F04, #377): the
// base plate with every edge flange/hem laid out as a coplanar tab, each extending from its
// bend line by the developed length (the rule's bend allowance plus the flange's straight
// run). The flat is a derived body — it recomputes from the current folded model and rule,
// so a thickness or K-factor edit changes the flat extents through the same allowance math.
//
// It errors when the part is not sheet metal or has no base Face to develop. Mid-sheet
// bend/fold lines and stacked flange chains are not yet placed (see flat_pattern.go); the
// developed tabs cover the common tray/bracket topology.
func (d *PartComponentDefinition) Unfold() (*feature.FlatPattern, error) {
	if d.sheetMetal == nil {
		return nil, fmt.Errorf("unfold: the active part is not a sheet-metal part")
	}
	base, ok := d.baseFace()
	if !ok {
		return nil, fmt.Errorf("unfold: the part has no base Face to develop")
	}
	def := base.Definition()
	return feature.BuildFlatPattern(def.Sketch, def.ProfileIndex, d.sheetMetal.Thickness(), d.flatTabs())
}

// baseFace returns the first sheet-metal base Face feature — the flat region the development
// is anchored on.
func (d *PartComponentDefinition) baseFace() (*feature.SheetMetalFaceFeature, bool) {
	fs := d.features
	for i := 0; i < fs.Count(); i++ {
		if face, ok := fs.Item(i).Definition().(*feature.SheetMetalFaceFeature); ok {
			return face, true
		}
	}
	return nil, false
}

// flatTabs develops every healthy, unsuppressed edge bend into a flat tab, projecting its
// recorded bend geometry into the base sketch plane and adding the rule's bend allowance to
// the flange's straight run.
func (d *PartComponentDefinition) flatTabs() []feature.FlatTab {
	base, ok := d.baseFace()
	if !ok {
		return nil
	}
	plane := base.Definition().Sketch.Plane()
	fs := d.features
	var tabs []feature.FlatTab
	for i := 0; i < fs.Count(); i++ {
		if tab, ok := d.developTab(fs.Item(i), plane); ok {
			tabs = append(tabs, tab)
		}
	}
	return tabs
}

// developTab develops one feature into a flat tab, or ok=false when it is not a healthy
// placed edge bend. It projects the recorded bend line and outward direction into the base
// plane and adds the rule's bend allowance to the flange's straight run.
func (d *PartComponentDefinition) developTab(pf *feature.PartFeature, plane sketch.Plane) (feature.FlatTab, bool) {
	if pf.Suppressed() || !pf.Health().OK() {
		return feature.FlatTab{}, false
	}
	placed, ok := pf.Definition().(feature.PlacedBend)
	if !ok {
		return feature.FlatTab{}, false
	}
	p, ok := placed.Placement()
	if !ok {
		return feature.FlatTab{}, false
	}
	out := p.Outward.AsVector()
	x, y := plane.XAxis().AsVector(), plane.YAxis().AsVector()
	return feature.FlatTab{
		A:       plane.ToSketch(p.AxisStart),
		B:       plane.ToSketch(p.AxisEnd),
		Outward: math.V2(out.Dot(x), out.Dot(y)),
		Length:  d.sheetMetal.Unfold().BendAllowance(p.Angle, p.Radius, p.Thickness) + p.Length,
		Angle:   p.Angle,
	}, true
}
