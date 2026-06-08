// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati/math"
)

// DefaultCreaseAngle is the dihedral threshold for SmoothShadeNormals: facets meeting at a
// shallower angle are treated as one smooth surface; sharper meetings stay a crisp edge. 35°
// keeps box corners and a loft's section edges sharp while smoothing a lofted/swept twist that
// is faceted into many near-coplanar quads.
func DefaultCreaseAngle() float64 { return 35 * stdmath.Pi / 180 }

// SmoothShadeNormals returns per-vertex display normals for a body mesh: at each spatial vertex
// position, the normals of incident triangles are averaged across facets whose normals are
// within creaseAngle, so a faceted-but-smooth surface (a loft/sweep skin, built as many planar
// facets) shades smoothly while genuine sharp edges keep split normals. Positions and indices
// are unchanged and the input mesh is not mutated — this is a DISPLAY-only refinement; mass
// properties stay on the raw per-face normals (which point reliably outward per facet).
//
// It operates on the existing outward normals (TessellateBody already orients them), grouping
// coincident vertices by position: two facets that share an edge produce coincident boundary
// vertices, so averaging within a position group blends exactly the facets that meet there.
func SmoothShadeNormals(m *Mesh, creaseAngle float64) []math.Vector3 {
	cosThresh := stdmath.Cos(creaseAngle)
	groups := make(map[[3]int64][]int, len(m.Positions))
	for i, p := range m.Positions {
		k := weldKey(p)
		groups[k] = append(groups[k], i)
	}
	out := make([]math.Vector3, len(m.Normals))
	for i := range m.Normals {
		out[i] = smoothedNormal(m, i, groups[weldKey(m.Positions[i])], cosThresh)
	}
	return out
}

// smoothedNormal averages vertex i's normal with the coincident vertices whose normals are
// within the crease angle (dot ≥ cosThresh), falling back to the original when nothing matches.
func smoothedNormal(m *Mesh, i int, group []int, cosThresh float64) math.Vector3 {
	ni := unitOr(m.Normals[i])
	var sum math.Vector3
	for _, j := range group {
		nj := unitOr(m.Normals[j])
		if float64(ni.Dot(nj)) >= cosThresh {
			sum = sum.Add(nj)
		}
	}
	if sum.LengthSquared() < math.DefaultTolerance {
		return ni
	}
	return unitOr(sum)
}

// unitOr normalizes v, returning it unchanged when degenerate (zero length).
func unitOr(v math.Vector3) math.Vector3 {
	l := v.Length()
	if l < math.Scalar(stdmath.Sqrt(float64(math.DefaultTolerance))) {
		return v
	}
	return v.Scale(1 / l)
}

// weldKey quantizes a position to 1e-5 (database units) so coincident vertices from adjacent
// faces — which share exact discretized edge points — land in the same group.
func weldKey(p math.Point3) [3]int64 {
	const q = 1e5
	return [3]int64{
		int64(stdmath.Round(float64(p.X) * q)),
		int64(stdmath.Round(float64(p.Y) * q)),
		int64(stdmath.Round(float64(p.Z) * q)),
	}
}
