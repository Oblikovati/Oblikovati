// SPDX-License-Identifier: GPL-2.0-only

package router

import "oblikovati.org/model/compdef"

// Construction (hidden, consumer-tied) work features auto-delete when their last consumer is
// deleted (#1849). Every delete that can remove a consumer — a part feature, a sketch, or a work
// datum — snapshots the construction datums that have a consumer BEFORE the delete, then calls
// pruneConstructionAfterDelete to remove those left with none. Snapshot-then-prune (rather than a
// blanket sweep) means a datum is removed exactly when its last consumer goes, never a datum that
// was created but not yet consumed.

// snapshotConstructionConsumers records the construction datums that currently have a consumer, or
// nil for a non-part host (an assembly datum has no construction-sketch lifecycle here).
func snapshotConstructionConsumers(part *compdef.PartComponentDefinition) []string {
	if part == nil {
		return nil
	}
	return part.ConstructionConsumerSnapshot()
}

// pruneConstructionAfterDelete removes any snapshot datum whose last consumer the just-applied
// delete removed, recomputing when any were pruned.
func pruneConstructionAfterDelete(part *compdef.PartComponentDefinition, snapshot []string) {
	if part == nil || len(snapshot) == 0 {
		return
	}
	if part.PruneConstructionOrphans(snapshot) > 0 {
		part.Recompute()
	}
}
