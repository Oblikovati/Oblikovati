// SPDX-License-Identifier: GPL-2.0-only

package ops

import "oblikovati.org/kernel/topo"

// FillInternalVoids returns b with its internal void (cavity) shells removed, so fully
// enclosed cavities become solid material — the hole-patching path for shrinkwrap
// simplification (M11-F06). A body with no voids is returned unchanged (same pointer).
//
// Only cavities disconnected from the boundary are filled: a through-hole's wall
// belongs to the outer shell, so it is untouched here and is eliminated instead by
// envelope replacement (or, later, by cap synthesis). Surface (open-shell) bodies pass
// through unchanged.
//
// Example: solid := ops.FillInternalVoids(hollowCasting, ops.DefaultQuality())
func FillInternalVoids(b *topo.Body, q Quality) *topo.Body {
	shells := b.Shells()
	kept := make([]*topo.Shell, 0, len(shells))
	for _, sh := range shells {
		if !ShellIsVoid(sh, q) {
			kept = append(kept, sh)
		}
	}
	if len(kept) == len(shells) {
		return b // no void shells to drop
	}
	return topo.BodyFromShells(b.Lineage(), b.IsSolid(), kept...)
}
