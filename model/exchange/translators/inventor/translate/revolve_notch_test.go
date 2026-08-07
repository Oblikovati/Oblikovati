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

// TestRevolveKernelFallbackBuildsTorquimeterWheel guards the kernel revolve fallback: TorquimeterWheel's
// profile has a notch whose mouth points land on the INTERIOR of a through-edge (a T-junction), so the
// ipt line-ring gate (isClosedRing, a head-to-tail walk) can't close it and the part fell back to its
// non-manifold mesh (SURFACE 1092 mm3 = 1.98x). The kernel arranges that profile correctly (splitting
// the through-edge at the notch), so revolving the kernel's own profile builds the solid: ~552 mm3
// (Inventor). Corpus-gated (no generated fixture reproduces the notch T-junction); set IPT_CORPUS.
func TestRevolveKernelFallbackBuildsTorquimeterWheel(t *testing.T) {
	dir := os.Getenv("IPT_CORPUS")
	if dir == "" {
		dir = `P:\ReelToReel\Mechanical`
	}
	data, err := os.ReadFile(filepath.Join(dir, "TorquimeterWheel.ipt"))
	if err != nil {
		t.Skipf("corpus part not available (%v); set IPT_CORPUS", err)
	}
	out := filepath.Join(t.TempDir(), "tw.opd")
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
		t.Fatal("no features — revolve not rebuilt (fell back to mesh)")
	}
	bodies := def.SurfaceBodies().All()
	if len(bodies) != 1 || !bodies[0].IsSolid() {
		t.Fatalf("want 1 solid body, got %d (solid=%v)", len(bodies), len(bodies) == 1 && bodies[0].IsSolid())
	}
	mp := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesLow)
	const oracle = 552.0 // Inventor STL volume, mm^3
	if math.Abs(mp.VolumeMm3-oracle) > 0.1*oracle {
		t.Errorf("revolved volume = %.0f mm^3, want ~%.0f (within 10%%)", mp.VolumeMm3, oracle)
	}
}
