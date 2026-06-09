// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	m "oblikovati.org/math"
)

// A corner tetrahedron with legs s has volume s³/6 and centroid (s/4,s/4,s/4) — an
// analytic check that the divergence-theorem volume/centroid are correct (planar faces
// make tessellation exact).
func TestTetraVolumeAndCentroid(t *testing.T) {
	const s = 2.0
	gp := BodyGeometryProperties(tetra(s, m.V3(0, 0, 0)), DefaultQuality())

	wantVol := s * s * s / 6
	if math.Abs(gp.Volume-wantVol) > 1e-6 {
		t.Errorf("Volume = %v, want %v", gp.Volume, wantVol)
	}
	want := s / 4
	if math.Abs(float64(gp.Centroid.X)-want) > 1e-6 ||
		math.Abs(float64(gp.Centroid.Y)-want) > 1e-6 ||
		math.Abs(float64(gp.Centroid.Z)-want) > 1e-6 {
		t.Errorf("Centroid = %v, want (%v,%v,%v)", gp.Centroid, want, want, want)
	}
}

// Volume must be translation-invariant even though the tetra method references the
// origin — the off-origin parts cancel.
func TestVolumeIsTranslationInvariant(t *testing.T) {
	atOrigin := BodyGeometryProperties(tetra(2, m.V3(0, 0, 0)), DefaultQuality())
	moved := BodyGeometryProperties(tetra(2, m.V3(100, -50, 25)), DefaultQuality())
	if math.Abs(atOrigin.Volume-moved.Volume) > 1e-6 {
		t.Errorf("volume changed under translation: %v vs %v", atOrigin.Volume, moved.Volume)
	}
}
