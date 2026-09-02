// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"oblikovati.org/kernel/mesh"
	"oblikovati.org/math"
)

// Vertex/polygon classification for the BSP split (csg.js convention).
const (
	coplanar = 0
	frontOf  = 1
	backOf   = 2
	spanning = 3
)

// splitTri classifies triangle t against the plane (n·p = w) and appends its pieces to
// the four buckets, fan-triangulating any split so every bucket holds triangles. A
// triangle spanning the plane is cut along it into a front and a back polygon.
func splitTri(n math.Vector3, w float64, t mesh.Tri, coFront, coBack, fr, bk *[]mesh.Tri, planeTol float64) {
	pts := t.Points()
	types, polyType := classifyVertices(n, w, pts, planeTol)
	switch polyType {
	case coplanar:
		if n.Dot(t.N) > 0 {
			*coFront = append(*coFront, t)
		} else {
			*coBack = append(*coBack, t)
		}
	case frontOf:
		*fr = append(*fr, t)
	case backOf:
		*bk = append(*bk, t)
	default: // spanning
		f, b := splitSpanning(n, w, pts, types)
		appendFan(fr, f)
		appendFan(bk, b)
	}
}

// classifyVertices labels each triangle vertex front/back/coplanar against the plane and
// returns the per-vertex labels plus their OR (the polygon's overall relationship).
func classifyVertices(n math.Vector3, w float64, pts [3]math.Point3, planeTol float64) ([]int, int) {
	types := make([]int, 3)
	polyType := 0
	for i, p := range pts {
		d := n.Dot(p.AsVector()) - w
		ty := coplanar
		if d < -planeTol {
			ty = backOf
		} else if d > planeTol {
			ty = frontOf
		}
		types[i] = ty
		polyType |= ty
	}
	return types, polyType
}

// splitSpanning cuts a triangle that crosses the plane into its front and back vertex
// loops, inserting the edge–plane intersection points (csg.js polygon split).
func splitSpanning(n math.Vector3, w float64, pts [3]math.Point3, types []int) (f, b []math.Point3) {
	for i := range 3 {
		j := (i + 1) % 3
		ti, tj := types[i], types[j]
		if ti != backOf {
			f = append(f, pts[i])
		}
		if ti != frontOf {
			b = append(b, pts[i])
		}
		if (ti | tj) == spanning {
			vi, vj := pts[i].AsVector(), pts[j].AsVector()
			tParam := (w - n.Dot(vi)) / n.Dot(vj.Sub(vi))
			mid := pts[i].TranslateBy(pts[i].VectorTo(pts[j]).Scale(tParam))
			f = append(f, mid)
			b = append(b, mid)
		}
	}
	return f, b
}

// appendFan fan-triangulates a convex vertex loop (3 or 4 points) into dst, dropping
// degenerate triangles.
func appendFan(dst *[]mesh.Tri, loop []math.Point3) {
	for i := 1; i+1 < len(loop); i++ {
		if t, ok := mesh.NewTri(loop[0], loop[i], loop[i+1]); ok {
			*dst = append(*dst, t)
		}
	}
}
