// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Flat-pattern development (M13-F04, #377). The flat pattern is the manufacturable
// unfolding of the folded sheet: the base plate with each edge flange laid out as a
// coplanar tab, extending from its bend line by the developed length (the rule's bend
// allowance plus the flange's straight run). Building all pieces in the base sketch plane at
// the gauge yields one watertight flat solid whose extents grow with the K-factor exactly as
// the bend allowance does — the acceptance criterion for the flat pattern.
//
// This increment develops edge flanges/hems (the common tray/bracket topology). Mid-sheet
// bend/fold lines (which split the base into hinged regions) and stacked flange-on-flange
// chains develop differently and are a follow-up; compdef.Unfold passes only the edge bends
// it can place.

// FlatTab is one edge bend developed into the base plane: its bend line (A→B, in base-plane
// 2D) and the tab that extends Outward (a unit direction) by the developed Length. Angle is
// carried through for the fold-line annotation.
type FlatTab struct {
	A, B    math.Point2
	Outward math.Vector2
	Length  float64
	Angle   float64
}

// FlatBendLine is one fold line in the flat pattern: the segment (base-plane 2D) and the
// bend angle — what a DXF export draws on the bend layer.
type FlatBendLine struct {
	A, B  math.Point2
	Angle float64
}

// FlatPattern is the developed flat: the flat solid, its fold lines, the 2D extents, and the
// gauge. Body is one watertight solid (the base plate unioned with every tab).
type FlatPattern struct {
	Body      *topo.Body
	Bends     []FlatBendLine
	Extents   math.Box2d
	Thickness float64
}

// BuildFlatPattern thickens the base profile in its plane, unions each tab on as a coplanar
// plate, and reports the fold lines and the 2D extents (the footprint in the base plane).
//
// Example:
//
//	fp, err := BuildFlatPattern(face.Sketch, 0, 0.1, tabs) // tabs from compdef.Unfold
//	width, height := fp.Extents.Diagonal().X, fp.Extents.Diagonal().Y
func BuildFlatPattern(baseSketch *sketch.Sketch, baseProfile int, thickness float64, tabs []FlatTab) (*FlatPattern, error) {
	if thickness <= 0 {
		return nil, fmt.Errorf("flat pattern: thickness must be positive, got %g", thickness)
	}
	profiles, err := resolveClosedProfiles(baseSketch, []int{baseProfile}, "flat pattern base")
	if err != nil {
		return nil, err
	}
	plane := baseSketch.Plane()
	sp := span{near: 0, far: thickness}
	body := buildProfilePrisms(profiles, plane, sp, 0, "FlatPattern")
	fp := &FlatPattern{Thickness: thickness}
	for _, tab := range tabs {
		if body, err = appendTab(body, plane, sp, tab); err != nil {
			return nil, err
		}
		fp.Bends = append(fp.Bends, FlatBendLine{A: tab.A, B: tab.B, Angle: tab.Angle})
	}
	fp.Body = body
	fp.Extents = flatExtents(body, plane)
	return fp, nil
}

// appendTab unions one developed tab plate onto the running flat body. The tab is the quad
// bounded by the bend line (A→B) and that line shifted Outward by the developed length.
func appendTab(body *topo.Body, plane sketch.Plane, sp span, tab FlatTab) (*topo.Body, error) {
	offset := tab.Outward.Scale(tab.Length)
	poly := []math.Point2{tab.A, tab.B, tab.B.TranslateBy(offset), tab.A.TranslateBy(offset)}
	plate := buildPrism(poly, plane, sp, 0, "FlatPattern")
	merged, err := combine([]*topo.Body{body}, plate, ops.Join)
	if err != nil {
		return nil, err
	}
	if len(merged) != 1 {
		return nil, fmt.Errorf("flat pattern: tab union produced %d bodies, want 1 connected flat", len(merged))
	}
	return merged[0], nil
}

// flatExtents is the 2D bounding box of the flat body's footprint in the base plane —
// computed from the body's vertices so it captures the base plate and every tab.
func flatExtents(body *topo.Body, plane sketch.Plane) math.Box2d {
	box := math.EmptyBox2d()
	for _, v := range body.Vertices() {
		box = box.ExtendPoint(plane.ToSketch(v.Point()))
	}
	return box
}
