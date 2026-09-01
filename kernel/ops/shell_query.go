// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Shell and body point queries (M07-F06/F07, Oblikovati/Oblikovati#629/#630):
// signed shell volume (the void/solid classifier), and inside/on/outside
// containment for shells and whole bodies.

// PointContainment is the inside/on/outside verdict of a point query. The
// api/types Containment enum is the wire spelling; this is the kernel value.
type PointContainment uint8

const (
	ContainOutside PointContainment = iota
	ContainInside
	ContainOn
)

// String returns a stable name for diagnostics.
func (c PointContainment) String() string {
	switch c {
	case ContainInside:
		return "inside"
	case ContainOn:
		return "on"
	default:
		return "outside"
	}
}

// shellMesh merges the shell's face tessellations (each face oriented to its
// material-outward shading normals, Reversed honored).
func shellMesh(s *topo.Shell, q Quality) *Mesh {
	mesh := &Mesh{}
	for _, f := range s.Faces() {
		tessellate.MergeMesh(mesh, tessellate.TessellateFace(f, q))
	}
	return mesh
}

// ShellSignedVolume returns the divergence-theorem volume of the region the
// shell bounds, signed by its material orientation: positive for an outer
// (material-enclosing) shell, NEGATIVE for an inner void shell, whose
// material-outward face normals point into the cavity.
//
// It integrates the shell's ANALYTIC faces (AnalyticShellVolume, M48/C3 #3482) — the void-ness of
// a shell is a topological decision and must not be read off a merged face mesh, whose chord
// deficit shrinks a cavity skin's magnitude and, on a thin void, can reach zero. q parameterises
// only the fallback for a shell whose faces the analytic path declines.
//
// Example: if ops.ShellSignedVolume(sh, ops.DefaultQuality()) < 0 { /* cavity skin */ }
func ShellSignedVolume(s *topo.Shell, q Quality) float64 {
	if v, ok := AnalyticShellVolume(s); ok {
		return v
	}
	mesh := shellMesh(s, q)
	vol := 0.0
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		ia, ib, ic := mesh.Indices[t], mesh.Indices[t+1], mesh.Indices[t+2]
		a, b, c := mesh.Positions[ia], mesh.Positions[ib], mesh.Positions[ic]
		if outwardRef(mesh, ia, ib, ic).Dot(a.VectorTo(b).Cross(a.VectorTo(c))) < 0 {
			b, c = c, b // align geometric winding to the outward shading normal
		}
		vol += float64(a.AsVector().Dot(b.AsVector().Cross(c.AsVector()))) / 6
	}
	return vol
}

// ShellContainment classifies p against the region the shell bounds, analytically (M48/C3 #3428):
// ON within onTol of a trimmed face, else INSIDE by the analytic ray-parity classifier over the
// shell's own faces (brep.ClassifyShellPoint). No tessellation is read — the ground rule that a
// topological decision never consumes a mesh. The Quality argument is retained for call-site
// compatibility but no longer used now that the oracle is analytic.
func ShellContainment(s *topo.Shell, p math.Point3, _ Quality, onTol float64) PointContainment {
	return toPointContainment(brep.ClassifyShellPoint(s, p, onTol))
}

// BodyContainment classifies p against the solid body b, analytically (M48/C3 #3429): ON within
// onTol of a trimmed face, else INSIDE by the analytic point-in-solid classifier
// (brep.ClassifyPointTol, which dispatches an all-planar body to the generalized winding number and
// a curved body to the nearest-crossing ray classifier). A point inside a cavity sums to winding 0
// and so reads as OUTSIDE the material — the same verdict the retired mesh oracle gave, because the
// void skin's contribution cancels the outer shell's. No tessellation is read; the Quality argument
// is retained for call-site compatibility only.
func BodyContainment(b *topo.Body, p math.Point3, _ Quality, onTol float64) PointContainment {
	return toPointContainment(brep.ClassifyPointTol(b, p, onTol))
}

// toPointContainment maps the brep tri-state (Outside/OnSurface/Inside) onto the ops PointContainment
// enum. Their ordinals differ (brep: Outside=0,OnSurface=1,Inside=2; ops: Outside=0,Inside=1,On=2),
// so the two must be translated case by case and NEVER cast across.
func toPointContainment(c brep.Containment) PointContainment {
	switch c {
	case brep.Inside:
		return ContainInside
	case brep.OnSurface:
		return ContainOn
	default:
		return ContainOutside
	}
}
