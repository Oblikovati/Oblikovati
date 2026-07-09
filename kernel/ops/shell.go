// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// ShellDirection is which side of the original faces the wall grows onto — Inventor's
// ShellDirectionEnum.
type ShellDirection uint8

const (
	// ShellInside offsets the wall inward (the outer skin/dimensions are kept). Default.
	ShellInside ShellDirection = iota
	// ShellOutside grows the wall outward (outer dimensions increase by t); the original
	// surface becomes the inner cavity.
	ShellOutside
	// ShellBoth centres the wall on the original faces (t/2 in, t/2 out).
	ShellBoth
)

// Shell hollows a planar-faceted solid to wall thickness t, leaving the removed faces as
// openings — the inward shell (outer skin kept). It is [ShellDirected] with [ShellInside].
func Shell(solid *topo.Body, removedKeys [][]byte, t float64) (*topo.Body, error) {
	return ShellDirected(solid, removedKeys, t, ShellInside)
}

// ShellDirected hollows a planar-faceted solid to wall thickness t on the chosen side of the
// original faces, leaving the removed faces as openings. Every KEPT face is offset (a REMOVED
// face stays in place, so the wall stays flush with it and the difference opens it — the
// coplanar B-rep rule); the wall is then the region between the original solid and the offset
// solid(s):
//   - Inside:  solid − (kept faces offset inward t)               — outer skin kept.
//   - Outside: (kept faces offset outward t) − solid              — outer dimensions grow by t.
//   - Both:    (kept faces offset outward t/2) − (offset inward t/2) — wall centred on the faces.
func ShellDirected(solid *topo.Body, removedKeys [][]byte, t float64, dir ShellDirection) (*topo.Body, error) {
	if t <= 0 {
		return nil, fmt.Errorf("shell: thickness %g must be > 0", t)
	}
	removed, err := resolveFaceSet(solid, removedKeys)
	if err != nil {
		return nil, err
	}
	switch dir {
	case ShellInside:
		return Boolean(Cut, solid, offsetShellSolid(solid, removed, -t))
	case ShellOutside:
		return Boolean(Cut, offsetShellSolid(solid, removed, t), solid)
	case ShellBoth:
		return Boolean(Cut, offsetShellSolid(solid, removed, t/2), offsetShellSolid(solid, removed, -t/2))
	default:
		return nil, fmt.Errorf("shell: unknown direction %d", dir)
	}
}

// offsetShellSolid rebuilds the solid with every kept face's plane moved by d along its normal
// (d<0 inward, d>0 outward); removed faces stay in place so the shell opens flush there.
func offsetShellSolid(solid *topo.Body, removed map[uint64]bool, d float64) *topo.Body {
	return rebuildWithPlanes(solid, "shell-offset", false, func(f *topo.Face) geom.Plane {
		return shellFacePlane(f, removed, d)
	})
}

// shellFacePlane returns a kept face's plane offset by d along its normal (origin moved along
// +normal·d), or the unchanged plane for a removed face (so the shell stays flush there).
func shellFacePlane(f *topo.Face, removed map[uint64]bool, d float64) geom.Plane {
	pl := f.Geometry().(geom.Plane)
	if removed[f.ID()] {
		return pl
	}
	moved, _ := geom.NewPlaneFromAxes(pl.Origin.TranslateBy(pl.Normal().Scale(d)), pl.UAxis.AsVector(), pl.VAxis.AsVector())
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
