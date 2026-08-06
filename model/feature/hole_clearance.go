// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	"sort"
	"strings"
)

// Clearance holes (Oblikovati#1862). A clearance hole is not sized by the author: it is sized by
// the FASTENER that passes through it and the fit wanted, out of a published table. Recording the
// resolved diameter instead of the fastener loses that — swap an M6 screw for an M8 and the hole
// stays Ø6.6 — so the definition keeps the fastener and looks the diameter up every recompute.
//
// The table is ISO 273 "Fasteners — Clearance holes for bolts and screws", the standard Inventor's
// own metric clearance data follows. Values are the published nominal hole diameters in MILLIMETRES
// for the three fit series; the lookup converts to the kernel's centimetres.

// HoleClearanceInfo names the fastener a clearance hole is drilled for — Inventor's
// HoleClearanceInfo. An empty Fastener means the hole is not a clearance hole and Diameter rules.
type HoleClearanceInfo struct {
	// Standard is the table the fastener is drawn from; only "ISO 273" is carried today, and an
	// unknown one is refused rather than silently resolved against the wrong series.
	Standard string
	// Fastener is the thread designation, e.g. "M6". A full designation ("M6x1") is accepted — the
	// pitch does not change a clearance hole.
	Fastener string
	// Fit is "close", "medium" (the default) or "free".
	Fit string
}

// isSet reports whether the hole is a clearance hole at all.
func (c HoleClearanceInfo) isSet() bool { return strings.TrimSpace(c.Fastener) != "" }

// clearanceFits maps a fit name onto its column in the table.
var clearanceFits = map[string]int{"": 1, "close": 0, "medium": 1, "free": 2}

// iso273Holes is ISO 273's clearance-hole diameters in mm, keyed by the fastener's nominal
// diameter in mm, as {close, medium, free}.
var iso273Holes = map[float64][3]float64{
	1.6: {1.7, 1.8, 2.0}, 2: {2.2, 2.4, 2.6}, 2.5: {2.7, 2.9, 3.1}, 3: {3.2, 3.4, 3.6},
	4: {4.3, 4.5, 4.8}, 5: {5.3, 5.5, 5.8}, 6: {6.4, 6.6, 7.0}, 8: {8.4, 9.0, 10.0},
	10: {10.5, 11.0, 12.0}, 12: {13.0, 13.5, 14.5}, 14: {15.0, 15.5, 16.5}, 16: {17.0, 17.5, 18.5},
	18: {19.0, 20.0, 21.0}, 20: {21.0, 22.0, 24.0}, 22: {23.0, 24.0, 26.0}, 24: {25.0, 26.0, 28.0},
	27: {28.0, 30.0, 32.0}, 30: {31.0, 33.0, 35.0}, 33: {34.0, 36.0, 38.0}, 36: {37.0, 39.0, 42.0},
	39: {40.0, 42.0, 45.0}, 42: {43.0, 45.0, 48.0}, 45: {46.0, 48.0, 52.0}, 48: {50.0, 52.0, 56.0},
	52: {54.0, 56.0, 62.0}, 56: {58.0, 62.0, 66.0}, 60: {62.0, 66.0, 70.0}, 64: {66.0, 70.0, 74.0},
}

// ClearanceDiameter resolves the bore diameter, in model centimetres, for the named fastener and
// fit. Every failure names what was not recognised and what the table holds, because a clearance
// hole silently falling back to some other diameter is a part that will not assemble.
//
// Example:
//
//	d, err := feature.HoleClearanceInfo{Fastener: "M6", Fit: "close"}.ClearanceDiameter() // 0.64 cm
func (c HoleClearanceInfo) ClearanceDiameter() (float64, error) {
	if std := strings.TrimSpace(c.Standard); std != "" && !strings.EqualFold(std, "ISO 273") {
		return 0, fmt.Errorf("hole: clearance standard %q is not carried (only %q)", std, "ISO 273")
	}
	column, ok := clearanceFits[strings.ToLower(strings.TrimSpace(c.Fit))]
	if !ok {
		return 0, fmt.Errorf("hole: clearance fit %q is not one of close, medium or free", c.Fit)
	}
	spec, err := ParseThreadDesignation(clearanceDesignation(c.Fastener))
	if err != nil {
		return 0, fmt.Errorf("hole: clearance fastener %q: %w", c.Fastener, err)
	}
	row, ok := iso273Holes[nominalMillimetres(spec.MajorDiameter)]
	if !ok {
		return 0, fmt.Errorf("hole: ISO 273 has no clearance row for fastener %q (Ø%g mm); carried sizes are %s",
			c.Fastener, nominalMillimetres(spec.MajorDiameter), clearanceSizeList())
	}
	return row[column] / 10, nil // the table is in mm; the kernel works in cm
}

// clearanceDesignation completes a bare size ("M6") with a pitch, so a size outside the ISO coarse-
// pitch table (M22, M33, …) still parses. Only the major diameter is read back — a clearance hole is
// named by size alone, and its pitch never changes the hole.
func clearanceDesignation(fastener string) string {
	f := strings.TrimSpace(fastener)
	if strings.Contains(f, "x") || strings.Contains(f, "X") {
		return f
	}
	return f + "x1" // any pitch parses; only the major diameter is read back
}

// nominalMillimetres rounds a parsed major diameter (already mm) to the table's key precision, so
// floating-point noise from the designation parser cannot miss an exact row.
func nominalMillimetres(mm float64) float64 {
	return float64(int(mm*10+0.5)) / 10
}

// clearanceSizeList renders the carried fastener sizes for an error message.
func clearanceSizeList() string {
	sizes := make([]float64, 0, len(iso273Holes))
	for d := range iso273Holes {
		sizes = append(sizes, d)
	}
	sort.Float64s(sizes)
	out := make([]string, len(sizes))
	for i, d := range sizes {
		out[i] = fmt.Sprintf("M%g", d)
	}
	return strings.Join(out, ", ")
}
