// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"
	stdmath "math"

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
	fp, err := feature.BuildFlatPattern(def.Sketch, def.ProfileIndex, d.sheetMetal.Thickness(), d.flatTabs())
	if err != nil {
		return nil, err
	}
	fp.Punches = d.flatPunches(def.Sketch.Plane())
	return fp, nil
}

// flatPunches develops every healthy punch feature whose sketch is coplanar with the base into a
// flat punch representation (its outline + the feature name as the token). A punch on a flange
// develops through the fold and is a follow-up, so it is skipped here.
func (d *PartComponentDefinition) flatPunches(basePlane sketch.Plane) []feature.FlatPunch {
	fs := d.features
	var out []feature.FlatPunch
	for i := 0; i < fs.Count(); i++ {
		pf := fs.Item(i)
		punch, ok := pf.Definition().(*feature.SheetMetalPunchFeature)
		if !ok || pf.Suppressed() || !pf.Health().OK() {
			continue
		}
		out = append(out, developPunch(pf, punch, basePlane)...)
	}
	return out
}

// developPunch projects each closed profile of a coplanar punch into the base plane as a flat
// punch outline tagged with the feature name.
func developPunch(pf *feature.PartFeature, punch *feature.SheetMetalPunchFeature, basePlane sketch.Plane) []feature.FlatPunch {
	sk := punch.Definition().Sketch
	if stdmath.Abs(sk.Plane().Normal().AsVector().Dot(basePlane.Normal().AsVector())) < 0.999 {
		return nil // not coplanar: a flange punch, developed through the fold — a follow-up
	}
	profs := sk.Profiles()
	out := make([]feature.FlatPunch, 0, profs.Count())
	for j := 0; j < profs.Count(); j++ {
		outline := projectToPlane(profs.Item(j).OuterLoop().Polygon(), sk.Plane(), basePlane)
		out = append(out, punchResult(outline, pf.Name(), punch.Definition()))
	}
	return out
}

// punchResult fills one developed punch's placement data (#1963): where the tool goes, how it is
// turned, which side it comes from, and how deep. A nil depth is a punch clean through, which is
// reported as HAVING no depth rather than as a depth of zero.
func punchResult(outline []math.Point2, token string, def *feature.SheetMetalPunchDefinition) feature.FlatPunch {
	p := feature.FlatPunch{
		Outline:     outline,
		Token:       token,
		Position:    polygonCentroid2(outline),
		Angle:       longestRunAngle(outline),
		DirectionUp: def.Direction != feature.NegativeDir,
	}
	if def.Depth != nil {
		p.HasDepth, p.Depth = true, def.Depth()
	}
	return p
}

// polygonCentroid2 averages a closed outline's vertices — where the punch tool is placed.
func polygonCentroid2(pts []math.Point2) math.Point2 {
	if len(pts) == 0 {
		return math.P2(0, 0)
	}
	var sx, sy math.Scalar
	for _, p := range pts {
		sx, sy = sx+p.X, sy+p.Y
	}
	n := math.Scalar(len(pts))
	return math.Point2{X: sx / n, Y: sy / n}
}

// longestRunAngle reports how the punch is TURNED in the flat: the direction of the outline's
// longest straight run, which is the edge a rectangular or slotted tool is aligned to. A circular
// punch has no meaningful orientation and its many short chords average out to whichever is
// longest, which is harmless because every direction is equivalent for it.
func longestRunAngle(pts []math.Point2) float64 {
	best, angle := 0.0, 0.0
	for i := range pts {
		a, b := pts[i], pts[(i+1)%len(pts)]
		if d := float64(a.DistanceTo(b)); d > best {
			best, angle = d, stdmath.Atan2(float64(b.Y-a.Y), float64(b.X-a.X))
		}
	}
	return angle
}

// projectToPlane maps a loop's 2D points from one plane's frame into another's (lift to model
// space, drop back into the target plane) — for a coplanar punch this is its outline in base 2D.
func projectToPlane(poly []math.Point2, from, to sketch.Plane) []math.Point2 {
	out := make([]math.Point2, len(poly))
	for i, p := range poly {
		out[i] = to.ToSketch(from.ToModel(p))
	}
	return out
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
	return feature.FlatTab{
		A:        plane.ToSketch(p.AxisStart),
		B:        plane.ToSketch(p.AxisEnd),
		Length:   d.sheetMetal.Unfold().BendAllowance(p.Angle, p.Radius, p.Thickness) + p.Length,
		Angle:    p.Angle,
		FoldDown: p.FoldDown,
	}, true
}
