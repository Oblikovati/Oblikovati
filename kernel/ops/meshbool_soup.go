// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Adapter from the kernel topology to the exact mesh-arrangement boolean core
// (kernel/meshbool, ADR-0052). bodyToSoup is the IN half: it tessellates a body to
// a triangle soup meshbool.Boolean can consume. Because the tessellator welds
// vertices along shared edges, the soup is a closed, consistently outward-oriented
// mesh — exactly the operand form the boolean expects. Curved faces are faceted at
// the requested quality (the boolean is a faceted operation, like the planar B-rep
// boolean it will stand beside); planar faces triangulate exactly.

// bodyToSoup returns body b, tessellated at quality q, as an exact triangle soup.
func bodyToSoup(b *topo.Body, q Quality) [][3]meshbool.Point {
	mesh, _ := TessellateBody(b, q)
	return soupFromMesh(mesh)
}

// soupFromMesh converts an indexed tessellation Mesh into a meshbool triangle
// soup. It first WELDS near-coincident positions: TessellateBody meshes each face
// independently, so a vertex shared by two faces (e.g. a cylinder's cap/wall rim)
// gets a per-face copy that differs by ~1 ulp — invisible in display but a crack to
// the exact boolean, which needs a watertight input. Welding those copies to a
// single position corrects tessellation float-noise; it does not weaken the
// boolean, which stays exact on the welded mesh. Triangles that collapse to a
// degenerate (a sub-tolerance sliver) are dropped.
func soupFromMesh(m *Mesh) [][3]meshbool.Point {
	pos := weldPositions(m.Positions)
	soup := make([][3]meshbool.Point, 0, m.TriangleCount())
	for t := 0; t < m.TriangleCount(); t++ {
		i, j, k := m.Indices[3*t], m.Indices[3*t+1], m.Indices[3*t+2]
		tri := [3]meshbool.Point{meshbool.FromPoint3(pos[i]), meshbool.FromPoint3(pos[j]), meshbool.FromPoint3(pos[k])}
		if tri[0].Equal(tri[1]) || tri[1].Equal(tri[2]) || tri[2].Equal(tri[0]) {
			continue // welded to a degenerate sliver
		}
		soup = append(soup, tri)
	}
	return soup
}

// weldPositions snaps positions within a size-relative tolerance to a shared
// representative (a spatial-hash cluster), so per-face copies of a shared vertex
// become identical. The tolerance is far above tessellation float-noise (~1 ulp)
// and far below facet spacing, so distinct facet vertices never merge.
func weldPositions(pos []math.Point3) []math.Point3 {
	tol := weldTolerance(pos)
	cellOf := func(p math.Point3) [3]int64 {
		return [3]int64{int64(p.X / tol), int64(p.Y / tol), int64(p.Z / tol)}
	}
	buckets := make(map[[3]int64][]int)
	var reps []math.Point3
	out := make([]math.Point3, len(pos))
	for i, p := range pos {
		if r := findRep(p, cellOf(p), buckets, reps, tol); r >= 0 {
			out[i] = reps[r]
			continue
		}
		c := cellOf(p)
		buckets[c] = append(buckets[c], len(reps))
		reps = append(reps, p)
		out[i] = p
	}
	return out
}

// findRep returns the index of an existing representative within tol of p (scanning
// the 27 cells around p's), or -1.
func findRep(p math.Point3, c [3]int64, buckets map[[3]int64][]int, reps []math.Point3, tol float64) int {
	for dx := int64(-1); dx <= 1; dx++ {
		for dy := int64(-1); dy <= 1; dy++ {
			for dz := int64(-1); dz <= 1; dz++ {
				for _, ri := range buckets[[3]int64{c[0] + dx, c[1] + dy, c[2] + dz}] {
					if dist2(reps[ri], p) <= tol*tol {
						return ri
					}
				}
			}
		}
	}
	return -1
}

// weldTolerance returns a merge tolerance scaled to the mesh's bounding box, so it
// tracks coordinate magnitude (float noise grows with magnitude) yet stays orders
// of magnitude below facet spacing.
func weldTolerance(pos []math.Point3) float64 {
	if len(pos) == 0 {
		return 1e-9
	}
	lo, hi := pos[0], pos[0]
	for _, p := range pos {
		lo = math.P3(stdmath.Min(lo.X, p.X), stdmath.Min(lo.Y, p.Y), stdmath.Min(lo.Z, p.Z))
		hi = math.P3(stdmath.Max(hi.X, p.X), stdmath.Max(hi.Y, p.Y), stdmath.Max(hi.Z, p.Z))
	}
	diag := stdmath.Sqrt(dist2(lo, hi))
	if diag == 0 {
		return 1e-9
	}
	return 1e-9 * diag
}

func dist2(a, b math.Point3) float64 {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
	return dx*dx + dy*dy + dz*dz
}
