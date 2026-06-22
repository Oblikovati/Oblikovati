// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestFormatDisplayDecimal honors the length/angle display precision with fixed
// decimals, distinct from the lossless Format.
func TestFormatDisplayDecimal(t *testing.T) {
	m := DefaultUnitsOfMeasure() // mm, 3 length decimals; deg, 2 angle decimals
	// Display pads to the fixed precision; the lossless Format stays shortest.
	if got := m.FormatDisplay(Q(1, Length)); got != "10.000 mm" {
		t.Errorf("length display = %q, want \"10.000 mm\"", got)
	}
	if got := m.Format(Q(1, Length)); got != "10 mm" {
		t.Errorf("length precise = %q, want \"10 mm\"", got)
	}
	// Display rounds to the precision (12.3456 mm → 12.346 mm).
	if got := m.FormatDisplay(Q(1.23456, Length)); got != "12.346 mm" {
		t.Errorf("length display = %q, want \"12.346 mm\"", got)
	}
	// 45° rendered with two angle decimals.
	if got := m.FormatDisplay(Q(45*namedUnits["deg"].factor, Angle)); got != "45.00 deg" {
		t.Errorf("angle display = %q, want \"45.00 deg\"", got)
	}
}

// TestFormatDisplayFractional renders the preferred (inch) unit as reduced
// power-of-two fractions.
func TestFormatDisplayFractional(t *testing.T) {
	m := DefaultUnitsOfMeasure().Clone()
	if err := m.SetPreferred(Length, "in"); err != nil {
		t.Fatal(err)
	}
	m.SetLengthFormat(types.DisplayFormatFractional)
	if err := m.SetLengthPrecision(3); err != nil { // eighths
		t.Fatal(err)
	}
	cases := map[float64]string{ // value in cm → display
		2.54:    "1 in",       // exactly 1"
		3.175:   "1-1/4 in",   // 1.25"
		0.3175:  "1/8 in",     // 0.125"
		0:       "0 in",       // zero
		-3.9688: "-1-9/16 in", // negative, needs 1/16 (precision bumped below)
	}
	if err := m.SetLengthPrecision(4); err != nil { // sixteenths for the -1-9/16 case
		t.Fatal(err)
	}
	for cm, want := range cases {
		if got := m.FormatDisplay(Q(cm, Length)); got != want {
			t.Errorf("FormatDisplay(%g cm) = %q, want %q", cm, got, want)
		}
	}
}

// TestFormatDisplayArchitectural renders feet and fractional inches.
func TestFormatDisplayArchitectural(t *testing.T) {
	m := DefaultUnitsOfMeasure().Clone()
	if err := m.SetPreferred(Length, "in"); err != nil {
		t.Fatal(err)
	}
	m.SetLengthFormat(types.DisplayFormatArchitectural)
	if err := m.SetLengthPrecision(2); err != nil { // quarters
		t.Fatal(err)
	}
	// 13.5 in = 34.29 cm → 1' 1-1/2"
	if got := m.FormatDisplay(Q(34.29, Length)); got != `1' 1-1/2"` {
		t.Errorf("architectural = %q, want \"1' 1-1/2\\\"\"", got)
	}
	// 12 in = 30.48 cm → 1' 0"
	if got := m.FormatDisplay(Q(30.48, Length)); got != `1' 0"` {
		t.Errorf("architectural = %q, want \"1' 0\\\"\"", got)
	}
	// 5.5 in = 13.97 cm → 5-1/2" (no feet)
	if got := m.FormatDisplay(Q(13.97, Length)); got != `5-1/2"` {
		t.Errorf("architectural = %q, want \"5-1/2\\\"\"", got)
	}
}

// TestFormatDisplayDMS renders angles as degrees-minutes-seconds, carrying 60s.
func TestFormatDisplayDMS(t *testing.T) {
	m := DefaultUnitsOfMeasure()
	m.SetAngleFormat(AngleDMS)
	degToRad := namedUnits["deg"].factor
	cases := map[float64]string{ // degrees → display
		30.5:    `30° 30' 0"`,
		0:       `0° 0' 0"`,
		90:      `90° 0' 0"`,
		10.2589: `10° 15' 32"`,
	}
	for deg, want := range cases {
		if got := m.FormatDisplay(Q(deg*degToRad, Angle)); got != want {
			t.Errorf("DMS(%g deg) = %q, want %q", deg, got, want)
		}
	}
}

// TestFractionHelpers pins the fraction/gcd primitives.
func TestFractionHelpers(t *testing.T) {
	if g := gcdInt(12, 8); g != 4 {
		t.Errorf("gcd(12,8) = %d, want 4", g)
	}
	if s := fractionString(1.25, 8); s != "1-1/4" {
		t.Errorf("fractionString(1.25,8) = %q, want 1-1/4", s)
	}
	// Rounds up to the next whole when the fraction completes the denominator.
	if s := fractionString(0.999, 2); s != "1" {
		t.Errorf("fractionString(0.999,2) = %q, want 1", s)
	}
}

// TestDisplayRoundedExpr verifies a raw measured float collapses to the clean value the display
// precision shows, emitted as a parseable decimal expression (no float noise — the whole point).
func TestDisplayRoundedExpr(t *testing.T) {
	m := DefaultUnitsOfMeasure() // mm, 3 length decimals; deg, 2 angle decimals
	deg := namedUnits["deg"].factor
	cases := []struct {
		q    Quantity
		want string
	}{
		{Q(0.9999999998, Length), "10 mm"},     // ~10 mm edge with float noise → clean
		{Q(1.234567, Length), "12.346 mm"},     // rounds to 3 mm-decimals
		{Q(29.999999998*deg, Angle), "30 deg"}, // angle noise → clean (no deg↔rad round-trip noise)
		{Q(30.126*deg, Angle), "30.13 deg"},    // rounds to 2 angle-decimals
	}
	for _, c := range cases {
		if got := m.DisplayRoundedExpr(c.q); got != c.want {
			t.Errorf("DisplayRoundedExpr(%v) = %q, want %q", c.q, got, c.want)
		}
	}
	// A category without a rich display falls back to the lossless form.
	if got := m.DisplayRoundedExpr(Q(2.5, Mass)); got != m.Format(Q(2.5, Mass)) {
		t.Errorf("mass expr = %q, want lossless %q", got, m.Format(Q(2.5, Mass)))
	}
}

// TestDisplayRoundedExprFractional: a fractional-format document seeds a decimal expression that
// still parses (the parser has no fraction literal), rounded to the fraction granularity.
func TestDisplayRoundedExprFractional(t *testing.T) {
	m := DefaultUnitsOfMeasure().Clone()
	if err := m.SetPreferred(Length, "in"); err != nil {
		t.Fatal(err)
	}
	m.SetLengthFormat(types.DisplayFormatFractional)
	if err := m.SetLengthPrecision(3); err != nil { // eighths
		t.Fatal(err)
	}
	// 0.124 in ≈ 1/8 in → rounds to exactly 0.125 in, emitted as decimal.
	if got := m.DisplayRoundedExpr(Q(0.124*2.54, Length)); got != "0.125 in" {
		t.Errorf("fractional expr = %q, want \"0.125 in\"", got)
	}
}

// TestFormatDisplayEdgeCases covers the fallback, clamp, and negative paths.
func TestFormatDisplayEdgeCases(t *testing.T) {
	m := DefaultUnitsOfMeasure()
	// A non-length/angle category falls back to the precise value form.
	if got, want := m.FormatValueDisplay(Q(2, Mass)), m.FormatValue(Q(2, Mass)); got != want {
		t.Errorf("mass display = %q, want precise %q", got, want)
	}
	if m.AngleFormat() != AngleDecimal {
		t.Error("default angle format should be decimal")
	}

	inFrac := func(prec int, format types.ParameterDisplayFormat) UnitsOfMeasure {
		u := m.Clone()
		if err := u.SetPreferred(Length, "in"); err != nil {
			t.Fatal(err)
		}
		if err := u.SetLengthPrecision(prec); err != nil {
			t.Fatal(err)
		}
		u.SetLengthFormat(format)
		return u
	}

	// Precision 0 clamps the fraction denominator up to halves.
	if got := inFrac(0, types.DisplayFormatFractional).FormatDisplay(Q(2.54*1.5, Length)); got != "1-1/2 in" {
		t.Errorf("clamped-low fraction = %q, want \"1-1/2 in\"", got)
	}
	// Precision 20 clamps down to 128ths; renders feet-inches.
	if got := inFrac(20, types.DisplayFormatArchitectural).FormatDisplay(Q(2.54*13.5, Length)); got != `1' 1-1/2"` {
		t.Errorf("clamped-high architectural = %q", got)
	}
	// Negative architectural and DMS keep the leading sign.
	if got := inFrac(2, types.DisplayFormatArchitectural).FormatDisplay(Q(-2.54*13.5, Length)); got != `-1' 1-1/2"` {
		t.Errorf("negative architectural = %q", got)
	}
	dms := m.Clone()
	dms.SetAngleFormat(AngleDMS)
	if got := dms.FormatDisplay(Q(-30.5*namedUnits["deg"].factor, Angle)); got != `-30° 30' 0"` {
		t.Errorf("negative DMS = %q", got)
	}
	if gcdInt(0, 0) != 1 {
		t.Error("gcd(0,0) should fall back to 1")
	}
}
