// SPDX-License-Identifier: GPL-2.0-only

package blend

import "oblikovati.org/kernel/topo"

// patchToFilletFace adapts a resolveBlend result into the filletFace the hardened assembly consumes
// (assembleBody). The provider owns the surface + boundary loops; the extractor supplies the lineage
// (ADR-0043) since providers are topo-free. One adapter for every strangler entry so they never drift.
func patchToFilletFace(patch CornerBlendPatch, parent topo.Lineage) filletFace {
	return filletFace{surface: patch.Surface, loops: patch.Loops, parent: parent}
}
