// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

// squareRailLoop builds a valence-n closed RailLoop from unit line-segment sides — a minimal fixture for
// exercising a provider's Fits/Build gates without importing a real extractor. With n=4 it satisfies
// canalStationProvider's valence check so the station-payload branches can be reached.
func squareRailLoop(n int) RailLoop {
	corners := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0), math.P3(0.5, -1, 0)}
	sides := make([]Side, n)
	for i := range n {
		sides[i] = Side{Curve: geom.NewLineSegment(corners[i%len(corners)], corners[(i+1)%len(corners)]), Cont: G0}
	}
	return RailLoop{Sides: sides}
}

// TestCanalStationProviderBuildRejectsNilStations pins reject-path 1 (U4-4b): a loop with NO station
// payload is declined by both Fits and Build (the Stations pointer is the sole recognition signal, so a
// non-core loop never reaches the loft) — the do-no-harm invariant that keeps every non-core loop on
// coons4.
func TestCanalStationProviderBuildRejectsNilStations(t *testing.T) {
	t.Parallel()
	loop := squareRailLoop(4) // valence 4 but Stations == nil
	if (canalStationProvider{}).Fits(loop) {
		t.Errorf("Fits(nil-Stations valence-4 loop) = true, want false")
	}
	if _, _, ok := (canalStationProvider{}).Build(loop, tol.ForSize(1)); ok {
		t.Errorf("Build(nil-Stations loop) ok = true, want false (no station payload to loft)")
	}
}

// TestCanalStationProviderBuildRejectsWrongValence pins reject-path 2 (U4-4b): even WITH a station
// payload, a loop that is not valence-4 is declined — a canal panel is a four-sided (A-rim / seam / B-rim
// / seam) loop, never a triangle.
func TestCanalStationProviderBuildRejectsWrongValence(t *testing.T) {
	t.Parallel()
	loop := squareRailLoop(3)
	loop.Stations = &CanalStationFill{
		Centers: []math.Point3{math.P3(0, 0, 0), math.P3(0, 0, 1)},
		FeetA:   []math.Point3{math.P3(5, 0, 0), math.P3(5, 0, 1)},
		FeetB:   []math.Point3{math.P3(0, 5, 0), math.P3(0, 5, 1)},
		Radius:  5,
	}
	if (canalStationProvider{}).Fits(loop) {
		t.Errorf("Fits(valence-3 loop) = true, want false")
	}
	if _, _, ok := (canalStationProvider{}).Build(loop, tol.ForSize(1)); ok {
		t.Errorf("Build(valence-3 loop) ok = true, want false (canal panel is valence-4)")
	}
}

// TestCanalStationProviderBuildRejectsOffRadiusStation pins reject-path 3 (U4-4b): a valence-4 loop whose
// station payload carries a foot NOT at the rolling-ball radius is declined by LoftCanalStations' own
// fidelity gate (assertFootAtRadius) — a mis-supplied station is a construction bug, honest-rejected
// (ok=false) so resolveBlend falls through to coons4, never lofting a non-envelope surface.
func TestCanalStationProviderBuildRejectsOffRadiusStation(t *testing.T) {
	t.Parallel()
	loop := squareRailLoop(4)
	loop.Stations = &CanalStationFill{
		Centers: []math.Point3{math.P3(0, 0, 0), math.P3(0, 0, 1)},
		FeetA:   []math.Point3{math.P3(5, 0, 0), math.P3(5, 0, 1)},
		FeetB:   []math.Point3{math.P3(0, 5, 0), math.P3(0, 99, 1)}, // 2nd B foot is 99 off, not radius 5
		Radius:  5,
	}
	if !(canalStationProvider{}).Fits(loop) {
		t.Fatalf("Fits(valence-4 + Stations) = false, want true (the payload branches must be reachable)")
	}
	if _, _, ok := (canalStationProvider{}).Build(loop, tol.ForSize(1)); ok {
		t.Errorf("Build(off-radius station) ok = true, want false (LoftCanalStations must decline a non-envelope foot)")
	}
}
