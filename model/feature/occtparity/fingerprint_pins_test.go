// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

// TestByteIdentityFingerprints pins the canonical mesh fingerprint + volume of the prior-green corner
// bodies the CN-C8 brief names for do-no-harm (B3, N7, C2, C6, D5, D9, E4) plus C8 itself. CN-C8 changes no
// shared weld/retrim/geometry code (it is verification + a forensic receipt), so every one of these must
// stay bit-identical to the base worktree 6123169d. These VALUES were captured on base 6123169d and re-run
// on the CN-C8 HEAD; if a future shared-code edit perturbs any of them, this fails loud with the delta. The
// volumes match the brief's references (B3 190756.470897507, N7 963883.383205631) to the fingerprint's
// model-relative quantum.
func TestByteIdentityFingerprints(t *testing.T) {
	for _, tc := range byteIdentityPins() {
		t.Run(tc.name, func(t *testing.T) {
			fp := bodyMeshFingerprint(caseResultBody(t, tc.name))
			if fp.Hash != tc.hash || fp.Triangles != tc.tris {
				t.Fatalf("%s fingerprint drifted: hash=%#x tris=%d, want hash=%#x tris=%d (shared geometry changed)",
					tc.name, fp.Hash, fp.Triangles, tc.hash, tc.tris)
			}
			if rel := relErr(fp.Volume, tc.vol); rel > 1e-9 {
				t.Fatalf("%s volume %.9f != base %.9f (rel %.3g)", tc.name, fp.Volume, tc.vol, rel)
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
// worktree at 6123169d and confirmed bit-identical on the CN-C8 HEAD (see cnc8-report.md §byte-identity).
func byteIdentityPins() []fingerprintPin {
	return []fingerprintPin{
		{"B3", 190756.470862415, 31274, 0x2dff54b187389df4},
		{"N7", 963883.383133711, 31472, 0x8bc99405bb8efa01},
		{"C2", 510191.885540723, 121529, 0x1b3bc10e4d60136e},
		{"C6", 1559718.455929083, 121634, 0xe1e51a060a4c1b02},
		{"D5", 3432799.391739331, 64666, 0x7ef4f7950c6a410d},
		{"D9", 10302524.214101996, 81106, 0x2eecacfc3279f7d6},
		{"E4", 3460484.448938945, 59660, 0xf9912504516fdf8a},
		{"C8", 64858.504100833, 39992, 0x66606a97c42af3b2},
	}
}
