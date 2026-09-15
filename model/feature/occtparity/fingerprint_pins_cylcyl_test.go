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
		// REBASELINED 2026-09-15 (#3517): the rim rebuild now carries the chart onto the face it re-winds,
		// so this body's periodic host reaches the GENERAL chart-driven mesher. Every per-face DRAWEXE
		// oracle, watertight gate and the scoreboard stay green; ADR-0061 carries the table.
		{"K2", 905603.825644632918, 264824, 0x4e16917bcc4f46be, ""},
		{"K3", 905602.694346644566, 208990, 0x1210f8f57f0e0f8f, ""},
		{"K4", 214184.739786441060, 76882, 0x56a071ee991b1730, ""},
		{"B4", 1475254.165703097824, 125282, 0x66742467d6e3fa8d, "bfuseblend"},
		{"B5", 1021361.833912160713, 97580, 0xab480d22fb74385b, "bfuseblend"},
		// P1: the equal-parallel valley LINE seam (exact cylinder arm + the existing single-arm
		// runout weld); mesh volume vs DRAWEXE `vprops` 2135150: −0.001%.
		{"P1", 2135128.286760622635, 4348, 0x178e0228822f5040, ""},
	}
}
