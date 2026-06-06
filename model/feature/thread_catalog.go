// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// Thread standards and the size/pitch tables behind the thread tool's preferences. A standard
// (ISO/ANSI/JIS) groups sizes under a system (metric or imperial); each size offers one or more
// pitches. The picked (standard, size, pitch) resolves to a designation a ThreadFeature uses.

// ThreadStandard names a thread standard family.
type ThreadStandard string

const (
	StandardISO  ThreadStandard = "ISO"  // ISO metric (M) — also the JIS basis
	StandardANSI ThreadStandard = "ANSI" // Unified inch (UNC/UNF)
	StandardJIS  ThreadStandard = "JIS"  // Japanese — ISO metric profile
)

// ThreadSystem is metric or imperial.
type ThreadSystem string

const (
	SystemMetric   ThreadSystem = "metric"
	SystemImperial ThreadSystem = "imperial"
)

// ThreadSize is one nominal size under a standard: its display name, major diameter (mm), and
// the pitches it offers (mm — coarse first). Imperial sizes store the pitch converted to mm.
type ThreadSize struct {
	Name          string
	System        ThreadSystem
	MajorDiameter float64
	Pitches       []float64
}

// ThreadStandards lists the supported standards (UI order).
func ThreadStandards() []ThreadStandard {
	return []ThreadStandard{StandardISO, StandardANSI, StandardJIS}
}

// StandardSystem returns the measurement system a standard uses.
func StandardSystem(std ThreadStandard) ThreadSystem {
	if std == StandardANSI {
		return SystemImperial
	}
	return SystemMetric
}

// metricSizes is the ISO/JIS coarse+fine table (mm). The first pitch is the coarse pitch.
var metricSizes = []ThreadSize{
	{"M2", SystemMetric, 2, []float64{0.4, 0.25}},
	{"M2.5", SystemMetric, 2.5, []float64{0.45, 0.35}},
	{"M3", SystemMetric, 3, []float64{0.5, 0.35}},
	{"M4", SystemMetric, 4, []float64{0.7, 0.5}},
	{"M5", SystemMetric, 5, []float64{0.8, 0.5}},
	{"M6", SystemMetric, 6, []float64{1.0, 0.75}},
	{"M8", SystemMetric, 8, []float64{1.25, 1.0}},
	{"M10", SystemMetric, 10, []float64{1.5, 1.25, 1.0}},
	{"M12", SystemMetric, 12, []float64{1.75, 1.5, 1.25}},
	{"M16", SystemMetric, 16, []float64{2.0, 1.5}},
	{"M20", SystemMetric, 20, []float64{2.5, 1.5}},
	{"M24", SystemMetric, 24, []float64{3.0, 2.0}},
}

// ansiSizes is the Unified inch table (UNC coarse + UNF fine TPI → pitch mm). Major diameters
// are the nominal inch sizes in mm; the size Name omits the TPI (the pitch supplies it).
var ansiSizes = []ThreadSize{
	{"#4", SystemImperial, 0.112 * inchPerMM, []float64{inchPerMM / 40, inchPerMM / 48}},
	{"#6", SystemImperial, 0.138 * inchPerMM, []float64{inchPerMM / 32, inchPerMM / 40}},
	{"#8", SystemImperial, 0.164 * inchPerMM, []float64{inchPerMM / 32, inchPerMM / 36}},
	{"#10", SystemImperial, 0.190 * inchPerMM, []float64{inchPerMM / 24, inchPerMM / 32}},
	{"1/4", SystemImperial, 0.25 * inchPerMM, []float64{inchPerMM / 20, inchPerMM / 28}},
	{"5/16", SystemImperial, 0.3125 * inchPerMM, []float64{inchPerMM / 18, inchPerMM / 24}},
	{"3/8", SystemImperial, 0.375 * inchPerMM, []float64{inchPerMM / 16, inchPerMM / 24}},
	{"1/2", SystemImperial, 0.5 * inchPerMM, []float64{inchPerMM / 13, inchPerMM / 20}},
}

// ThreadSizes returns the sizes offered by a standard.
func ThreadSizes(std ThreadStandard) []ThreadSize {
	switch std {
	case StandardANSI:
		return ansiSizes
	default: // ISO and JIS share the metric table
		return metricSizes
	}
}

// findSize returns the named size under a standard.
func findSize(std ThreadStandard, name string) (ThreadSize, bool) {
	for _, s := range ThreadSizes(std) {
		if s.Name == name {
			return s, true
		}
	}
	return ThreadSize{}, false
}

// ThreadDesignation builds the parseable designation string for a catalog pick — "M8x1.25" for
// metric, "1/4-20" for imperial (TPI = 25.4/pitch) — that ParseThreadDesignation reads back. It
// errors for an unknown size or a pitch the size does not offer.
func ThreadDesignation(std ThreadStandard, sizeName string, pitch float64) (string, error) {
	size, ok := findSize(std, sizeName)
	if !ok {
		return "", fmt.Errorf("thread: %s has no size %q", std, sizeName)
	}
	if !hasPitch(size.Pitches, pitch) {
		return "", fmt.Errorf("thread: %s %s has no pitch %g (have %v)", std, sizeName, pitch, size.Pitches)
	}
	if size.System == SystemImperial {
		return fmt.Sprintf("%s-%d", sizeName, int(inchPerMM/pitch+0.5)), nil
	}
	return fmt.Sprintf("%sx%g", sizeName, pitch), nil
}

func hasPitch(ps []float64, p float64) bool {
	for _, x := range ps {
		if x == p {
			return true
		}
	}
	return false
}
