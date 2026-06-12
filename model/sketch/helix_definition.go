// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Helical curve definitions (M06-F09, Oblikovati/Oblikovati#624): beyond the
// constant shape of M22-F04, a helix may carry a variable shape — a row table
// where pitch/diameter change per station — plus start/end transition
// conditions. Flat ends compile into extra stations: the transition sweep
// blends the pitch linearly down to zero, then the flat sweep continues at
// zero pitch, so the realized curve is exact, not a visual approximation.

// HelixRow is one shape station. Diameter/Pitch are lengths (cm; zero means
// "interpolate between the neighboring specified rows"); exactly one of
// Height (cm from the base) or Revolution (turns from the start) positions
// the row, matching the definition's shape kind.
type HelixRow struct {
	Diameter   float64
	Pitch      float64
	Height     float64
	Revolution float64
}

// HelixEndCondition is the treatment of one helix end. The zero value is a
// natural end; a flat end carries its transition/flat sweep angles (radians).
type HelixEndCondition struct {
	Kind            types.HelixEndKind
	TransitionAngle float64
	FlatAngle       float64
}

// flat reports whether the end appends transition/flat sweeps.
func (c HelixEndCondition) flat() bool { return c.Kind == types.HelixEndFlat }

// HelixDefinition is a helical curve's stored definition: the shape kind, an
// optional variable row table, and the two end conditions.
type HelixDefinition struct {
	ShapeKind types.HelicalShapeDefinitionKind
	Rows      []HelixRow
	Start     HelixEndCondition
	End       HelixEndCondition
}

// Variable reports a row-table definition.
func (d *HelixDefinition) Variable() bool { return len(d.Rows) > 0 }

// Definition returns the entity's stored definition. A helix created before
// definitions existed reads as a constant pitch-and-revolution shape.
func (h *HelicalCurve3D) Definition() *HelixDefinition {
	if h.definition == nil {
		h.definition = &HelixDefinition{ShapeKind: types.HelixShapePitchRevolution}
	}
	return h.definition
}

// SetConstantShape redefines the helix with constant pitch/growth/turns and
// drops any variable rows; the curve regenerates from the new values.
func (h *HelicalCurve3D) SetConstantShape(kind types.HelicalShapeDefinitionKind, pitch, radialPerTurn, turns float64) error {
	if turns <= 0 {
		return fmt.Errorf("helix %d needs a positive turn count, got %g", h.id, turns)
	}
	h.AxialPerTurn, h.RadialPerTurn, h.Turns = pitch, radialPerTurn, turns
	def := h.Definition()
	def.ShapeKind, def.Rows = kind, nil
	return nil
}

// SetVariableShape redefines the helix with a station row table. Zero
// diameters/pitches interpolate between the neighboring specified rows; rows
// are positioned by Revolution (or by Height, converted via the trapezoid
// pitch integral). The entity's start radius and turn count follow the table.
func (h *HelicalCurve3D) SetVariableShape(kind types.HelicalShapeDefinitionKind, rows []HelixRow) error {
	if len(rows) < 2 {
		return fmt.Errorf("helix %d needs >= 2 shape rows, got %d", h.id, len(rows))
	}
	filled := fillInterpolatedRows(rows)
	stations, err := stationsFromRows(filled)
	if err != nil {
		return fmt.Errorf("helix %d: %w", h.id, err)
	}
	h.StartRadius = math.Scalar(stations[0].Radius)
	h.Turns = stations[len(stations)-1].Turn
	h.AxialPerTurn = stations[0].Pitch
	def := h.Definition()
	def.ShapeKind, def.Rows = kind, filled
	return nil
}

// SetEndConditions stores the start/end treatments (nil keeps the current).
func (h *HelicalCurve3D) SetEndConditions(start, end *HelixEndCondition) {
	def := h.Definition()
	if start != nil {
		def.Start = *start
	}
	if end != nil {
		def.End = *end
	}
}

// fillInterpolatedRows resolves zero Diameter/Pitch entries by linear
// interpolation between the nearest specified neighbors (ends clamp).
func fillInterpolatedRows(rows []HelixRow) []HelixRow {
	out := append([]HelixRow(nil), rows...)
	fillField(out, func(r *HelixRow) *float64 { return &r.Diameter })
	fillField(out, func(r *HelixRow) *float64 { return &r.Pitch })
	return out
}

// fillField interpolates one row field's zeros in place (by row index — the
// rows are stations, evenly meaningful in table order).
func fillField(rows []HelixRow, field func(*HelixRow) *float64) {
	for i := range rows {
		if *field(&rows[i]) != 0 {
			continue
		}
		prev, next := -1, -1
		for j := i - 1; j >= 0; j-- {
			if *field(&rows[j]) != 0 {
				prev = j
				break
			}
		}
		for j := i + 1; j < len(rows); j++ {
			if *field(&rows[j]) != 0 {
				next = j
				break
			}
		}
		*field(&rows[i]) = interpolatedValue(rows, field, i, prev, next)
	}
}

// interpolatedValue blends the prev/next specified values by row index.
func interpolatedValue(rows []HelixRow, field func(*HelixRow) *float64, i, prev, next int) float64 {
	switch {
	case prev < 0 && next < 0:
		return 0
	case prev < 0:
		return *field(&rows[next])
	case next < 0:
		return *field(&rows[prev])
	default:
		f := float64(i-prev) / float64(next-prev)
		return *field(&rows[prev]) + (*field(&rows[next])-*field(&rows[prev]))*f
	}
}

// stationsFromRows converts shape rows to kernel stations. Revolution
// positions a row directly; a Height-positioned row inverts the trapezoid
// pitch integral in closed form: Δturn = 2·Δheight / (p₀ + p₁).
func stationsFromRows(rows []HelixRow) ([]geom.HelixStation, error) {
	out := make([]geom.HelixStation, len(rows))
	turn, height := 0.0, 0.0
	for i, r := range rows {
		if i > 0 {
			next, err := advanceStation(turn, height, out[i-1].Pitch, r)
			if err != nil {
				return nil, err
			}
			turn = next
		}
		out[i] = geom.HelixStation{Turn: turn, Radius: r.Diameter / 2, Pitch: r.Pitch}
		height = heightAtRow(height, out, i)
	}
	return out, nil
}

// advanceStation returns the next row's turn coordinate from its Revolution
// or Height position.
func advanceStation(turn, height, prevPitch float64, r HelixRow) (float64, error) {
	if r.Revolution != 0 {
		if r.Revolution <= turn {
			return 0, fmt.Errorf("shape rows must advance in revolutions (%g after %g)", r.Revolution, turn)
		}
		return r.Revolution, nil
	}
	if r.Height <= height {
		return 0, fmt.Errorf("shape rows must advance in height (%g after %g)", r.Height, height)
	}
	if prevPitch+r.Pitch <= 0 {
		return 0, fmt.Errorf("height-positioned rows need positive pitch (got %g + %g)", prevPitch, r.Pitch)
	}
	return turn + 2*(r.Height-height)/(prevPitch+r.Pitch), nil
}

// heightAtRow accumulates the trapezoid height through station i.
func heightAtRow(prev float64, stations []geom.HelixStation, i int) float64 {
	if i == 0 {
		return 0
	}
	a, b := stations[i-1], stations[i]
	return prev + (a.Pitch+b.Pitch)/2*(b.Turn-a.Turn)
}

// definitionStations compiles the definition into the kernel station table:
// the shape rows (or the constant shape as two stations), with flat ends
// appended/prepended as extra stations.
func (h *HelicalCurve3D) definitionStations() ([]geom.HelixStation, error) {
	def := h.Definition()
	var stations []geom.HelixStation
	if def.Variable() {
		s, err := stationsFromRows(def.Rows)
		if err != nil {
			return nil, err
		}
		stations = s
	} else {
		stations = []geom.HelixStation{
			{Turn: 0, Radius: float64(h.StartRadius), Pitch: h.AxialPerTurn},
			{Turn: h.Turns, Radius: float64(h.StartRadius) + h.RadialPerTurn*h.Turns, Pitch: h.AxialPerTurn},
		}
	}
	stations = appendFlatEnd(stations, def.End)
	stations = prependFlatStart(stations, def.Start)
	return stations, nil
}

// appendFlatEnd adds the end transition (pitch → 0) and flat (pitch 0)
// sweeps after the last station.
func appendFlatEnd(stations []geom.HelixStation, c HelixEndCondition) []geom.HelixStation {
	if !c.flat() {
		return stations
	}
	last := stations[len(stations)-1]
	turn := last.Turn
	if c.TransitionAngle > 0 {
		turn += c.TransitionAngle / (2 * stdmath.Pi)
		stations = append(stations, geom.HelixStation{Turn: turn, Radius: last.Radius, Pitch: 0})
	}
	if c.FlatAngle > 0 {
		turn += c.FlatAngle / (2 * stdmath.Pi)
		stations = append(stations, geom.HelixStation{Turn: turn, Radius: last.Radius, Pitch: 0})
	}
	return stations
}

// prependFlatStart mirrors appendFlatEnd before the first station, shifting
// every turn coordinate so the table still starts at 0.
func prependFlatStart(stations []geom.HelixStation, c HelixEndCondition) []geom.HelixStation {
	if !c.flat() {
		return stations
	}
	first := stations[0]
	var lead []geom.HelixStation
	turn := 0.0
	if c.FlatAngle > 0 {
		lead = append(lead, geom.HelixStation{Turn: turn, Radius: first.Radius, Pitch: 0})
		turn += c.FlatAngle / (2 * stdmath.Pi)
	}
	if c.TransitionAngle > 0 {
		lead = append(lead, geom.HelixStation{Turn: turn, Radius: first.Radius, Pitch: 0})
		turn += c.TransitionAngle / (2 * stdmath.Pi)
	}
	out := append([]geom.HelixStation(nil), lead...)
	for _, s := range stations {
		out = append(out, geom.HelixStation{Turn: s.Turn + turn, Radius: s.Radius, Pitch: s.Pitch})
	}
	return out
}

// HelixDefinitionView adapts an entity to api/contract.HelicalCurveDefinition.
type HelixDefinitionView struct{ H *HelicalCurve3D }

func (v HelixDefinitionView) ShapeKind() types.HelicalShapeDefinitionKind {
	return v.H.Definition().ShapeKind
}
func (v HelixDefinitionView) Variable() bool               { return v.H.Definition().Variable() }
func (v HelixDefinitionView) RowCount() int                { return len(v.H.Definition().Rows) }
func (v HelixDefinitionView) Clockwise() bool              { return v.H.Clockwise }
func (v HelixDefinitionView) StartEnd() types.HelixEndKind { return endKind(v.H.Definition().Start) }
func (v HelixDefinitionView) EndEnd() types.HelixEndKind   { return endKind(v.H.Definition().End) }
func endKind(c HelixEndCondition) types.HelixEndKind {
	if c.Kind == 0 {
		return types.HelixEndNatural
	}
	return c.Kind
}
