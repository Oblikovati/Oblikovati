// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"
	stdmath "math"
	"strconv"
	"strings"

	"oblikovati.org/api/types"
)

// Display formatting (Oblikovati/Oblikovati#146): the GetStringFromValue
// equivalent that honors the document's display PRECISION and FORMAT — decimal,
// fractional and architectural lengths, and decimal-degree or DMS angles. It is
// distinct from [UnitsOfMeasure.Format], which is the lossless, full-precision
// rendering (GetPreciseStringFromValue) that parse round-trips depend on.

// FormatDisplay renders q in its category's preferred unit at the document's
// display precision and format, with the unit appended (e.g. a 0.3175 cm length
// in fractional inches → "1/8 in"). Lengths and angles get the rich treatment;
// other categories fall back to the precise form.
//
// Example: with mm units at precision 2, FormatDisplay(Q(1.005, Length)) → "10.05 mm".
func (m UnitsOfMeasure) FormatDisplay(q Quantity) string {
	value := m.FormatValueDisplay(q)
	// Architectural lengths and DMS angles already carry their own ' " ° symbols.
	if q.Unit == Length && m.lengthFormat == types.DisplayFormatArchitectural {
		return value
	}
	if q.Unit == Angle && m.angleFormat == AngleDMS {
		return value
	}
	if name := m.PreferredName(q.Unit); name != "" {
		return value + " " + name
	}
	return value
}

// FormatValueDisplay renders just the numeric part of q at the display precision
// and format (no trailing unit name; architectural/DMS forms keep their inline
// ' " ° symbols since those ARE the value). It is what a UI field shows while
// the unit label is drawn separately.
func (m UnitsOfMeasure) FormatValueDisplay(q Quantity) string {
	switch q.Unit {
	case Length:
		return m.formatLengthDisplay(q)
	case Angle:
		return m.formatAngleDisplay(q)
	default:
		return m.FormatValue(q)
	}
}

// formatLengthDisplay renders a length per the configured length format.
func (m UnitsOfMeasure) formatLengthDisplay(q Quantity) string {
	switch m.lengthFormat {
	case types.DisplayFormatFractional:
		return fractionString(m.ToPreferred(q), m.fractionDenominator())
	case types.DisplayFormatArchitectural:
		return architecturalString(m.inchesOf(q), m.fractionDenominator())
	default:
		return formatFixed(m.ToPreferred(q), m.lengthPrecision)
	}
}

// formatAngleDisplay renders an angle as decimal degrees or DMS.
func (m UnitsOfMeasure) formatAngleDisplay(q Quantity) string {
	if m.angleFormat == AngleDMS {
		return dmsString(m.degreesOf(q))
	}
	return formatFixed(m.ToPreferred(q), m.anglePrecision)
}

// DisplayRoundedExpr renders q rounded to the granularity the document's display precision/format
// actually shows, as a plain, always-parseable decimal expression in the displayed unit
// ("10 mm", "30 deg", "0.125 in"). A dimension seeded with this stores exactly the clean on-screen
// number instead of the raw measured float ("9.999999998 mm") (Oblikovati/Oblikovati#146
// follow-up). The value is rounded in the displayed unit and emitted as decimal even when the
// FORMAT is fractional/architectural/DMS — so it round-trips through the expression parser
// (which has no fraction/°/feet literals) without the db→preferred float noise that re-dividing a
// db value would reintroduce for an irrational unit factor (e.g. degrees↔radians). Categories
// without a rich display fall back to the lossless form.
//
// Example: with mm units at precision 3, DisplayRoundedExpr(Q(0.9999999998, Length)) → "10 mm".
func (m UnitsOfMeasure) DisplayRoundedExpr(q Quantity) string {
	switch q.Unit {
	case Length:
		switch m.lengthFormat {
		case types.DisplayFormatFractional:
			return decimalExpr(roundToFraction(m.ToPreferred(q), m.fractionDenominator()), m.PreferredName(Length))
		case types.DisplayFormatArchitectural:
			return decimalExpr(roundToFraction(m.inchesOf(q), m.fractionDenominator()), "in")
		default:
			return decimalExpr(roundToDecimals(m.ToPreferred(q), m.lengthPrecision), m.PreferredName(Length))
		}
	case Angle:
		if m.angleFormat == AngleDMS {
			return decimalExpr(roundToDecimals(m.degreesOf(q)*3600, 0)/3600, "deg") // nearest arcsecond
		}
		return decimalExpr(roundToDecimals(m.ToPreferred(q), m.anglePrecision), m.PreferredName(Angle))
	default:
		return m.Format(q)
	}
}

// decimalExpr joins a shortest-decimal value with a unit name into a parseable expression.
func decimalExpr(v float64, unit string) string {
	s := strconv.FormatFloat(v, 'g', -1, 64)
	if unit == "" {
		return s
	}
	return s + " " + unit
}

// roundToDecimals rounds v to places decimal places (places < 0 treated as 0).
func roundToDecimals(v float64, places int) float64 {
	if places < 0 {
		places = 0
	}
	scale := stdmath.Pow(10, float64(places))
	return stdmath.Round(v*scale) / scale
}

// roundToFraction rounds v to the nearest 1/denom.
func roundToFraction(v float64, denom int) float64 {
	return stdmath.Round(v*float64(denom)) / float64(denom)
}

// fractionDenominator maps the length precision to a power-of-two fraction
// denominator: precision 1→halves, 3→eighths, 6→sixty-fourths, clamped to the
// 1/2…1/128 range Inventor offers.
func (m UnitsOfMeasure) fractionDenominator() int {
	p := m.lengthPrecision
	if p < 1 {
		p = 1
	}
	if p > 7 {
		p = 7
	}
	return 1 << p
}

// inchesOf returns q (a database-unit length) expressed in inches.
func (m UnitsOfMeasure) inchesOf(q Quantity) float64 {
	return q.Value / namedUnits["in"].factor
}

// degreesOf returns q (a database-unit angle) expressed in degrees.
func (m UnitsOfMeasure) degreesOf(q Quantity) float64 {
	return q.Value / namedUnits["deg"].factor
}

// formatFixed renders v with exactly places decimals (display precision).
func formatFixed(v float64, places int) string {
	return strconv.FormatFloat(v, 'f', places, 64)
}

// fractionString renders v as a whole number plus a reduced power-of-two
// fraction with the given denominator (e.g. 1.25, /8 → "1-1/4").
func fractionString(v float64, denom int) string {
	neg := v < 0
	if neg {
		v = -v
	}
	units := int(stdmath.Round(v * float64(denom)))
	s := fractionPart(units/denom, units%denom, denom)
	if neg {
		return "-" + s
	}
	return s
}

// fractionPart renders whole + rem/denom, reducing the fraction; rem==0 yields
// the bare whole number.
func fractionPart(whole, rem, denom int) string {
	if rem == 0 {
		return strconv.Itoa(whole)
	}
	g := gcdInt(rem, denom)
	rem, denom = rem/g, denom/g
	if whole == 0 {
		return fmt.Sprintf("%d/%d", rem, denom)
	}
	return fmt.Sprintf("%d-%d/%d", whole, rem, denom)
}

// architecturalString renders a length given in inches as feet and fractional
// inches (e.g. 13.5 in, /2 → "1' 1-1/2\""). The 1/denom rounding is done on the
// total so a rounded-up inch carries into feet.
func architecturalString(inches float64, denom int) string {
	neg := inches < 0
	if neg {
		inches = -inches
	}
	units := int(stdmath.Round(inches * float64(denom))) // total in 1/denom inch
	feetUnits := 12 * denom
	feet := units / feetUnits
	rem := units - feet*feetUnits
	inchStr := fractionPart(rem/denom, rem%denom, denom) + `"`

	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	if feet > 0 {
		fmt.Fprintf(&b, "%d' ", feet)
	}
	b.WriteString(inchStr)
	return b.String()
}

// dmsString renders an angle in degrees as degrees-minutes-seconds, carrying
// rounded 60s up (e.g. 30.5 → `30° 30' 0"`).
func dmsString(deg float64) string {
	neg := deg < 0
	if neg {
		deg = -deg
	}
	d := int(deg)
	minutes := (deg - float64(d)) * 60
	mi := int(minutes)
	si := int(stdmath.Round((minutes - float64(mi)) * 60))
	if si == 60 {
		si, mi = 0, mi+1
	}
	if mi == 60 {
		mi, d = 0, d+1
	}
	s := fmt.Sprintf("%d° %d' %d\"", d, mi, si)
	if neg {
		return "-" + s
	}
	return s
}

// gcdInt is the greatest common divisor (Euclid), for reducing fractions.
func gcdInt(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a == 0 {
		return 1
	}
	return a
}
