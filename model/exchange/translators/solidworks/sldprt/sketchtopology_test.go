// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"encoding/binary"
	"math"
	"testing"
)

// TestReconstructLinesNoEntities is a regression for a real corpus part ("cilindro scarico") whose
// line sketch decoded with a cached point but no entity-reference records and no construction-flag
// table. assembleLines then reached the all-referenced branch with an empty reference list and
// indexed refs[0]. reconstructLines must decline (ok=false) so the caller falls back, not panic.
func TestReconstructLinesNoEntities(t *testing.T) {
	region := []byte{pointMarker[0], pointMarker[1]}
	var coord [16]byte
	binary.LittleEndian.PutUint64(coord[0:], math.Float64bits(0.02)) // a valid in-range point
	binary.LittleEndian.PutUint64(coord[8:], math.Float64bits(0.03))
	region = append(region, coord[:]...) // one cached point, then nothing (no refs, no flag table)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("reconstructLines panicked on a no-entity region: %v", r)
		}
	}()
	if _, _, _, ok := reconstructLines(region); ok {
		t.Error("reconstructLines accepted a region with no entity references")
	}
}
