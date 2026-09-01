// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/retopo"
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
	return ShellVaried(solid, removedKeys, t, dir, nil)
}

// ShellFaceThickness gives one RETAINED face its own wall thickness, overriding the shell's
// default — Inventor's SetFaceThickness / unique-thickness face set (#1864).
type ShellFaceThickness struct {
	FaceKey   []byte
	Thickness float64
}

// ShellVaried is [ShellDirected] with per-face wall thicknesses: a thickened boss wall or a thin
// window in an otherwise uniform shell. Each override offsets only its own face, so the walls meet
// at the ordinary mitre the offset planes already produce — there is no separate blend step.
// Naming a REMOVED face is an error: an opening has no wall to be thick.
func ShellVaried(solid *topo.Body, removedKeys [][]byte, t float64, dir ShellDirection,
	overrides []ShellFaceThickness) (*topo.Body, error) {
	if t <= 0 {
		return nil, fmt.Errorf("shell: thickness %g must be > 0", t)
	}
	removed, err := retopo.ResolveFaceSet(solid, removedKeys)
	if err != nil {
		return nil, err
	}
	wall, err := resolveWallThickness(solid, overrides, removed, t)
	if err != nil {
		return nil, err
	}
	return shellByDirection(solid, removed, wall, dir)
}

// shellByDirection cuts the wall out of the region between the solid and its offset copies, on
// whichever side of the original faces the direction names.
func shellByDirection(solid *topo.Body, removed map[uint64]bool, wall faceThickness,
	dir ShellDirection) (*topo.Body, error) {
	switch dir {
	case ShellInside:
		return Boolean(Cut, solid, offsetShellSolid(solid, removed, wall.scaled(-1)))
	case ShellOutside:
		return Boolean(Cut, offsetShellSolid(solid, removed, wall.scaled(1)), solid)
	case ShellBoth:
		return Boolean(Cut, offsetShellSolid(solid, removed, wall.scaled(0.5)),
			offsetShellSolid(solid, removed, wall.scaled(-0.5)))
	default:
		return nil, fmt.Errorf("shell: unknown direction %d", dir)
	}
}

// faceThickness is the wall thickness to build at each face: the shell default, with per-face
// overrides applied.
type faceThickness struct {
	dflt  float64
	perID map[uint64]float64
}

// at returns the wall thickness for face f.
func (w faceThickness) at(f *topo.Face) float64 {
	if t, ok := w.perID[f.ID()]; ok {
		return t
	}
	return w.dflt
}

// scaled returns each face's offset distance for one side of the wall (s<0 inward).
func (w faceThickness) scaled(s float64) func(*topo.Face) float64 {
	return func(f *topo.Face) float64 { return s * w.at(f) }
}

// resolveWallThickness binds the overrides to face IDs, rejecting a lost key (via retopo.ResolveFaceSet's
// rules), a removed face, and a non-positive thickness — each of which would otherwise produce a
// quietly wrong wall rather than a sick feature.
func resolveWallThickness(solid *topo.Body, overrides []ShellFaceThickness, removed map[uint64]bool,
	t float64) (faceThickness, error) {
	wall := faceThickness{dflt: t, perID: make(map[uint64]float64, len(overrides))}
	for _, o := range overrides {
		if o.Thickness <= 0 {
			return wall, fmt.Errorf("shell: face thickness %g must be > 0 (face %x)", o.Thickness, o.FaceKey)
		}
		ids, err := retopo.ResolveFaceSet(solid, [][]byte{o.FaceKey})
		if err != nil {
			return wall, fmt.Errorf("shell: face thickness: %w", err)
		}
		for id := range ids {
			if removed[id] {
				return wall, fmt.Errorf("shell: face %x is removed, so it is an opening and has no "+
					"wall; give a thickness for a RETAINED face", o.FaceKey)
			}
			wall.perID[id] = o.Thickness
		}
	}
	return wall, nil
}

// offsetShellSolid rebuilds the solid with every kept face's plane moved by dist(f) along its
// normal (negative inward); removed faces stay in place so the shell opens flush there.
func offsetShellSolid(solid *topo.Body, removed map[uint64]bool, dist func(*topo.Face) float64) *topo.Body {
	return retopo.RebuildWithPlanes(solid, "shell-offset", false, func(f *topo.Face) geom.Plane {
		return shellFacePlane(f, removed, dist(f))
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
