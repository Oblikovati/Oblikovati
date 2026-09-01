// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestBoxSharedCornerArea regresses the corner-solve orientation bug (G1 1b, corpus simple/P8,V8).
//
// Filleting two top edges of a box that share a corner (r=1) mutually trims the two rolling-ball
// cylinders into a MITER. On a natively-built box this is correct — OCCT's reference total is
// 145.137, and brep.SolidBlock yields 145.126. But on a STEP-IMPORTED box (reversed faces, inward
// plane normals) the same fillet came out at 150.14 (+3.4%): the miter's shared/outer face normals
// were read raw via planeNormal, ignoring face.Reversed(), so one arm's frame flipped — its cylinder
// overshot (8.43 vs 7.28) and the shared top face lost only one strip instead of two (20 vs 16).
// This is the corner-path analogue of the edgePlanarFaces fix (commit dbd28339); the fix routes the
// corner normals through the material-outward normal helper too. It also covers CornerRound, whose
// solveBlend sphere solve had the identical latent defect. Real user-facing bug: corner fillets on
// every imported STEP solid were wrong. Must stay green; never made green by loosening the tolerance.
func TestBoxSharedCornerArea(t *testing.T) {
	t.Parallel()
	const occtArea = 145.137 // OCCT checkprops -s for simple/P8 and simple/V8

	t.Run("native/miter", func(t *testing.T) {
		assertCornerArea(t, filletBoxCorner(t, mustBox5(t), blend.CornerMiter), occtArea)
	})
	t.Run("imported/miter", func(t *testing.T) {
		box := importOrientedBox5(t)
		assertCornerArea(t, filletBoxCorner(t, box, blend.CornerMiter), occtArea)
	})
	t.Run("imported/round", func(t *testing.T) {
		// CornerRound rounds the sharp third edge into a full sphere corner; its area differs from
		// the miter, but on the imported box solveBlend must still place the sphere on the material
		// side. We assert only that the result is watertight and near the native round's area, so a
		// flipped-normal sphere (which lands outside / self-intersects) is caught.
		nativeRound := filletBoxCorner(t, mustBox5(t), blend.CornerRound)
		importedRound := filletBoxCorner(t, importOrientedBox5(t), blend.CornerRound)
		want := query.BodyGeometryProperties(nativeRound, ops.PropertyQuality()).Area
		assertCornerArea(t, importedRound, want)
	})
}

// filletBoxCorner fillets the two top edges sharing vertex (5,0,5) — the +Y edge at x=5,z=5 and
// the +X edge at y=0,z=5 (the corpus P8/V8 picks) — at r=1 with the given corner strategy.
func filletBoxCorner(t *testing.T, b *topo.Body, strategy blend.CornerStrategy) *topo.Body {
	t.Helper()
	var picks []blend.EdgeFilletRadii
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if boxCornerEdge(a, c, math.P3(5, 0, 5), math.P3(5, 5, 5)) ||
			boxCornerEdge(a, c, math.P3(0, 0, 5), math.P3(5, 0, 5)) {
			picks = append(picks, blend.EdgeFilletRadii{Key: e.ReferenceKey(), R0: 1, R1: 1})
		}
	}
	if len(picks) != 2 {
		t.Fatalf("expected 2 corner-sharing top edges, found %d", len(picks))
	}
	res, err := blend.FilletEdgesCorner(b, picks, strategy, blend.FillConcaveOutward)
	if err != nil {
		t.Fatalf("FilletEdgesCorner(%v): %v", strategy, err)
	}
	return res
}

// boxCornerEdge reports whether the edge endpoints (a,c) match the segment (p,q) either way round.
func boxCornerEdge(a, c, p, q math.Point3) bool {
	return (a.DistanceTo(p) < 1e-6 && c.DistanceTo(q) < 1e-6) ||
		(a.DistanceTo(q) < 1e-6 && c.DistanceTo(p) < 1e-6)
}

// assertCornerArea checks the filleted body's total surface area is within OCCT's 1% (deps 0.01).
func assertCornerArea(t *testing.T, res *topo.Body, want float64) {
	t.Helper()
	assertWatertight(t, res, "box shared-corner fillet")
	got := query.BodyGeometryProperties(res, ops.PropertyQuality()).Area
	if rel := (got - want) / want; rel < -0.01 || rel > 0.01 {
		t.Fatalf("box shared-corner area %.4f, want %.4f within 1%% (rel %.4f)", got, want, rel)
	}
}

func mustBox5(t *testing.T) *topo.Body {
	t.Helper()
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(5, 5, 5), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return box
}

// importOrientedBox5 imports the box 5^3 STEP fixture (reversed faces, inward plane normals — the
// P8/V8 input solid), the orientation that exposed the corner-solve normal bug. The fixture is a
// copy of model/feature/occtparity/fixtures/simple/P8.step (only its FILE_NAME timestamp differs);
// it is package-local so the kernel test needs no cross-package fixture path. A hand-built reversed
// box would not exercise the STEP reader's SAME_SENSE-flag path that actually sets Reversed().
func importOrientedBox5(t *testing.T) *topo.Body {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "box5_corner_oriented.step"))
	if err != nil {
		t.Fatalf("read box5 fixture: %v", err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("import box5: %v (n=%d)", err, len(bodies))
	}
	return bodies[0]
}
