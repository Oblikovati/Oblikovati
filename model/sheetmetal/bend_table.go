// SPDX-License-Identifier: GPL-2.0-only

package sheetmetal

import (
	stdmath "math"
	"sort"
)

// BendTable is a measured bend-allowance table: rows of (angle, radius, thickness →
// allowance) captured from shop tests for a material. Lookups select the rows whose radius
// and thickness match (within tolerance), then linearly interpolate the allowance by bend
// angle; outside the sampled angle range the nearest row's value is held. This is the
// honest minimum — a single material/gauge characterised across angles — and is the input
// most bend tables actually carry; multi-gauge tables stack independent BendTables.
type BendTable struct {
	rows []BendTableRow
}

// BendTableRow is one measured sample: at this bend angle (radians), inside radius and
// material thickness (database units, cm), the developed neutral-axis length is Allowance.
type BendTableRow struct {
	Angle     float64
	Radius    float64
	Thickness float64
	Allowance float64
}

// matchTol is the radius/thickness match tolerance (cm) — 1 µm, tight enough that distinct
// gauges never alias yet loose enough to absorb floating-point noise in stored geometry.
const matchTol = 1e-4

// NewBendTable returns a table over the given rows (copied, then sorted by angle so lookups
// can binary-search and interpolate).
func NewBendTable(rows []BendTableRow) *BendTable {
	cp := append([]BendTableRow(nil), rows...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].Angle < cp[j].Angle })
	return &BendTable{rows: cp}
}

// Rows returns the table's sample rows (sorted by angle).
func (t *BendTable) Rows() []BendTableRow { return t.rows }

// BendAllowance returns the interpolated allowance for a bend at angle/radius/thickness, or
// ok=false when no row matches the radius+thickness so the caller falls back to K-factor.
func (t *BendTable) BendAllowance(angle, radius, thickness float64) (float64, bool) {
	matches := t.rowsMatching(radius, thickness)
	if len(matches) == 0 {
		return 0, false
	}
	return interpolateByAngle(matches, angle), true
}

// rowsMatching returns the rows whose radius and thickness equal the query within matchTol.
func (t *BendTable) rowsMatching(radius, thickness float64) []BendTableRow {
	var out []BendTableRow
	for _, r := range t.rows {
		if stdmath.Abs(r.Radius-radius) <= matchTol && stdmath.Abs(r.Thickness-thickness) <= matchTol {
			out = append(out, r)
		}
	}
	return out
}

// interpolateByAngle linearly interpolates the allowance at angle across angle-sorted rows,
// holding the end value outside the sampled range. rows is non-empty and sorted by angle.
func interpolateByAngle(rows []BendTableRow, angle float64) float64 {
	if angle <= rows[0].Angle {
		return rows[0].Allowance
	}
	last := rows[len(rows)-1]
	if angle >= last.Angle {
		return last.Allowance
	}
	for i := 1; i < len(rows); i++ {
		hi := rows[i]
		if angle <= hi.Angle {
			lo := rows[i-1]
			span := hi.Angle - lo.Angle
			if span == 0 {
				return lo.Allowance
			}
			f := (angle - lo.Angle) / span
			return lo.Allowance + f*(hi.Allowance-lo.Allowance)
		}
	}
	return last.Allowance
}
