// SPDX-License-Identifier: GPL-2.0-only

package feature

import "oblikovati.org/math"

// BendLineage is implemented by sheet-metal features that introduce one or more bends. The
// flat pattern (M13-F04, #377) reads it to develop each bend: the architecture requires
// every bend to record its unfold parameters (angle + radius) so the flat develops at the
// correct length. Recording lives on the feature definition — the feature history IS the
// bend graph — rather than being detected from the faceted bend geometry after the fact.
type BendLineage interface {
	// BendSpecs reports the bends this feature introduces, given the material thickness.
	// Thickness is passed because some bends derive their radius from the gauge (a closed
	// hem folds at a fraction of the thickness). A feature with no live bend returns nil.
	BendSpecs(thickness float64) []BendSpec
}

// BendSpec is the raw geometry of one bend before it is developed: the swept angle and the
// inside bend radius. A non-positive Radius means "this feature does not override the
// radius" — the projecting layer fills in the rule's default bend radius. The developed
// values (allowance/deduction) are computed by the rule, not here, so the spec stays a
// pure geometric statement independent of the unfold method.
type BendSpec struct {
	Angle  float64 // swept bend angle (radians)
	Radius float64 // inside bend radius (cm); <= 0 ⇒ use the rule's default
}

// BendPlacement is the resolved geometry of one edge bend, captured during the feature's
// recompute (when its edge and frame are in hand). The flat pattern lays the bend's flange
// out as a tab in the base plane, extending from the bend line (AxisStart→AxisEnd) along
// Outward by the developed length (the rule's bend allowance + Length). The allowance is
// filled in by the rule at unfold time, so the placement itself stays free of the unfold
// method — the same separation BendSpec keeps.
type BendPlacement struct {
	AxisStart, AxisEnd math.Point3      // the bend line — the picked edge's endpoints
	Outward            math.UnitVector3 // in-plane direction the flat tab extends (away from the sheet)
	Up                 math.UnitVector3 // fold normal: the bend-axis centre is the bend line + Up·Radius (M13-F04 develop)
	Angle, Radius      float64          // swept bend angle (radians) and inside radius (cm)
	Thickness, Length  float64          // material thickness and the flange's straight-run length (cm)
	FoldDown           bool             // the bend folds the material toward the back (a flipped flange) ⇒ a bend-down fold line
}

// PlacedBend is implemented by the edge-bend walls (flange, hem) that develop into a flat
// tab. Placement reports the resolved bend geometry from the last successful recompute; ok
// is false before the first recompute or after the feature went sick. The flat pattern reads
// it to lay each flange out in the base plane.
type PlacedBend interface {
	Placement() (BendPlacement, bool)
}

// PlacedBends is a feature that places MORE THAN ONE edge bend in a single feature — a multi-edge
// flange (#2071). The flat pattern lays out every one; a feature that implements both interfaces has
// its Placement be the first of these. A single-edge feature can implement only PlacedBend.
type PlacedBends interface {
	Placements() []BendPlacement
}
