// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// STL-vs-STEP volume cross-check (the user's standing STL-oracle rule). It is the guard that would
// have caught the J2 large-sphere-zone tessellation bug and any future sphere-zone mis-measurement:
// our STEP-import tessellated volume, an INDEPENDENT OCCT STL mesh of the same solid, and OCCT's
// analytic vprops must all agree. Before the fix J2's zone STEP-imported at volume 30334 (the small
// north cap) — a 94% error this cross-check flags loud; after it, all three agree to ~1e-3.

// occtVpropsJ2Input is OCCT's analytic volume of J2's INPUT solid (psphere s 5 -90 45; tscale ·10):
// DRAWEXE `vprops` reports Mass=493200 with relative error 3e-16 (test-utilities/occt-blend/oracle),
// matching the closed-form zone integral π∫[−R,zc](R²−z²)dz = 493197.8 (R=50, zc=35.3553). Ground truth.
const occtVpropsJ2Input = 493200.0

// stepZoneVolTol bounds our STEP-import tessellated volume against OCCT's analytic value: our fine
// PropertyQuality mesh must be accurate (the fix lands it at ~9e-5). A model-relative fraction.
const stepZoneVolTol = 1e-3

// stlZoneVolTol bounds the committed OCCT STL oracle (J2_input.stl, incmesh deflection 0.05) and its
// agreement with our STEP mesh: an inscribed linear-deflection mesh under-reports the sphere zone's
// bulge by ~1.4e-3, so its tolerance is looser than the STEP mesh's. Still ~700× tighter than the bug.
const stlZoneVolTol = 3e-3

// TestJ2ZoneStlStepVolumeParity is the reusable STL-vs-STEP volume oracle instantiated for J2: it
// asserts the STEP-import tessellated volume, the OCCT STL mesh volume, and OCCT vprops all agree.
func TestJ2ZoneStlStepVolumeParity(t *testing.T) {
	stepVol := stepImportVolume(t, "simple/J2.step")
	stlVol := stlMeshVolume(t, "simple/J2_input.stl")
	t.Logf("J2 input zone volume — OCCT vprops=%.1f  STEP-import=%.1f  STL-import=%.1f", occtVpropsJ2Input, stepVol, stlVol)
	assertRelWithin(t, "STEP-import vs OCCT vprops", stepVol, occtVpropsJ2Input, stepZoneVolTol)
	assertRelWithin(t, "STL-import vs OCCT vprops", stlVol, occtVpropsJ2Input, stlZoneVolTol)
	assertRelWithin(t, "STEP-import vs STL-import", stepVol, stlVol, stlZoneVolTol)
}

// stepImportVolume imports a corpus STEP solid and returns its tessellated volume at PropertyQuality —
// the property-grade volume the corpus itself scores areas at.
func stepImportVolume(t *testing.T, rel string) float64 {
	t.Helper()
	body, err := importInput(filepath.Join(CorpusFixtureDir(), rel))
	if err != nil {
		t.Fatalf("import %s: %v", rel, err)
	}
	return ops.BodyGeometryProperties(body, ops.PropertyQuality()).Volume
}

// stlMeshVolume decodes an STL through kernel/exchange/meshio and returns the enclosed volume of the
// triangle soup by the divergence-theorem signed-tetra sum — an oracle INDEPENDENT of our STEP
// tessellation (OCCT wrote the mesh with outward winding, so the magnitude is the enclosed volume).
func stlMeshVolume(t *testing.T, rel string) float64 {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(CorpusFixtureDir(), rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	raw, err := meshio.DecodeSTL(data)
	if err != nil {
		t.Fatalf("DecodeSTL %s: %v", rel, err)
	}
	return rawMeshVolume(raw)
}

// rawMeshVolume is the enclosed volume (magnitude) of a decoded triangle soup by the divergence theorem.
func rawMeshVolume(raw meshio.RawMesh) float64 {
	o := math.P3(0, 0, 0)
	var v float64
	for _, tri := range raw.Tris {
		a, b, c := raw.Verts[tri[0]], raw.Verts[tri[1]], raw.Verts[tri[2]]
		v += float64(o.VectorTo(a).Dot(o.VectorTo(b).Cross(o.VectorTo(c)))) / 6
	}
	return stdmath.Abs(v)
}

// assertRelWithin fails when |got−want|/|want| exceeds tol (a model-relative fraction).
func assertRelWithin(t *testing.T, what string, got, want, tol float64) {
	t.Helper()
	if rel := stdmath.Abs(got-want) / stdmath.Abs(want); rel > tol {
		t.Fatalf("%s: %.4f vs %.4f (rel %.3g > tol %.1g)", what, got, want, rel, tol)
	}
}
