// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Variable-pitch coils (M06-F09, Oblikovati/Oblikovati#624): the coil's helix
// rail may carry a pitch row table — pitch changes per revolution station —
// plus flat start/end conditions, the standard variable-pitch spring recipe.
// The rows compile into a kernel station table whose closed-form pitch
// integral places each profile section, so the solid is exact to the rail.

// CoilPitchRow is one pitch station: at Revolution turns from the start the
// rail advances Pitch per revolution. Revolution 0 marks the first row.
type CoilPitchRow struct {
	Pitch      float64
	Revolution float64
}

// CoilEndCondition is one end's treatment: a flat end appends a transition
// sweep (pitch blends to zero) and a flat sweep (zero pitch), angles in
// radians.
type CoilEndCondition struct {
	Flat            bool
	TransitionAngle float64
	FlatAngle       float64
}

// coilStations compiles the definition's rail into kernel stations: the
// constant pitch (or the row table), with flat ends appended as extra
// stations. Radius is irrelevant for the rail's rise; it stays zero.
func coilStations(def *CoilDefinition, revolutions float64) ([]geom.HelixStation, error) {
	stations, err := coilShapeStations(def, revolutions)
	if err != nil {
		return nil, err
	}
	stations = coilAppendFlat(stations, def.EndEnd)
	stations = coilPrependFlat(stations, def.StartEnd)
	return stations, nil
}

// coilShapeStations is the rail before end treatment.
func coilShapeStations(def *CoilDefinition, revolutions float64) ([]geom.HelixStation, error) {
	if len(def.PitchRows) == 0 {
		pitch := callOrZero(def.Pitch)
		return []geom.HelixStation{
			{Turn: 0, Pitch: pitch},
			{Turn: revolutions, Pitch: pitch},
		}, nil
	}
	if len(def.PitchRows) < 2 {
		return nil, fmt.Errorf("coil: a variable pitch table needs >= 2 rows, got %d", len(def.PitchRows))
	}
	out := make([]geom.HelixStation, len(def.PitchRows))
	for i, r := range def.PitchRows {
		out[i] = geom.HelixStation{Turn: r.Revolution, Pitch: r.Pitch}
	}
	return out, nil
}

// coilAppendFlat appends the end transition (pitch → 0) and flat sweeps.
func coilAppendFlat(stations []geom.HelixStation, c CoilEndCondition) []geom.HelixStation {
	if !c.Flat {
		return stations
	}
	last := stations[len(stations)-1]
	turn := last.Turn
	if c.TransitionAngle > 0 {
		turn += c.TransitionAngle / (2 * stdmath.Pi)
		stations = append(stations, geom.HelixStation{Turn: turn, Pitch: 0})
	}
	if c.FlatAngle > 0 {
		turn += c.FlatAngle / (2 * stdmath.Pi)
		stations = append(stations, geom.HelixStation{Turn: turn, Pitch: 0})
	}
	return stations
}

// coilPrependFlat mirrors coilAppendFlat before the first station, shifting
// turns so the table still starts at zero.
func coilPrependFlat(stations []geom.HelixStation, c CoilEndCondition) []geom.HelixStation {
	if !c.Flat {
		return stations
	}
	shift := (c.TransitionAngle + c.FlatAngle) / (2 * stdmath.Pi)
	var out []geom.HelixStation
	turn := 0.0
	if c.FlatAngle > 0 {
		out = append(out, geom.HelixStation{Turn: turn, Pitch: 0})
		turn += c.FlatAngle / (2 * stdmath.Pi)
	}
	if c.TransitionAngle > 0 {
		out = append(out, geom.HelixStation{Turn: turn, Pitch: 0})
	}
	for _, s := range stations {
		out = append(out, geom.HelixStation{Turn: s.Turn + shift, Pitch: s.Pitch})
	}
	return out
}

// coilRail validates the definition and compiles its rail: the
// rise-per-angle closure plus the rail's total turn count.
func coilRail(def *CoilDefinition) (func(angle float64) float64, float64, error) {
	revs := callOrZero(def.Revolutions)
	if revs <= 0 {
		return nil, 0, fmt.Errorf("coil: revolutions must be > 0, got %g", revs)
	}
	stations, err := coilStations(def, revs)
	if err != nil {
		return nil, 0, err
	}
	return coilRise(stations)
}

// coilRise returns the rise-per-angle function over the compiled stations
// (the trapezoid pitch integral), plus the rail's total turns.
func coilRise(stations []geom.HelixStation) (func(angle float64) float64, float64, error) {
	rail, err := geom.NewVariableHelix3d(
		math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), stations, false)
	if err != nil {
		return nil, 0, fmt.Errorf("coil: %w", err)
	}
	return func(angle float64) float64 {
		return rail.HeightAtTurn(angle / (2 * stdmath.Pi))
	}, rail.TotalTurns(), nil
}
