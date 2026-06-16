// SPDX-License-Identifier: GPL-2.0-only

package feature

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
