// SPDX-License-Identifier: GPL-2.0-only

package sheetmetal

// Bend is one recorded bend in a sheet-metal part's history, carrying the unfold values the
// flat pattern (M13-F04, #377) develops it by. The architecture requires every bend to
// record its unfold parameters so the flat develops at the correct length; this is that
// record, projected from the part's feature history rather than detected from the faceted
// geometry. Lengths are in database units (cm); Angle is the swept bend angle in radians.
type Bend struct {
	Feature   string  // the feature that introduced the bend (e.g. "Flange1")
	Angle     float64 // swept bend angle (radians)
	Radius    float64 // inside bend radius (cm)
	Thickness float64 // material thickness at the bend (cm)
	Allowance float64 // developed neutral-axis arc length the flat must include (cm)
	Deduction float64 // setback subtracted from the two outside flange lengths (cm)
}

// NewBend records a bend, computing its allowance and deduction from the rule's unfold
// method so the values are fixed to the rule active at projection time.
//
// Example:
//
//	b := sheetmetal.NewBend("Flange1", math.Pi/2, 0.2, 0.1, rule.Unfold())
//	flatLen := legA + legB + b.Allowance // develop two legs through one 90° bend
func NewBend(feature string, angle, radius, thickness float64, m UnfoldMethod) Bend {
	return Bend{
		Feature:   feature,
		Angle:     angle,
		Radius:    radius,
		Thickness: thickness,
		Allowance: m.BendAllowance(angle, radius, thickness),
		Deduction: m.BendDeduction(angle, radius, thickness),
	}
}
