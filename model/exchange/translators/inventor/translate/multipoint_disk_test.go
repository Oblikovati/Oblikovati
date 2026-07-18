// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	stdmath "math"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/persistence"
)

// TestMultipointDiskRebuildsAsAClosedSolid pins #33 on the part that exposed it. real_multipoint_disk
// is ReelToReel's TorquimeterDisk (byte-identical to the corpus file), whose cut names 52 regions at
// once — the biggest multi-region selection in the corpus.
//
// Merged into one 878-face 52-shell tool, the planar boolean returned a body it had ITSELF just
// classified invalid: booleanGeneral only attempts its CSG fallback when the operands fit under
// csgFallbackFaceLimit (256), and above it ships the known-bad planar result. The part came out at
// 1630 faces in 25 shells, NOT CLOSED, so the translator could only report it as a surface. Applying
// the 52 prisms one at a time — the same set operation, A−(B₁∪B₂) = ((A−B₁)−B₂) — gives 461 faces,
// ONE shell, a closed solid.
//
// This needs the REAL part: a synthetic plate with 40 clean bores does NOT reproduce it (merged stays
// valid and closed even at 1226 faces), so the failure is in this geometry, not in scale alone. The
// synthetic feature-level guard for the same fix is feature.TestMultiRegionCutKeepsTheAnalyticPath,
// which pins the analytic path the merged tool loses.
//
// The volume is Inventor 2027's own, measured via COM on this file: 7679 mm³. 5% covers the
// inscribed-N-gon tessellation bias on a part this curved; the defect made it unbuildable.
func TestMultipointDiskRebuildsAsAClosedSolid(t *testing.T) {
	d, err := ipt.Open(readCorpus(t, "real_multipoint_disk.ipt"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	document, _, err := buildPart(ws, filepath.Join(t.TempDir(), "disk.opd"), d, false)
	if err != nil {
		t.Fatalf("buildPart: %v", err)
	}
	def := document.Content().(*compdef.PartComponentDefinition)
	bodies := def.SurfaceBodies().All()
	if len(bodies) != 1 {
		t.Fatalf("rebuilt %d bodies, want 1", len(bodies))
	}
	b := bodies[0]
	rep := ops.Validate(b)
	if !b.IsSolid() || !rep.Closed {
		t.Errorf("disk rebuilt as IsSolid=%v Closed=%v (%d faces, %d shells) — want ONE closed solid; "+
			"a merged multi-lump tool leaves it open", b.IsSolid(), rep.Closed, len(b.Faces()), len(b.Shells()))
	}
	const inventorMm3 = 7679.0
	got := analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesLow).VolumeMm3
	if stdmath.Abs(got-inventorMm3)/inventorMm3 > 0.05 {
		t.Errorf("disk volume = %.0f mm³, want %.0f ±5%% (Inventor 2027)", got, inventorMm3)
	}
}
