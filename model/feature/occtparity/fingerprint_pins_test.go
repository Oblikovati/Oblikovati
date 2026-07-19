// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

// TestByteIdentityFingerprints pins the canonical mesh fingerprint + volume of the prior-green corner
// bodies the CN-C8 brief names for do-no-harm (B3, N7, C2, C6, D5, D9, E4) plus C8 itself. CN-C8 changes no
// shared weld/retrim/geometry code (it is verification + a forensic receipt), so every one of these must
// stay bit-identical to the base worktree 6123169d. These VALUES were captured on base 6123169d and re-run
// on the CN-C8 HEAD; if a future shared-code edit perturbs any of them, this fails loud with the delta. The
// RAW volumes match the brief's references exactly (B3 190756.470897507, N7 963883.383205631).
func TestByteIdentityFingerprints(t *testing.T) {
	for _, tc := range byteIdentityPins() {
		t.Run(tc.name, func(t *testing.T) {
			fp := bodyMeshFingerprint(caseResultBody(t, tc.name))
			if fp.Hash != tc.hash || fp.Triangles != tc.tris {
				t.Fatalf("%s fingerprint drifted: hash=%#x tris=%d, want hash=%#x tris=%d (shared geometry changed)",
					tc.name, fp.Hash, fp.Triangles, tc.hash, tc.tris)
			}
			// RAW volume compared at rel 1e-9 — re-run noise is ~1e-13, so this has ~4 decades of margin
			// and (unlike the earlier quantum-rounded volume) cannot false-red on a quantization boundary.
			if rel := relErr(fp.Volume, tc.vol); rel > 1e-9 {
				t.Fatalf("%s volume %.12f != base %.12f (rel %.3g)", tc.name, fp.Volume, tc.vol, rel)
			}
		})
	}
}

// fingerprintPin is one body's captured order-independent fingerprint.
type fingerprintPin struct {
	name string
	vol  float64
	tris int
	hash uint64
}

// byteIdentityPins are the base-6123169d fingerprints, captured by running bodyMeshFingerprint in a git
// worktree at 6123169d and confirmed bit-identical on the CN-C8 HEAD (see cnc8-report.md §byte-identity)
// and again on the CN6 HEAD (cn6-report.md §byte-identity; CN6 is a test-harness-only change).
//
// CROSS-PLATFORM RISK (record only — do NOT fix here; for the pre-PR cross-platform-stability pass,
// CLAUDE.md hard-coded-values rule): these are hard-coded 64-bit hashes of quantized tessellated
// coordinates. arm64 FMA/libm rounding could shift a vertex across a 1e-6·scale quantization boundary
// and flip a hash → a FALSE red off linux/amd64. Fine for the corpus's linux/amd64 runs. The pre-PR
// pass must re-capture on arm64 and either confirm identical or widen the quantum / drop the hash pins
// to the (rounding-robust) volume+triangle-count pins there. No arm64 re-capture is attempted here.
func byteIdentityPins() []fingerprintPin {
	return []fingerprintPin{
		{"B3", 190756.470897506602, 31274, 0x2dff54b187389df4},
		{"N7", 963883.383205630700, 31472, 0x8bc99405bb8efa01},
		{"C2", 510191.885601512506, 121529, 0x1b3bc10e4d60136e},
		{"C6", 1559718.455869767116, 121634, 0xe1e51a060a4c1b02},
		{"D5", 3432799.391629283316, 64666, 0x7ef4f7950c6a410d},
		{"D9", 10302524.214111814275, 81106, 0x2eecacfc3279f7d6},
		{"E4", 3460484.448986444157, 59660, 0xf9912504516fdf8a},
		{"C8", 64858.504095408265, 39992, 0x66606a97c42af3b2},
		// The 8 single-arm curved-RUNOUT greens (R1 fb1b0ca4 · R2a ac0249e9 · R2b 3f09eafe · R3
		// 455b9543): B6/C9/C1 perpendicular, M7 plane-host inner-loop, C5/D8 reflex 270° major-sector,
		// D4/E3 oblique cap. Captured on the R3 HEAD 455b9543; they lock the whole single-arm runout
		// construction so any later slice that touches fillet_curved_single_runout*.go fails loud if it
		// perturbs a runout body. Same cross-platform-risk caveat as above applies.
		{"B6", 585938.532761026, 2564, 0x98fea16cb0b47c11},
		{"C9", 74878.585980316, 33282, 0x83be9eb887bda640},
		{"C1", 516328.521818100, 91108, 0x367226063c9789b0},
		{"M7", 1070772.658316682, 1556, 0xbda4ca6e02f5e3fa},
		{"C5", 1565080.392898256, 67716, 0x481e407c7b323a88},
		{"D8", 10309699.178065298, 46272, 0x47fb6c5622612116},
		{"D4", 3434287.473928707, 23974, 0x34f745da81d00eaf},
		{"E3", 3478295.925438132, 52086, 0x8309a22efbfc928f},
	}
}
