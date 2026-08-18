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

// TestRevolveConstructionAxisBuildsPressureRoller guards two extensions of the kernel revolve
// fallback. PressureRoller's centreline is a vertical CONSTRUCTION line at x=0 (not the isolated line
// RevolveAxisLine keys on — a real internal edge at x=0.8 also reads as isolated, making it
// ambiguous), and that internal edge splits the profile into TWO closed regions (an annulus 0.25-0.8
// and 0.8-1.3). revolveAxisIndex falls back to the lone vertical construction line, and the fallback
// revolves BOTH closed profiles and unions them, yielding the full roller ~4600 mm3 (Inventor). It
// built a non-manifold mesh before (SURFACE 9106 = 1.98x). Corpus-gated; set IPT_CORPUS.
func TestRevolveConstructionAxisBuildsPressureRoller(t *testing.T) {
	dir := os.Getenv("IPT_CORPUS")
	if dir == "" {
		dir = `P:\ReelToReel\Mechanical`
	}
	data, err := os.ReadFile(filepath.Join(dir, "PressureRoller.ipt"))
	if err != nil {
		t.Skipf("corpus part not available (%v); set IPT_CORPUS", err)
	}
	out := filepath.Join(t.TempDir(), "pr.opd")
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
	if len(bodies) != 1 || !bodies[0].IsSolid() {
		t.Fatalf("want 1 solid body, got %d (solid=%v)", len(bodies), len(bodies) == 1 && bodies[0].IsSolid())
	}
	mp := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesLow)
	const oracle = 4600.0 // Inventor STL volume, mm^3
	if math.Abs(mp.VolumeMm3-oracle) > 0.1*oracle {
		t.Errorf("revolved volume = %.0f mm^3, want ~%.0f (within 10%%)", mp.VolumeMm3, oracle)
	}
}
