// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	"strconv"
	"strings"

	"oblikovati.org/api/types"
)

// ThreadSpec is the resolved geometry of a thread designation (the cosmetic-thread data the
// reference records on a face): pitch and the basic-profile diameters, handedness, the
// tolerance class, and whether it is an internal (hole) or external (shaft) thread. A
// cosmetic thread does not cut the solid; this is the data that drives the thread display,
// hole tables (M14), and downstream fit checks — one source of truth per the #325 audit.
type ThreadSpec struct {
	Designation      string
	ThreadType       string  // catalog standard ("ISO" for metric, "ANSI" for Unified inch)
	NominalSize      string  // "M8", "1/4", "#8"
	Class            string  // tolerance class ("6H", "2A", …); empty = unspecified
	Pitch            float64 // mm between crests
	MajorDiameter    float64 // mm (nominal)
	MinorDiameter    float64 // mm (root, ISO basic d − 1.0825·P)
	PitchDiameter    float64 // mm (ISO basic d − 0.6495·P)
	TapDrillDiameter float64 // mm (standard tap drill, d − P)
	Metric           bool
	Internal         bool
	RightHanded      bool
	Tapered          bool // pipe thread (the reference's TaperedThreadInfo split)
	// ModelDiameter declares which thread diameter the modeled cylindrical face
	// represents (major when unset) — consumed by drawings/hole tables, not geometry.
	ModelDiameter types.ModelDiameterFromThread
}

// fillDerivedDiameters computes the ISO-basic pitch diameter and the standard tap drill from
// the major diameter and pitch (shared by the metric and Unified parsers — the Unified basic
// profile uses the same 60° relations, in mm).
func (s *ThreadSpec) fillDerivedDiameters() {
	s.PitchDiameter = s.MajorDiameter - 0.6495*s.Pitch
	s.TapDrillDiameter = s.MajorDiameter - s.Pitch
}

// coarsePitch is the ISO metric coarse-pitch table for common nominal diameters (mm).
var coarsePitch = map[float64]float64{
	2: 0.4, 2.5: 0.45, 3: 0.5, 3.5: 0.6, 4: 0.7, 5: 0.8, 6: 1.0, 8: 1.25,
	10: 1.5, 12: 1.75, 14: 2.0, 16: 2.0, 18: 2.5, 20: 2.5, 24: 3.0,
}

// ParseThreadDesignation parses an ISO metric designation into a [ThreadSpec]: "M8x1.25"
// (explicit pitch) or "M8" (coarse pitch from the table). A trailing "-LH" marks a left-hand
// thread. The minor diameter follows the ISO basic profile, d = D − 1.0825·P.
//
//	spec, _ := ParseThreadDesignation("M8x1.25")  // metric: pitch 1.25, major 8, minor 6.647
//	spec, _ := ParseThreadDesignation("1/4-20")   // Unified inch: 20 TPI
func ParseThreadDesignation(s string) (ThreadSpec, error) {
	spec := ThreadSpec{Designation: s, RightHanded: true}
	body := strings.TrimSpace(s)
	if u := strings.ToUpper(body); strings.HasSuffix(u, "-LH") {
		spec.RightHanded = false
		body = body[:len(body)-3]
	}
	if len(body) == 0 {
		return ThreadSpec{}, fmt.Errorf("thread: empty designation")
	}
	if body[0] == '#' || (body[0] >= '0' && body[0] <= '9') {
		return parseImperial(body, spec)
	}
	if body[0] != 'M' && body[0] != 'm' {
		return ThreadSpec{}, fmt.Errorf("thread: unrecognised designation %q (want metric M8x1.25 or inch 1/4-20)", s)
	}
	return parseMetric(body[1:], spec)
}

// inchPerMM is mm per inch, for Unified inch threads.
const inchPerMM = 25.4

// parseImperial resolves a Unified inch designation "<size>-<TPI>" — size is a fraction
// ("1/4", "5/16") or a numbered gauge ("#8", "8") — into the spec. Major Ø = size·25.4 mm,
// pitch = 25.4/TPI mm.
func parseImperial(body string, spec ThreadSpec) (ThreadSpec, error) {
	i := strings.LastIndex(body, "-")
	if i < 0 {
		return ThreadSpec{}, fmt.Errorf("thread: inch designation %q needs a TPI ('1/4-20')", spec.Designation)
	}
	sizeStr, tpiStr := strings.TrimSpace(body[:i]), strings.TrimSpace(body[i+1:])
	tpi, err := strconv.ParseFloat(tpiStr, 64)
	if err != nil || tpi <= 0 {
		return ThreadSpec{}, fmt.Errorf("thread: bad TPI in %q: %v", spec.Designation, err)
	}
	majorIn, err := inchSize(sizeStr)
	if err != nil {
		return ThreadSpec{}, err
	}
	spec.MajorDiameter = majorIn * inchPerMM
	spec.Pitch = inchPerMM / tpi
	spec.MinorDiameter = spec.MajorDiameter - 1.0825*spec.Pitch
	if spec.MinorDiameter <= 0 {
		return ThreadSpec{}, fmt.Errorf("thread: %s too coarse (minor diameter ≤ 0)", spec.Designation)
	}
	spec.ThreadType, spec.NominalSize = string(StandardANSI), sizeStr
	spec.fillDerivedDiameters()
	return spec, nil
}

func parseMetric(body string, spec ThreadSpec) (ThreadSpec, error) {
	major, pitchStr, err := parseMetricMajorAndPitch(body)
	if err != nil {
		return ThreadSpec{}, err
	}
	spec.MajorDiameter = major
	spec.Pitch, err = metricPitch(major, pitchStr)
	if err != nil {
		return ThreadSpec{}, err
	}
	spec.MinorDiameter = major - 1.0825*spec.Pitch
	if spec.MinorDiameter <= 0 {
		return ThreadSpec{}, fmt.Errorf("thread: pitch %g too coarse for M%g (minor diameter ≤ 0)", spec.Pitch, major)
	}
	spec.Metric, spec.ThreadType, spec.NominalSize = true, string(StandardISO), fmt.Sprintf("M%g", major)
	spec.fillDerivedDiameters()
	return spec, nil
}

func parseMetricMajorAndPitch(body string) (float64, string, error) {
	pitchStr := ""
	if i := strings.IndexAny(body, "xX"); i >= 0 {
		body, pitchStr = body[:i], body[i+1:]
	}
	major, err := strconv.ParseFloat(strings.TrimSpace(body), 64)
	if err != nil || major <= 0 {
		return 0, "", fmt.Errorf("thread: bad nominal diameter in %q: %v", body, err)
	}
	return major, pitchStr, nil
}

func metricPitch(major float64, pitchStr string) (float64, error) {
	if pitchStr == "" {
		p, ok := coarsePitch[major]
		if !ok {
			return 0, fmt.Errorf("thread: no coarse pitch for M%g; give an explicit pitch (e.g. M%gx1.0)", major, major)
		}
		return p, nil
	}
	p, err := strconv.ParseFloat(strings.TrimSpace(pitchStr), 64)
	if err != nil || p <= 0 {
		return 0, fmt.Errorf("thread: bad pitch %q: %v", pitchStr, err)
	}
	return p, nil
}

// inchSize parses a fraction ("1/4", "5/16") or numbered gauge ("#8", "8") to inches. Gauge N
// has major diameter 0.060 + 0.013·N inch.
func inchSize(s string) (float64, error) {
	s = strings.TrimPrefix(s, "#")
	if j := strings.Index(s, "/"); j >= 0 {
		num, err1 := strconv.ParseFloat(s[:j], 64)
		den, err2 := strconv.ParseFloat(s[j+1:], 64)
		if err1 != nil || err2 != nil || den == 0 {
			return 0, fmt.Errorf("thread: bad inch fraction %q", s)
		}
		return num / den, nil
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("thread: bad inch size %q", s)
	}
	return 0.060 + 0.013*n, nil // numbered gauge
}
