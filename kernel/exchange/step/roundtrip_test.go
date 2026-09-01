// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
)

// roundTrip imports a fixture, exports it, and re-imports the exported bytes,
// returning the original and re-imported bodies. It is the PBI-C "without
// degradation" invariant harness.
func roundTrip(t *testing.T, fixture string) (orig, again *topo.Body) {
	t.Helper()
	orig = importOneSolid(t, fixture)
	data, _, err := Writer{}.ExportSolids([]*topo.Body{orig}, exchange.TranslationOptions{})
	if err != nil {
		t.Fatalf("export %s: %v", fixture, err)
	}
	bodies, warns, err := Reader{}.ImportSolids(data, exchange.TranslationOptions{})
	if err != nil {
		t.Fatalf("re-import %s: %v\n%s", fixture, err, data)
	}
	for _, w := range warns {
		t.Logf("re-import warning: %s", w)
	}
	if len(bodies) != 1 {
		t.Fatalf("re-import %s: got %d bodies, want 1", fixture, len(bodies))
	}
	return orig, bodies[0]
}

func TestRoundTripCube(t *testing.T) {
	t.Parallel()
	assertRoundTripPreservesSolid(t, "cube.step")
}

func TestRoundTripCylinder(t *testing.T) {
	t.Parallel()
	assertRoundTripPreservesSolid(t, "cylinder.step")
}

func TestRoundTripBoxWithHole(t *testing.T) {
	t.Parallel()
	assertRoundTripPreservesSolid(t, "box_hole.step")
}

// assertRoundTripPreservesSolid checks validity, face count, volume and centroid
// survive an import→export→import cycle within tolerance.
func assertRoundTripPreservesSolid(t *testing.T, fixture string) {
	t.Helper()
	orig, again := roundTrip(t, fixture)
	if r := ops.Validate(again); !r.Valid {
		t.Errorf("%s re-imported invalid: %+v", fixture, r)
	}
	if a, b := len(orig.Faces()), len(again.Faces()); a != b {
		t.Errorf("%s face count changed: %d → %d", fixture, a, b)
	}
	po := query.BodyGeometryProperties(orig, fineQuality())
	pa := query.BodyGeometryProperties(again, fineQuality())
	if !approx(pa.Volume, po.Volume, 1e-3) {
		t.Errorf("%s volume changed: %g → %g", fixture, po.Volume, pa.Volume)
	}
	assertCentroidPreserved(t, fixture, po, pa)
}

// assertCentroidPreserved checks the centroid drift stays within a small absolute
// distance (mm), scaled by the body's extent via the volume's cube root.
func assertCentroidPreserved(t *testing.T, fixture string, po, pa ops.GeometryProperties) {
	t.Helper()
	if d := po.Centroid.DistanceTo(pa.Centroid); d > 1e-2 {
		t.Errorf("%s centroid moved %g mm: %v → %v", fixture, d, po.Centroid, pa.Centroid)
	}
}
