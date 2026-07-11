// SPDX-License-Identifier: GPL-2.0-only

// Package occtparity is the test-scope harness that replays OpenCASCADE's tests/blend corpus
// against our real fillet feature. It consumes per-case records emitted by the OCCT oracle
// (test-utilities/occt-blend/oracle), imports OCCT's exact input solid, geometrically locates
// each picked edge, drives the fillet feature, and asserts surface area within OCCT's own
// tolerance. It is imported only by _test.go and never ships.
package occtparity

import (
	"encoding/json"
	"fmt"
)

// Locator is a geometry-only edge locator, as OCCT resolved the picked edge. Midpoint and
// Direction are the mid-parameter point + tangent (reference only). Centroid and Length are
// the MATCHING key: STEP import reparameterizes edges to [0,1], so a curved edge's
// mid-parameter point does not correspond between kernels, but the arc-length centroid and
// total length are parameterization-invariant — both kernels compute the same values for the
// same physical edge, so we re-find it by centroid without depending on edge ordering.
type Locator struct {
	Midpoint  [3]float64 `json:"midpoint"`
	Direction [3]float64 `json:"direction"`
	Centroid  [3]float64 `json:"centroid"`
	Length    float64    `json:"length"`
}

// Pick is one blended edge. Radius is the constant fillet radius; Law is the variable-radius
// (parameter, radius) profile from `updatevol` and is nil for constant-radius picks.
type Pick struct {
	Radius  float64      `json:"radius"`
	Locator Locator      `json:"locator"`
	Law     [][2]float64 `json:"law"`
}

// Record is one OCCT tests/blend case: which blend verb ran, OCCT's reference area and
// tolerance, any TODO marker (mirrored, never exceeded), the input-solid STEP filename, and
// the picks. Shape is frozen in test-utilities/occt-blend/oracle/schema.json.
type Record struct {
	Grid         string  `json:"grid"`
	Case         string  `json:"case"`
	Verb         string  `json:"verb"`
	ExpectedArea float64 `json:"expectedArea"`
	Deps         float64 `json:"deps"`
	TODO         string  `json:"todo"`
	InputStep    string  `json:"inputStep"`
	Picks        []Pick  `json:"picks"`
}

// parseRecord decodes one oracle JSON record.
//
// Example:
//
//	r, err := parseRecord([]byte(`{"grid":"simple","case":"A1","verb":"blend",...}`))
func parseRecord(b []byte) (Record, error) {
	var r Record
	if err := json.Unmarshal(b, &r); err != nil {
		return Record{}, fmt.Errorf("parseRecord: invalid oracle JSON %q: %w", snippet(b), err)
	}
	return r, nil
}

// snippet bounds an error message to the first 200 bytes of the offending input.
func snippet(b []byte) string {
	const max = 200
	if len(b) > max {
		return string(b[:max]) + "…"
	}
	return string(b)
}
