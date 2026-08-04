// SPDX-License-Identifier: GPL-2.0-only

package occtparity

// cylCylSeamPins pin the five closed Cylinder∧Cylinder SSI-seam canal greens
// (kernel/ops/fillet_cylcyl_seam*.go: K2/K3 the orthogonal bore's two exit rims, K4 the oblique
// over-radius cutter, bfuseblend/B4 the orthogonal fuse's two loops in one op, bfuseblend/B5 the
// oblique fuse) so any later slice that touches the seam station math, the canal loft, or the
// closed-rim rebuild fails loud if it perturbs a body. Captured on this branch's first green
// (per-face DRAWEXE receipts in cylcyl_seam_perface_test.go; mesh volume vs DRAWEXE `vprops`
// 1.e-12: K2/K3 −0.040%, K4 −0.0027%, B4 −0.028%, B5 −0.083% — the inscribed-mesh deficit).
// The arm64 cross-platform caveat of fingerprint_pins_test.go applies verbatim.
func cylCylSeamPins() []fingerprintPin {
	return []fingerprintPin{
		{"K2", 905236.670983666671, 162456, 0xfd37f76a44fc4ec2, ""},
		{"K3", 905236.672781282454, 162456, 0x8d9da0e0de8b7104, ""},
		{"K4", 214184.268224860512, 98856, 0x872a76ba951d24e4, ""},
		{"B4", 1474885.299245121889, 78432, 0x1d5f7a06d418c2e5, "bfuseblend"},
		{"B5", 1020545.699659498641, 47120, 0x2987b17b9ee2b5eb, "bfuseblend"},
		// P1: the equal-parallel valley LINE seam (exact cylinder arm + the existing single-arm
		// runout weld); mesh volume vs DRAWEXE `vprops` 2135150: −0.001%.
		{"P1", 2135128.286760622635, 4348, 0x178e0228822f5040, ""},
	}
}
