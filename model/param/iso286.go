// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"
	stdmath "math"
	"strconv"
	"strings"
)

// millimetresPerDatabaseLength converts a database length value (centimetres —
// the Length category's database unit) to the millimetres the ISO 286 tables and
// [ISOFitBand] work in.
const millimetresPerDatabaseLength = 10

// ISOFitBand resolves an ISO 286 limits-and-fits class (e.g. "H7", "g6", "js7")
// at a nominal size (millimetres) into its deviation band relative to nominal,
// returned in millimetres with upper ≥ lower (#1848). An uppercase leading
// letter is a hole class, lowercase a shaft class. It errors for an unsupported
// class, grade, or nominal size rather than returning a dubious band — geometry
// and dimensioning must never be silently wrong.
//
// Example: ISOFitBand(50, "H7") → (0.025, 0) mm; ISOFitBand(50, "g6") →
// (-0.009, -0.025) mm.
func ISOFitBand(nominalMM float64, class string) (upper, lower float64, err error) {
	upMicron, loMicron, err := fitBandMicron(nominalMM, class)
	if err != nil {
		return 0, 0, err
	}
	return float64(upMicron) / 1000, float64(loMicron) / 1000, nil
}

// fitBandMicron computes the deviation band in whole micrometres (the units the
// ISO tables are published in).
func fitBandMicron(nominalMM float64, class string) (upper, lower int, err error) {
	letter, grade, isHole, err := parseFitClass(class)
	if err != nil {
		return 0, 0, err
	}
	step, err := iso286StepIndex(nominalMM)
	if err != nil {
		return 0, 0, err
	}
	itWidth, err := iso286ITWidth(grade, step)
	if err != nil {
		return 0, 0, err
	}
	return composeFitBand(letter, isHole, itWidth, iso286GeometricMean(step))
}

// composeFitBand places the IT width around the fundamental deviation for the
// letter: a symmetric ±IT/2 for js/JS, otherwise the shaft es (≤0) with ei =
// es − IT for a shaft, or its mirror EI = −es with ES = EI + IT for a hole.
func composeFitBand(letter string, isHole bool, itWidth int, d float64) (upper, lower int, err error) {
	if strings.EqualFold(letter, "js") {
		half := roundMicron(float64(itWidth) / 2)
		return half, -half, nil
	}
	es, ok := shaftFundamentalMicron(strings.ToLower(letter), d)
	if !ok {
		return 0, 0, fmt.Errorf("param: unsupported ISO fit letter %q (want d, f, g, h, or js, upper-case for a hole)", letter)
	}
	if isHole {
		ei := -es // hole EI = −es of the same-letter shaft (H → 0)
		return ei + itWidth, ei, nil
	}
	return es, es - itWidth, nil
}

// shaftFundamentalMicron returns the shaft upper deviation es in µm (rounded, ≤0)
// for the clearance/location letters at geometric mean d (mm); h is exactly 0.
// The formulas are ISO 286-1's fundamental-deviation expressions, restricted to
// the letters whose rounded output was validated against the published table
// (d, f, g, h) — e is deliberately omitted (its rounding could diverge by 1 µm
// from the standard and there was no in-repo table to confirm it).
func shaftFundamentalMicron(letter string, d float64) (int, bool) {
	switch letter {
	case "h":
		return 0, true
	case "g":
		return roundMicron(-2.5 * stdmath.Pow(d, 0.34)), true
	case "f":
		return roundMicron(-5.5 * stdmath.Pow(d, 0.41)), true
	case "d":
		return roundMicron(-16 * stdmath.Pow(d, 0.44)), true
	}
	return 0, false
}

// parseFitClass splits a class string into its letter run, IT grade, and hole
// flag (leading letter upper-case ⇒ hole).
func parseFitClass(class string) (letter string, grade int, isHole bool, err error) {
	i := 0
	for i < len(class) && (class[i] < '0' || class[i] > '9') {
		i++
	}
	letter, digits := class[:i], class[i:]
	if letter == "" || digits == "" {
		return "", 0, false, fmt.Errorf("param: malformed ISO fit class %q (want a letter and an IT grade, e.g. \"H7\")", class)
	}
	grade, convErr := strconv.Atoi(digits)
	if convErr != nil {
		return "", 0, false, fmt.Errorf("param: ISO fit class %q has a non-numeric grade %q", class, digits)
	}
	return letter, grade, class[0] >= 'A' && class[0] <= 'Z', nil
}

// iso286StepIndex returns the nominal-size step containing nominalMM, rejecting
// sizes outside the supported (3, 500] mm range.
func iso286StepIndex(nominalMM float64) (int, error) {
	if nominalMM <= 3 || nominalMM > 500 {
		return 0, fmt.Errorf("param: ISO fit nominal size %g mm out of range (supported: over 3 up to 500 mm)", nominalMM)
	}
	for i, hi := range iso286StepUpper {
		if nominalMM <= hi {
			return i, nil
		}
	}
	return 0, fmt.Errorf("param: ISO fit nominal size %g mm out of range", nominalMM)
}

// iso286ITWidth returns the IT grade width (µm) for a step, bounding the grade.
func iso286ITWidth(grade, step int) (int, error) {
	if grade < iso286FirstGrade || grade > iso286LastGrade {
		return 0, fmt.Errorf("param: unsupported ISO IT grade %d (supported: IT%d–IT%d)", grade, iso286FirstGrade, iso286LastGrade)
	}
	return iso286IT[step][grade-iso286FirstGrade], nil
}

// iso286GeometricMean is the geometric mean D = √(lo·hi) of a size step, the
// characteristic diameter the fundamental-deviation formulas take.
func iso286GeometricMean(step int) float64 {
	return stdmath.Sqrt(iso286StepLower[step] * iso286StepUpper[step])
}

// roundMicron rounds a deviation to the nearest whole micrometre (half away from
// zero, matching the ISO tables).
func roundMicron(v float64) int { return int(stdmath.Round(v)) }
