// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/math"
)

// Interference analysis (M12-F05, Oblikovati/Oblikovati#362/#368) reports the overlapping
// volumes between occurrences — where two placed components occupy the same space. These are
// the result data types (satisfying contract.InterferenceResult(s)); the geometric computation
// (transform bodies to world, boolean-intersect, measure volume) lives in the host (compdef),
// which owns the body geometry.

// InterferenceResult is one overlapping pair: the two occurrence ids, the overlap volume, and a
// representative point inside the overlap.
type InterferenceResult struct {
	A, B   uint64
	Vol    float64
	Center math.Point3
}

// OccurrenceA / OccurrenceB are the interfering occurrences' session ids.
func (r InterferenceResult) OccurrenceA() uint64 { return r.A }
func (r InterferenceResult) OccurrenceB() uint64 { return r.B }

// Volume is the overlap volume (cubic centimetres).
func (r InterferenceResult) Volume() float64 { return r.Vol }

// InterferenceResults is an interference analysis outcome: the interfering pairs and the total
// overlap volume.
type InterferenceResults struct {
	Results []InterferenceResult
	Total   float64
}

// Count returns the number of interfering pairs.
func (rs InterferenceResults) Count() int { return len(rs.Results) }

// Item returns the i-th interference result, or nil when out of range.
func (rs InterferenceResults) Item(i int) contract.InterferenceResult {
	if i < 0 || i >= len(rs.Results) {
		return nil
	}
	return rs.Results[i]
}

// TotalVolume is the sum of every pair's overlap volume.
func (rs InterferenceResults) TotalVolume() float64 { return rs.Total }
