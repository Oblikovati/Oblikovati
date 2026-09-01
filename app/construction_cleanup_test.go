// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestConstructionCleanupWithoutActivePart: the construction-cleanup seam is a safe no-op when the
// active document is not a part (an assembly, or none) — snapshot yields nil, prune does nothing (#1849).
func TestConstructionCleanupWithoutActivePart(t *testing.T) {
	t.Parallel()
	s := assemblySession(t) // active document is an assembly, not a part
	if snap := s.ConstructionConsumerSnapshot(); snap != nil {
		t.Errorf("snapshot on a non-part active document should be nil, got %v", snap)
	}
	// A no-op even given a non-empty snapshot (no active part to prune against).
	s.PruneOrphanedConstructionDatums([]string{"plane/3"})
}
