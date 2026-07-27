// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// Reflex single-arm curved RUNOUT gate — OCCT blend/simple/C5 and D8, the two 270° (reflex) single-arm
// runouts. C5 is a ¾-frustum cone host (pcone 90 40 150 270, arc r=40 top); D8 is a ¾-sphere host
// (psphere 15 -60 60 270 ×10 = R150, arc r=75 latitude). Both were floored by the old blanket reflex reject;
// they now weld once the host contact rail keeps the MAJOR sector (reflexContactRail/subArcMajor) and, for the
// sphere host, the shell is seeded to mesh the material zone (orientRunoutSphereHost). This mirrors D9's
// reflex tessellation-fold gate: it asserts a watertight 6-face solid, every face fold-free with a finite
// positive area, the summed mesh area equals OCCT's whole-body total within the corpus deps (0.01), and — the
// crux of the fix — that the curved host holds its true MAJOR sector rather than the minor complement a
// three-point re-fit would snap to. Tessellation correctness is the repo's highest priority (CLAUDE.md).

const reflexRunoutDeps = 0.01 // corpus whole-body deps gate (1%)

// TestC5WholeBodyWatertight asserts the reflex cone-host runout welds a watertight 6-face manifold solid.
func TestC5WholeBodyWatertight(t *testing.T) {
	assertWatertight(t, "C5", caseResultBody(t, "C5"), 6)
}

// TestD8WholeBodyWatertight asserts the reflex sphere-host runout welds a watertight 6-face manifold solid.
func TestD8WholeBodyWatertight(t *testing.T) {
	assertWatertight(t, "D8", caseResultBody(t, "D8"), 6)
}

// TestC5TessellationFoldGate meshes every C5 face fold-free and pins the summed area to OCCT's 90385.8.
func TestC5TessellationFoldGate(t *testing.T) {
	assertReflexRunoutMesh(t, "C5", caseResultBody(t, "C5"), 90385.8)
}

// TestD8TessellationFoldGate meshes every D8 face fold-free and pins the summed area to OCCT's 276716.
func TestD8TessellationFoldGate(t *testing.T) {
	assertReflexRunoutMesh(t, "D8", caseResultBody(t, "D8"), 276716)
}

// TestC5ConeHostMajorSector asserts the receded cone host holds its 270° MAJOR sector. A recession can only
// REMOVE area, so the host must mesh strictly LESS than the full un-receded 270° frustum lateral sector
// (48433); the minor-rail bug wound a self-overlapping region reading 62711 (> the full sector) — caught here.
func TestC5ConeHostMajorSector(t *testing.T) {
	f := singleSurfaceFace[geom.Cone](t, caseResultBody(t, "C5"), "cone host")
	const fullSector270 = 48433.0 // π·(90+40)·√(150²+50²)·(270/360)
	if a := faceMeshArea2(f); a <= 0.5*fullSector270 || a >= fullSector270 {
		t.Fatalf("C5 cone host area %.1f, want a receded MAJOR sector in (%.0f, %.0f) — the minor-rail bug read 62711",
			a, 0.5*fullSector270, fullSector270)
	}
}

// TestD8SphereHostMaterialZone asserts the sphere host meshes the >hemisphere MATERIAL zone (lat −60..60 over
// 270° ≈ 183596), NOT the complement (~100313). The discriminator is a true hemisphere (2πR² = 141371): the
// material zone exceeds it, the complement falls below it — so a winding regression (orientRunoutSphereHost)
// that filled the complement fails loud here.
func TestD8SphereHostMaterialZone(t *testing.T) {
	f := singleSurfaceFace[geom.Sphere](t, caseResultBody(t, "D8"), "sphere host")
	sph := f.Geometry().(geom.Sphere)
	hemisphere := 2 * stdmath.Pi * sph.Radius * sph.Radius
	if a := faceMeshArea2(f); a <= hemisphere {
		t.Fatalf("D8 sphere host area %.1f, want the >hemisphere material zone (> %.0f); the complement reads ~100313", a, hemisphere)
	}
}

// assertReflexRunoutMesh meshes every face of a reflex runout result: each must be fold-free with a finite
// positive area, and the summed area must equal OCCT's whole-body total within reflexRunoutDeps.
func assertReflexRunoutMesh(t *testing.T, name string, body *topo.Body, wantTotal float64) {
	t.Helper()
	total := 0.0
	for _, f := range body.Faces() {
		m := ops.TessellateFace(f, ops.PropertyQuality())
		area := ops.MeshArea(m)
		total += area
		if area <= 0 || stdmath.IsInf(area, 0) || stdmath.IsNaN(area) {
			t.Fatalf("%s %T face meshed to %.4f, want a finite positive area", name, f.Geometry(), area)
		}
		assertFaceFoldFreeAtEveryQuality(t, name, f, m)
	}
	if rel := stdmath.Abs(total-wantTotal) / wantTotal; rel > reflexRunoutDeps {
		t.Fatalf("%s total mesh area %.2f, want OCCT %.1f within deps %.2f (rel %.4f)", name, total, wantTotal, reflexRunoutDeps, rel)
	}
}

// singleSurfaceFace returns the unique result face whose geometry is surface type S, failing when none or
// more than one matches — the reflex runouts carry exactly one cone (C5) or one sphere (D8) host.
func singleSurfaceFace[S geom.Surface](t *testing.T, body *topo.Body, label string) *topo.Face {
	t.Helper()
	var found *topo.Face
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(S); !ok {
			continue
		}
		if found != nil {
			t.Fatalf("body carries more than one %s face", label)
		}
		found = f
	}
	if found == nil {
		t.Fatalf("body carries no %s face", label)
	}
	return found
}
