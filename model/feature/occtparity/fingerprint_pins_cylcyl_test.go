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
		// RE-measured 2026-09-22 (review 3 C1): the classification now selects exactly ONE path, so a
		// charted face whose trim DEVELOPS keeps the structured grid — W8 and K4 are bit-identical to
		// their pre-#3517 pins again — and the boundary clearance reads a chord capped at the
		// covering's own cell. Every moved face's worst chord sagitta falls; ADR-0061 §R4.6 has it.
		// RE-measured on the rebase onto m48/csg-leftovers: #3551's covering merges two records of one
		// LOCATION into one vertex, which takes tens of triangles off the five bodies whose charted
		// faces carry a coincident pair. Volumes move 6.5e-12 to 1.5e-8 — merge noise, not geometry.
		{"K2", 905604.327694532112, 265462, 0x05563cdc34253d6f, ""},
		{"K3", 905603.366296867258, 209230, 0x0f661b61170a5556, ""},
		{"K4", 214184.268224860512, 98856, 0x872a76ba951d24e4, ""},
		{"B4", 1475254.845156956464, 125526, 0xef30cfa692ead306, "bfuseblend"},
		{"B5", 1021362.302084454568, 97832, 0xc8e632163110fd47, "bfuseblend"},
		// P1: the equal-parallel valley LINE seam (exact cylinder arm + the existing single-arm
		// runout weld); mesh volume vs DRAWEXE `vprops` 2135150: −0.001%.
		{"P1", 2135128.286760622635, 4348, 0x178e0228822f5040, ""},
	}
}
