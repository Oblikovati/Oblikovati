// SPDX-License-Identifier: GPL-2.0-only

package occtparity

// bsplineHostPins are the wave-G BSpline-host fillet engine's greened cases (kernel/ops
// fillet_bspline_host*.go — edge-anchored section-plane Newton march against a
// geom.BSplineSurface host). Captured on this worktree's HEAD after the corpus merge; the two
// runout cases G5/G9 needed the reversedEndSeg srcEdge-preservation fix (fillet_blend_faces.go)
// before they certified as watertight — every other case here built clean on the first pass.
//
// J9 and B2 (the two CLOSED-rim cases) are NOT pinned here: they were briefly greened through
// periodicNurbsFaceMesh's widened "lacks a pcurve" gate, which turned out to be unsafe on
// ordinary imports (Oblikovati#585 — TestImportedNurbsDuctVolumeAndFolds regressed −1.18%
// volume on the committed bulged_duct fixture, a body no fillet ever touched). The gate was
// reverted and buildBsplineHostCanal now declines closed specs outright, so J9/B2 have no
// shipped body to pin — they are FAIL(faulty) again, exactly as before this engine existed.
func bsplineHostPins() []fingerprintPin {
	return []fingerprintPin{
		{"G5", 2291990.06452883, 53980, 0x44b42fb87bbe7347, ""},
		{"G7", 371103.26729947, 42266, 0x73c9c89c81888c64, ""},
		{"G9", 370841.255715241, 56226, 0xddc9feeceadad67a, ""},
		{"I5", 1454820.08081625, 36624, 0x221e85534b186275, ""},
		{"I7", 1454260.24945166, 57502, 0xd716c75d06baf3f2, ""},
		{"V6", 8965527.71587158, 71024, 0xa9e1a666cf7b9cf1, ""},
	}
}
