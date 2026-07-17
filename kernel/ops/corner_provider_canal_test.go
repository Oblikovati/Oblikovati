// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
)

// The canal tier (M6' C1-C3): Fits classifies by the RailLoop.Canal payload pointer; Build composes
// the offset-SSI spine + cross-section loft (geom.CanalCornerFill) into the rolling-ball corner patch
// and certifies it, or honest-rejects (→ coons4). Assertions are against the DRAWEXE oracle (area
// 90.194), never our own output. Supersedes corner_provider_plate_test.go (deleted with the plate
// stub, ADR-C3): canalProvider took the plate's old tier slot.

// TestCanalProviderFits pins the Fits truth table: true iff the loop carries a non-nil Canal payload
// AND is valence-4. A default (Canal==nil) valence-4 loop and a Canal-populated valence-3 loop must
// both decline — the payload pointer is the ONLY classification signal (ADR-C2; canal-corner-seam-
// architecture.md §2), not loop shape alone.
func TestCanalProviderFits(t *testing.T) {
	p := canalProvider{}

	unmarkedV4 := quarterCylLoop(t, 8) // Canal defaults to nil
	if p.Fits(unmarkedV4) {
		t.Fatal("canalProvider.Fits must be false for an unmarked (Canal==nil) valence-4 loop")
	}

	markedV3 := sphereTriLoop(t, 4)
	markedV3.Canal = &CanalCorner{Radius: 4}
	if p.Fits(markedV3) {
		t.Fatal("canalProvider.Fits must be false for a Canal-populated valence-3 loop (needs valence 4 too)")
	}

	markedV4 := quarterCylLoop(t, 8)
	markedV4.Canal = &CanalCorner{Radius: 5}
	if !p.Fits(markedV4) {
		t.Fatal("canalProvider.Fits must be true for a Canal-populated valence-4 loop")
	}
}

// TestCanalProviderBuildDeclinesOnIncompletePayload pins the do-no-harm floor: a loop marked Canal but
// WITHOUT the roll hosts / ends (an incomplete payload) makes geom.CanalCornerFill error (host count
// != 2), so Build honest-rejects (ok=false) → resolveBlend falls through to coons4. A solver that
// cannot build the surface never fabricates a patch (ADR-0051 ADR-3).
func TestCanalProviderBuildDeclinesOnIncompletePayload(t *testing.T) {
	loop := quarterCylLoop(t, 8)
	loop.Canal = &CanalCorner{Radius: 5} // no Rolls, no Ends
	p := canalProvider{}
	if !p.Fits(loop) {
		t.Fatal("fixture must Fit for this test to be meaningful")
	}
	if _, _, ok := p.Build(loop, blendScale()); ok {
		t.Fatal("canalProvider.Build must decline on an incomplete Canal payload (no roll hosts)")
	}
}

// The tier-order assertion (analyticSphere, canal, coons4, tri3) lives in TestResolveBlendTiersOrder;
// the B3/octant → BlendKindSphere and non-canal-valence-4 → BlendKindCoons4 cases live in
// TestResolveBlendSphereWins / TestResolveBlendCoons4 (all corner_resolve_test.go) — not duplicated.

// TestN7LoopResolvesCanal is the C3 gate at the resolveBlend seam: the real N7 fixture extracts a
// Canal-populated valence-4 loop that the canal tier now BUILDS and certifies (BlendKindCanal), with
// the emergent DRAWEXE-oracle area 90.194 within the C2-justified 0.05% and the Loops emitted on the
// RECEIVED rails (watertight — same rails coons4 would have used). The certificate is independently
// Valid at the model scale.
func TestN7LoopResolvesCanal(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok || loop.Valence() != 4 || loop.Canal == nil {
		t.Fatalf("extractCurvedCorner: want a Canal-marked valence-4 N7 loop; ok=%v valence=%d canal=%v", ok, loop.Valence(), loop.Canal != nil)
	}
	patch, ok := resolveBlend(loop, res)
	if !ok || patch.Kind != BlendKindCanal {
		t.Fatalf("N7 loop must resolve to the canal tier; ok=%v kind=%q", ok, patch.Kind)
	}
	if area := geom.SurfaceArea(patch.Surface); stdmath.Abs(area-90.194)/90.194 > 5e-4 {
		t.Fatalf("N7 canal area = %.5f, want 90.194 within 0.05%%", area)
	}
	assertLoopsAreReceivedRails(t, patch, loop)
}

// TestN7CanalCertificateValid witnesses the canal patch's certificate directly: the surface welds G1
// to its two fillet arms (the end cross-sections) AND is tangent to both roll hosts (the foot-loci),
// so all five fields pass at the model scale.
func TestN7CanalCertificateValid(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	loop, _ := extractCurvedCorner(w, arms, res)
	_, cert, ok := canalProvider{}.Build(loop, res)
	if !ok {
		t.Fatal("canalProvider.Build must succeed on the real N7 loop")
	}
	if !cert.Valid(res) {
		t.Fatalf("canal certificate not Valid: Closed=%v WeldsArms=%v NoFold=%v MaxDev=%.3e MaxAngleDev=%.3e",
			cert.Closed, cert.WeldsArms, cert.NoFold, cert.MaxDev, cert.MaxAngleDev)
	}
}

// TestCurvedCornerFaceAdmitsCanal proves the weld Kind-gate admits BlendKindCanal (mirrors the coons4
// arm of the whitelist switch): driving curvedCornerFace on the real N7 corner returns a fillet face
// whose surface is the CANAL BSpline patch — NOT the do-no-harm sphere fallback — so the canal patch
// is not silently dropped by the whitelist's default arm.
func TestCurvedCornerFaceAdmitsCanal(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	dummy, err := geom.NewSphere(w.center, 5) // the fallback surface, distinct from the canal BSpline
	if err != nil {
		t.Fatalf("dummy sphere: %v", err)
	}
	face, ok := curvedCornerFace(w, dummy, arms, res)
	if !ok {
		t.Fatal("curvedCornerFace declined the N7 canal corner")
	}
	if _, isBSpline := face.surface.(geom.BSplineSurface); !isBSpline {
		t.Fatalf("weld dropped the canal patch: face surface is %T, want geom.BSplineSurface (the canal)", face.surface)
	}
}

// assertLoopsAreReceivedRails proves the emitted patch Loops ARE the received rails (watertight weld):
// they are byte-for-byte railLoopToFilletLoops(loop) — the same boundary points and per-segment curves
// coons4 would have emitted, so the canal patch welds to the arm end-sections on the identical rails.
func assertLoopsAreReceivedRails(t *testing.T, patch CornerBlendPatch, loop RailLoop) {
	t.Helper()
	want := railLoopToFilletLoops(loop)
	if len(patch.Loops) != len(want) || len(want) == 0 {
		t.Fatalf("patch Loops count = %d, want %d (railLoopToFilletLoops)", len(patch.Loops), len(want))
	}
	got := patch.Loops[0]
	if len(got.pts) != len(want[0].pts) || len(got.pts) == 0 {
		t.Fatalf("loop point count = %d, want %d", len(got.pts), len(want[0].pts))
	}
	for i, p := range want[0].pts {
		if got.pts[i] != p {
			t.Fatalf("loop point %d = %v, want the received rail point %v — patch not on received rails", i, got.pts[i], p)
		}
	}
}
