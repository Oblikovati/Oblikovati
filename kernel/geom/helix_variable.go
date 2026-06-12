// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	"sort"

	"oblikovati.org/math"
)

// VariableHelix3d is a helix whose radius and pitch vary over a station table
// (M06-F09, Oblikovati/Oblikovati#624) — the standard recipe for
// variable-pitch springs. Stations give (turn, radius, pitch) rows; radius
// and pitch interpolate linearly in the turn coordinate between stations, so
// the axial height (the integral of pitch over turns) is piecewise quadratic
// and evaluated in closed form. Parameterized t ∈ [0, 1] over the whole turn
// range, like [Helix3d].
type VariableHelix3d struct {
	Origin    math.Point3
	Axis      math.UnitVector3
	RefDir    math.UnitVector3
	Clockwise bool
	stations  []HelixStation
	heights   []float64 // cumulative axial height at each station
}

// HelixStation is one shape row: at Turn revolutions from the start, the
// helix has this Radius, advancing Pitch per revolution.
type HelixStation struct {
	Turn   float64
	Radius float64
	Pitch  float64
}

// NewVariableHelix3d builds a variable helix through the stations (at least
// two, strictly increasing turns, the first at turn 0).
func NewVariableHelix3d(origin math.Point3, axis, refDir math.Vector3, stations []HelixStation, clockwise bool) (VariableHelix3d, error) {
	n, err := math.UnitVector3FromVector(axis)
	if err != nil {
		return VariableHelix3d{}, err
	}
	rows := append([]HelixStation(nil), stations...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Turn < rows[j].Turn })
	if err := validateStations(rows); err != nil {
		return VariableHelix3d{}, err
	}
	return VariableHelix3d{
		Origin: origin, Axis: n, RefDir: planarRef(n, refDir), Clockwise: clockwise,
		stations: rows, heights: cumulativeHeights(rows),
	}, nil
}

// validateStations enforces the table shape: >= 2 rows, first at turn 0,
// strictly increasing turns.
func validateStations(rows []HelixStation) error {
	if len(rows) < 2 {
		return fmt.Errorf("geom: a variable helix needs >= 2 stations, got %d", len(rows))
	}
	if rows[0].Turn != 0 {
		return fmt.Errorf("geom: the first helix station must sit at turn 0, got %g", rows[0].Turn)
	}
	for i := 1; i < len(rows); i++ {
		if rows[i].Turn <= rows[i-1].Turn {
			return fmt.Errorf("geom: helix stations must strictly increase in turns (%g then %g)",
				rows[i-1].Turn, rows[i].Turn)
		}
	}
	return nil
}

// cumulativeHeights integrates the piecewise-linear pitch over turns: each
// segment contributes the trapezoid (p₀+p₁)/2 · Δturns.
func cumulativeHeights(rows []HelixStation) []float64 {
	out := make([]float64, len(rows))
	for i := 1; i < len(rows); i++ {
		dTurn := rows[i].Turn - rows[i-1].Turn
		out[i] = out[i-1] + (rows[i-1].Pitch+rows[i].Pitch)/2*dTurn
	}
	return out
}

// Stations returns the (sorted) shape rows.
func (h VariableHelix3d) Stations() []HelixStation {
	return append([]HelixStation(nil), h.stations...)
}

// TotalTurns returns the last station's turn coordinate.
func (h VariableHelix3d) TotalTurns() float64 { return h.stations[len(h.stations)-1].Turn }

// segmentAt finds the station segment containing the turn coordinate.
func (h VariableHelix3d) segmentAt(turn float64) int {
	i := sort.Search(len(h.stations)-1, func(k int) bool { return h.stations[k+1].Turn >= turn })
	if i > len(h.stations)-2 {
		i = len(h.stations) - 2
	}
	return i
}

// radiusAt linearly interpolates the radius at the turn coordinate.
func (h VariableHelix3d) radiusAt(turn float64) float64 {
	i := h.segmentAt(turn)
	a, b := h.stations[i], h.stations[i+1]
	f := (turn - a.Turn) / (b.Turn - a.Turn)
	return a.Radius + (b.Radius-a.Radius)*f
}

// pitchAt linearly interpolates the pitch at the turn coordinate.
func (h VariableHelix3d) pitchAt(turn float64) float64 {
	i := h.segmentAt(turn)
	a, b := h.stations[i], h.stations[i+1]
	f := (turn - a.Turn) / (b.Turn - a.Turn)
	return a.Pitch + (b.Pitch-a.Pitch)*f
}

// heightAt returns the closed-form axial advance at the turn coordinate: the
// station's cumulative height plus the partial trapezoid into its segment.
func (h VariableHelix3d) heightAt(turn float64) float64 {
	i := h.segmentAt(turn)
	a := h.stations[i]
	d := turn - a.Turn
	return h.heights[i] + (a.Pitch+h.pitchAt(turn))/2*d
}

// angleAt returns the signed winding angle at parameter t.
func (h VariableHelix3d) angleAt(t float64) float64 {
	a := twoPi * h.TotalTurns() * t
	if h.Clockwise {
		return -a
	}
	return a
}

// PointAt returns the position at parameter t ∈ [0, 1].
func (h VariableHelix3d) PointAt(t float64) math.Point3 {
	turn := h.TotalTurns() * t
	p := pointOnCircle(h.Origin, h.RefDir.AsVector(), h.Axis.Cross(h.RefDir), h.radiusAt(turn), h.angleAt(t))
	return p.TranslateBy(h.Axis.AsVector().Scale(math.Scalar(h.heightAt(turn))))
}

// TangentAt returns dP/dt: radial growth + winding + axial advance, with the
// rates read from the station table at t.
func (h VariableHelix3d) TangentAt(t float64) math.Vector3 {
	total := h.TotalTurns()
	turn := total * t
	i := h.segmentAt(turn)
	a, b := h.stations[i], h.stations[i+1]
	dRadius := (b.Radius - a.Radius) / (b.Turn - a.Turn) * total
	dHeight := h.pitchAt(turn) * total
	dAngle := twoPi * total
	if h.Clockwise {
		dAngle = -dAngle
	}
	ang := h.angleAt(t)
	cos, sin := cosSin(ang)
	ref, bin := h.RefDir.AsVector(), h.Axis.Cross(h.RefDir)
	radialUnit := ref.Scale(math.Scalar(cos)).Add(bin.Scale(math.Scalar(sin)))
	radialTangent := ref.Scale(math.Scalar(-sin)).Add(bin.Scale(math.Scalar(cos)))
	out := radialUnit.Scale(math.Scalar(dRadius))
	out = out.Add(radialTangent.Scale(math.Scalar(h.radiusAt(turn) * dAngle)))
	return out.Add(h.Axis.AsVector().Scale(math.Scalar(dHeight)))
}

// Domain returns [0, 1].
func (h VariableHelix3d) Domain() (lo, hi float64) { return 0, 1 }

// Length integrates |dP/dt| by composite Simpson (same budget as Helix3d).
func (h VariableHelix3d) Length() float64 {
	return simpsonLength(func(t float64) float64 {
		return float64(h.TangentAt(t).Length())
	}, 0, 1, helixLengthIntervals)
}

var _ Curve3 = VariableHelix3d{}

// HeightAtTurn exposes the closed-form axial advance at a turn coordinate —
// the seam the coil feature rides to place profile sections along a
// variable-pitch rail (M06-F09, #624).
func (h VariableHelix3d) HeightAtTurn(turn float64) float64 { return h.heightAt(turn) }
