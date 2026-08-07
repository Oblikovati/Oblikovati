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

// TestRevolveGraphFallbackBuildsEncoderWheel guards the revolve graph-sketch fallback: EncoderWheel's
// 13-line profile is SPLIT by incidence into three isolated single-line sketches, so no revolve binds
// and the part fell back to its non-manifold display mesh (SURFACE, 3520 mm3 = 1.97x — an open-shell
// artifact). The intact node graph keeps the profile whole with a vertical in-profile centreline, so
// the revolve now builds parametrically: a solid ~1785 mm3 (Inventor's volume) with the revolve
// feature preserved. Corpus-gated (no generated fixture reproduces the incidence split); point
// IPT_CORPUS at the ReelToReel Mechanical directory to enable.
func TestRevolveGraphFallbackBuildsEncoderWheel(t *testing.T) {
	dir := os.Getenv("IPT_CORPUS")
	if dir == "" {
		dir = `P:\ReelToReel\Mechanical`
	}
	data, err := os.ReadFile(filepath.Join(dir, "EncoderWheel.ipt"))
	if err != nil {
		t.Skipf("corpus part not available (%v); set IPT_CORPUS", err)
	}
	out := filepath.Join(t.TempDir(), "wheel.opd")
	if _, err := FromInventor(data, out); err != nil {
		t.Fatalf("FromInventor: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(out, true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	def := reopened.Content().(*compdef.PartComponentDefinition)
	if def.Features().Count() == 0 {
		t.Fatal("no features — revolve not rebuilt parametrically (fell back to mesh)")
	}
	bodies := def.SurfaceBodies().All()
	if len(bodies) != 1 || !bodies[0].IsSolid() {
		t.Fatalf("want 1 solid body, got %d (solid=%v) — the graph revolve did not close", len(bodies), len(bodies) == 1 && bodies[0].IsSolid())
	}
	mp := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesLow)
	const oracle = 1785.0 // Inventor STL volume, mm^3
	if math.Abs(mp.VolumeMm3-oracle) > 0.1*oracle {
		t.Errorf("revolved volume = %.0f mm^3, want ~%.0f (within 10%%)", mp.VolumeMm3, oracle)
	}
}
