// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Shell hollows a planar-faceted solid to wall thickness t, leaving the removed faces as
// openings. It builds the inner cavity solid by offsetting every KEPT face inward by t
// (via [rebuildWithPlanes]) while leaving the REMOVED faces in place — so the cavity stays
// flush with them and the difference opens them (the coplanar B-rep rule) — then returns
// solid − cavity. Inward shell only; outward/both-sides are a follow-up.
func Shell(solid *topo.Body, removedKeys [][]byte, t float64) (*topo.Body, error) {
	if t <= 0 {
		return nil, fmt.Errorf("shell: thickness %g must be > 0", t)
	}
	removed, err := resolveFaceSet(solid, removedKeys)
	if err != nil {
		return nil, err
	}
	cavity := rebuildWithPlanes(solid, "shell-cavity", func(f *topo.Face) geom.Plane {
		return shellFacePlane(f, removed, t)
	})
	return Boolean(Cut, solid, cavity)
}

// shellFacePlane returns a kept face's plane offset inward by t (origin moved along
// −normal), or the unchanged plane for a removed face (so the cavity stays flush there).
func shellFacePlane(f *topo.Face, removed map[uint64]bool, t float64) geom.Plane {
	pl := f.Geometry().(geom.Plane)
	if removed[f.ID()] {
		return pl
	}
	moved, _ := geom.NewPlaneFromAxes(pl.Origin.TranslateBy(pl.Normal().Scale(-t)), pl.UAxis.AsVector(), pl.VAxis.AsVector())
	return moved
}

// resolveFaceSet turns face reference keys into the set of face IDs they name, erroring if a
// key no longer resolves (the feature must go Sick honestly).
func resolveFaceSet(solid *topo.Body, keys [][]byte) (map[uint64]bool, error) {
	set := make(map[uint64]bool, len(keys))
	for _, k := range keys {
		// ADR-0043 resolution guard: a key must bind to exactly one face. >1 is a topological-
		// naming collision, surfaced as an honest error rather than a silent first-match.
		match := solid.FacesByKey(k)
		switch len(match) {
		case 1:
			set[match[0].ID()] = true
		case 0:
			return nil, fmt.Errorf("face reference lost: %x", k)
		default:
			return nil, fmt.Errorf("face reference %x is ambiguous — it matches %d faces (a topological-naming collision)", k, len(match))
		}
	}
	return set, nil
}
