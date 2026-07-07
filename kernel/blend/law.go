// SPDX-License-Identifier: GPL-2.0-only

package blend

import "sort"

// RadiusLaw maps a spine abscissa to the fillet radius there — the evolution of a variable-radius
// fillet along its guide. It mirrors OCCT Law_Function (Law_Constant / Law_Linear / Law_Interpol):
// the marcher samples At(w) at each section station so one guide can carry a changing radius.
type RadiusLaw interface {
	// At returns the radius at spine abscissa w.
	At(w float64) float64
}

// ConstLaw is a constant radius (OCCT Law_Constant) — the ordinary uniform fillet.
type ConstLaw struct {
	R float64
}

// At returns the constant radius.
func (l ConstLaw) At(float64) float64 { return l.R }

// LinearLaw ramps the radius linearly from R0 at abscissa S0 to R1 at S1 (OCCT Law_Linear),
// clamped to [R0,R1] outside [S0,S1]. It is the two-ended variable fillet.
type LinearLaw struct {
	S0, R0, S1, R1 float64
}

// At returns the linearly interpolated radius at w, clamped at the ends.
func (l LinearLaw) At(w float64) float64 {
	if l.S1 == l.S0 {
		return l.R0
	}
	t := (w - l.S0) / (l.S1 - l.S0)
	if t <= 0 {
		return l.R0
	}
	if t >= 1 {
		return l.R1
	}
	return l.R0 + t*(l.R1-l.R0)
}

// LawStop is one (abscissa, radius) breakpoint of an interpolated radius law.
type LawStop struct {
	S, R float64
}

// InterpLaw is a piecewise-linear radius through ordered (abscissa,radius) stops (OCCT
// Law_Interpol) — a fillet with intermediate radius set-points. Stops are sorted by abscissa on
// construction; between them the radius interpolates linearly, and it is clamped outside the range.
type InterpLaw struct {
	stops []LawStop
}

// NewInterpLaw builds an interpolated law from stops (which need not be pre-sorted). It returns a
// ConstLaw-equivalent single value when given fewer than two stops.
func NewInterpLaw(stops []LawStop) InterpLaw {
	s := append([]LawStop(nil), stops...)
	sort.Slice(s, func(i, j int) bool { return s[i].S < s[j].S })
	return InterpLaw{stops: s}
}

// At returns the piecewise-linear radius at w, clamped to the first/last stop outside the range.
func (l InterpLaw) At(w float64) float64 {
	if len(l.stops) == 0 {
		return 0
	}
	if w <= l.stops[0].S {
		return l.stops[0].R
	}
	last := l.stops[len(l.stops)-1]
	if w >= last.S {
		return last.R
	}
	for i := 1; i < len(l.stops); i++ {
		if w <= l.stops[i].S {
			a, b := l.stops[i-1], l.stops[i]
			t := (w - a.S) / (b.S - a.S)
			return a.R + t*(b.R-a.R)
		}
	}
	return last.R
}
