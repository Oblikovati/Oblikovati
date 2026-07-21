// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
)

// occtH6ConeArea is OCCT's per-face reference area for each of H6's two host cones (drawexe sprops on
// the 270° apex-collapsed cone sectors, apex (0,0,±250), 45° half-angle, base r=200): 133286 each,
// documented in h6-curved-retrim-rootcause.md §"Reproduction + ground truth".
const occtH6ConeArea = 133286.0

// TestH6HostConesTessellateToOCCTArea is the ROOT-1 regression on the REAL fixture: H6's imported
// (pre-fillet) body carries two geometrically congruent 270° cone sectors in MIRROR orientations. The
// apex-collapsed-sector tessellation bug meshed them to DIFFERENT areas — one via the (u,v) path, the
// other read as seam-crossing and over-covered as a full 2π cone (167927, ×1.26 — the tape measure) —
// even though a cone is developable and both integrate to 133286. coneApexSectorMesh fans both from
// the apex, so both now equal the OCCT oracle. This asserts the DRAWEXE-proven per-face defect
// directly (not just the whole-body area, which H6's inverted fillet — ROOT 2 — still corrupts).
func TestH6HostConesTessellateToOCCTArea(t *testing.T) {
	body, err := importInput(filepath.Join(CorpusFixtureDir(), "simple", "H6.step"))
	if err != nil {
		t.Fatalf("import H6: %v", err)
	}
	var areas []float64
	for _, f := range body.Faces() {
		if _, isCone := f.Geometry().(geom.Cone); !isCone {
			continue
		}
		areas = append(areas, ops.MeshArea(ops.TessellateFace(f, ops.PropertyQuality())))
	}
	if len(areas) != 2 {
		t.Fatalf("H6 imported body has %d cone faces, want 2 (the two host cone sectors)", len(areas))
	}
	for i, a := range areas {
		if rel := stdmath.Abs(a-occtH6ConeArea) / occtH6ConeArea; rel > 1e-3 {
			t.Errorf("H6 cone face %d area %.3f != OCCT %.1f (rel %.3g%%); apex-sector tessellation defect",
				i, a, occtH6ConeArea, 100*rel)
		}
	}
	if rel := stdmath.Abs(areas[0]-areas[1]) / occtH6ConeArea; rel > 1e-6 {
		t.Errorf("H6's two congruent cones tessellate to different areas %.3f vs %.3f (orientation-dependent)",
			areas[0], areas[1])
	}
}
