// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

// TestTapePathContainmentSelectsThinRibbons pins the containment fix for TapePath: its single extrude
// consumes a 60-cell arrangement whose tape-guide faces are thin ribbons. Two things used to send the
// whole sketch to the (wrong) curve-set fallback — a degenerate open sliver loop in the patch, and
// degenerate zero-width cells that could not be given an interior test point. The fallback selected
// 13 cells (4.83 of 20.49 cm²) and built the extrude at 7.6x its true volume (3069 vs Inventor's
// 403 mm³). With open material slivers dropped and un-pointable cells skipped (regionpoly.go),
// containment selects only the thin ribbons (~0.55 cm²), so the body lands near the oracle.
//
// Corpus-gated: the geometry only exists in the real part (no generated fixture reproduces the
// 60-cell thin-ribbon arrangement), so the test skips where the corpus is absent (CI) and runs on a
// dev checkout of the ReelToReel set. Point IPT_CORPUS at the Mechanical directory to enable it.
func TestTapePathContainmentSelectsThinRibbons(t *testing.T) {
	dir := os.Getenv("IPT_CORPUS")
	if dir == "" {
		dir = `P:\ReelToReel\Mechanical`
	}
	data, err := os.ReadFile(filepath.Join(dir, "TapePath.ipt"))
	if err != nil {
		t.Skipf("corpus part not available (%v); set IPT_CORPUS to the ReelToReel Mechanical dir", err)
	}
	out := filepath.Join(t.TempDir(), "tapepath.opd")
	if _, err := FromInventor(data, out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	bodies := def.SurfaceBodies().All()
	if len(bodies) == 0 {
		t.Fatalf("no body built")
	}
	vol := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesLow).VolumeMm3
	const oracle = 403.0 // Inventor STL volume, mm³
	// The old curve-set fallback shipped 3069 mm³ (7.6x). Guard against any regression back toward
	// that over-build while allowing the ~14% under-fill from the one ribbon whose corner gap exceeds
	// the containment join tolerance.
	if vol > 1.5*oracle {
		t.Errorf("TapePath volume = %.0f mm³, want <= %.0f (the 7.6x curve-set over-build must not return)", vol, 1.5*oracle)
	}
	if math.Abs(vol-oracle) > 0.20*oracle {
		t.Errorf("TapePath volume = %.0f mm³, want within 20%% of Inventor's %.0f", vol, oracle)
	}
}
