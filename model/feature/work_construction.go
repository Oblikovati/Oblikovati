// SPDX-License-Identifier: GPL-2.0-only

package feature

// Construction (hidden, consumer-tied) work features have Inventor's distinct lifecycle: a
// construction datum is auto-deleted when its last consumer is deleted; with no consumers it is the
// creator's responsibility to delete it (#1849). The host drives that lifecycle around a delete: it
// snapshots the construction datums that HAVE a consumer, applies the delete + recompute, then prunes
// the snapshot's datums that now have none — so a construction datum is removed exactly when its last
// consumer goes, and a never-consumed datum is never auto-removed. This file exposes the datum-side
// primitives; the host supplies the non-datum consumers (sketches, part features) via a predicate.

// ConstructionRefs returns the reference of every live user construction datum (plane/axis/point).
// Origin/coordinate-system datums are never construction; deleted datums are skipped.
func (g *WorkGeometry) ConstructionRefs() []WorkRef {
	var refs []WorkRef
	for i := 0; i < g.planes.Count(); i++ {
		if p := g.planes.Item(i); liveConstruction(p.construction, p.deleted, p.coordinateSystem) {
			refs = append(refs, p.key)
		}
	}
	for i := 0; i < g.axes.Count(); i++ {
		if a := g.axes.Item(i); liveConstruction(a.construction, a.deleted, a.coordinateSystem) {
			refs = append(refs, a.key)
		}
	}
	for i := 0; i < g.points.Count(); i++ {
		if p := g.points.Item(i); liveConstruction(p.construction, p.deleted, p.coordinateSystem) {
			refs = append(refs, p.key)
		}
	}
	return refs
}

// liveConstruction reports whether a datum is a live user construction datum: flagged construction,
// not tombstoned, and not part of the origin coordinate system.
func liveConstruction(construction, deleted, coordinateSystem bool) bool {
	return construction && !deleted && !coordinateSystem
}

// RefConsumedByDatum reports whether any live user datum references ref — the datum→datum half of a
// construction datum's consumer set (the sketch and part-feature halves are supplied by the host).
func (g *WorkGeometry) RefConsumedByDatum(ref WorkRef) bool {
	for _, f := range g.userFeatures() {
		if f.key == ref {
			continue
		}
		for _, r := range f.refs {
			if r == ref {
				return true
			}
		}
	}
	return false
}

// PruneConstructionOrphan tombstones a construction datum whose last consumer has just been removed
// (retainDependents=true — it has no dependents, or it would still have a consumer). It is a no-op,
// not an error, when ref no longer names a live user datum (already gone, or an origin datum).
func (g *WorkGeometry) PruneConstructionOrphan(ref WorkRef) {
	if d, err := g.userDatum(ref); err != nil || d.Deleted() || d.IsCoordinateSystemElement() {
		return
	}
	_, _ = g.DeleteWork(ref, true) // retain: a construction orphan has no dependents to cascade
}
