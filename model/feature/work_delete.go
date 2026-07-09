// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// User work features are referenced by their position in their collection ("plane/3", "axis/1")
// — a scheme that stays stable only because features are never removed from the slice. So a
// delete is a TOMBSTONE, not a slice removal: the slot is kept (surviving datums keep their
// positional refs), the datum is flagged deleted, and the resolvers stop resolving it — so it
// vanishes from every listing and any surviving dependent that still names it goes sick. Mirrors
// Inventor's WorkFeature.Delete(RetainDependents) (#1855).

// deletableDatum is the shared shape of a tombstonable work feature (plane / axis / point).
type deletableDatum interface {
	IsCoordinateSystemElement() bool
	Deleted() bool
	markDeleted()
}

// DeleteWork tombstones the user datum work feature named by ref. With retainDependents=false
// (Inventor's default) every user work feature that references it, directly or transitively, is
// tombstoned with it; with retainDependents=true only the named datum is removed and its
// dependents go unhealthy on the next recompute (their reference no longer resolves). It returns
// the refs of every datum removed, in deletion order (the target first). It errors on an unknown
// ref, an origin/coordinate-system datum, or an already-deleted datum; the caller recomputes so
// retained dependents re-derive.
func (g *WorkGeometry) DeleteWork(ref WorkRef, retainDependents bool) ([]WorkRef, error) {
	target, err := g.userDatum(ref)
	if err != nil {
		return nil, err
	}
	if target.IsCoordinateSystemElement() {
		return nil, fmt.Errorf("work geometry: %q is an origin/coordinate-system datum and cannot be deleted", ref)
	}
	if target.Deleted() {
		return nil, fmt.Errorf("work geometry: work feature %q is already deleted", ref)
	}
	doomed := []WorkRef{ref}
	if !retainDependents {
		doomed = g.dependencyClosure(ref)
	}
	for _, r := range doomed {
		d, err := g.userDatum(r)
		if err != nil {
			return nil, err
		}
		d.markDeleted()
	}
	return doomed, nil
}

// userDatum resolves a "plane/N" / "axis/N" / "point/N" reference to its live datum. Only user
// features are addressable this way; an origin ref (whose key is "origin/…") or any other string
// is not a deletable user datum.
func (g *WorkGeometry) userDatum(ref WorkRef) (deletableDatum, error) {
	if i, ok := userIndex(ref, "plane"); ok && i >= 0 && i < g.planes.Count() {
		return g.planes.Item(i), nil
	}
	if i, ok := userIndex(ref, "axis"); ok && i >= 0 && i < g.axes.Count() {
		return g.axes.Item(i), nil
	}
	if i, ok := userIndex(ref, "point"); ok && i >= 0 && i < g.points.Count() {
		return g.points.Item(i), nil
	}
	return nil, fmt.Errorf("work geometry: %q is not a user work plane, axis, or point", ref)
}

// datumRefs is one user datum's stable key and the references its definition is built on.
type datumRefs struct {
	key  WorkRef
	refs []WorkRef
}

// userFeatures lists every live (non-origin, non-deleted) user datum with its definition refs —
// the graph dependencyClosure walks to find what a deletion cascades to.
func (g *WorkGeometry) userFeatures() []datumRefs {
	var out []datumRefs
	collect := func(key WorkRef, cs, del bool, refs []WorkRef) {
		if cs || del {
			return
		}
		out = append(out, datumRefs{key: key, refs: refs})
	}
	for i := 0; i < g.planes.Count(); i++ {
		p := g.planes.Item(i)
		collect(p.key, p.coordinateSystem, p.deleted, p.def.refs())
	}
	for i := 0; i < g.axes.Count(); i++ {
		a := g.axes.Item(i)
		collect(a.key, a.coordinateSystem, a.deleted, a.def.refs())
	}
	for i := 0; i < g.points.Count(); i++ {
		p := g.points.Item(i)
		collect(p.key, p.coordinateSystem, p.deleted, p.def.refs())
	}
	return out
}

// dependencyClosure returns root plus every live user datum that references it transitively, in
// discovery order — the set a retainDependents=false delete removes together.
func (g *WorkGeometry) dependencyClosure(root WorkRef) []WorkRef {
	doomed := map[WorkRef]bool{root: true}
	order := []WorkRef{root}
	feats := g.userFeatures()
	for changed := true; changed; {
		changed = false
		for _, f := range feats {
			if doomed[f.key] {
				continue
			}
			for _, r := range f.refs {
				if doomed[r] {
					doomed[f.key] = true
					order = append(order, f.key)
					changed = true
					break
				}
			}
		}
	}
	return order
}
