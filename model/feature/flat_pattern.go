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
// 2D) and the developed Length the tab extends from that line. The outward direction is NOT
// carried here — it is derived in BuildFlatPattern from the base geometry (perpendicular to
// the bend line, away from the base), because the bend's 3D fold direction can leave the base
// plane (a flange folded off a bottom edge folds in −Z) and so cannot be projected into it.
// Angle is carried through for the fold-line annotation.
type FlatTab struct {
	A, B     math.Point2
	Length   float64
	Angle    float64
	FoldDown bool // the bend folds toward the back ⇒ a bend-down fold line
}

// FlatBendLine is one fold line in the flat pattern: the segment (base-plane 2D), the bend
// angle, and the fold direction — what a DXF export draws on the bend-up/bend-down layers.
type FlatBendLine struct {
	A, B     math.Point2
	Angle    float64
	FoldDown bool
}

// FlatPunch is one punch's representation in the flat pattern: its closed outline (base-plane
// 2D) and a token naming the punch/tool, drawn on the punch layer for the CAM programmer. It is
// an annotation — a marker, not a body hole — so it represents a punch even where the flat solid
// does not yet carry the cut.
type FlatPunch struct {
	Outline []math.Point2
	Token   string
}

// FlatPattern is the developed flat: the flat solid, its fold lines, punch representations, the
// 2D extents, the gauge, and the base plane the flat lies in (so callers can project the body to
// 2D). Body is one watertight solid (the base plate unioned with every tab).
type FlatPattern struct {
	Body      *topo.Body
	Bends     []FlatBendLine
	Punches   []FlatPunch
	Extents   math.Box2d
	Thickness float64
	Plane     sketch.Plane
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
	centroid := polygonCentroid(profiles[0].OuterLoop().Polygon())
	fp := &FlatPattern{Thickness: thickness}
	for _, tab := range tabs {
		if body, err = appendTab(body, plane, sp, tab, centroid); err != nil {
			return nil, err
		}
		fp.Bends = append(fp.Bends, FlatBendLine{A: tab.A, B: tab.B, Angle: tab.Angle, FoldDown: tab.FoldDown})
	}
	fp.Body = body
	fp.Extents = flatExtents(body, plane)
	fp.Plane = plane
	return fp, nil
}

// appendTab unions one developed tab plate onto the running flat body. The tab is the quad
// bounded by the bend line (A→B) and that line shifted outward by the developed length, where
// outward is the in-plane perpendicular pointing away from the base (the bend opens the flange
// onto the far side of its bend line).
func appendTab(body *topo.Body, plane sketch.Plane, sp span, tab FlatTab, baseCentroid math.Point2) (*topo.Body, error) {
	offset := tabOutward(tab.A, tab.B, baseCentroid).Scale(tab.Length)
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

// tabOutward is the unit in-plane direction the flat tab extends: perpendicular to the bend
// line (a→b), oriented away from the base centroid so the developed flange lands outside the
// base rather than overlapping it.
func tabOutward(a, b, centroid math.Point2) math.Vector2 {
	edge := a.VectorTo(b)
	perp := math.V2(-edge.Y, edge.X)
	if l := perp.Length(); l > 0 {
		perp = perp.Scale(1 / l)
	}
	if centroid.VectorTo(a.Midpoint(b)).Dot(perp) < 0 {
		perp = perp.Negate()
	}
	return perp
}

// polygonCentroid is the average of a polygon's vertices — enough to orient tabs away from a
// convex base.
func polygonCentroid(poly []math.Point2) math.Point2 {
	if len(poly) == 0 {
		return math.P2(0, 0)
	}
	var sx, sy float64
	for _, p := range poly {
		sx += float64(p.X)
		sy += float64(p.Y)
	}
	n := float64(len(poly))
	return math.P2(sx/n, sy/n)
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
