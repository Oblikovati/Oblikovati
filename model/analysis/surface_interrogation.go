// SPDX-License-Identifier: GPL-2.0-only

package analysis

import "oblikovati.org/math"

// Surface interrogation (M36-F12) — the reflection/highlight/isophote line families a styling
// reviewer judges a Class-A surface by. Each family is the iso-contour of a per-vertex scalar
// field over the tessellated surface, so one marching-triangles contourer serves all three; only
// the scalar differs. The classic property: a curvature discontinuity (a G1-only seam) bends these
// lines sharply, while a G2 seam leaves them smooth — the visual continuity test.

// Segment3 is one straight piece of an interrogation line on the surface.
type Segment3 struct{ A, B math.Point3 }

// SurfaceSamples is a tessellated surface: per-vertex positions and (unit) normals, plus the
// triangle index triples referencing them. It is what the renderer already has for any face, so
// interrogation runs on the exact mesh the user sees.
type SurfaceSamples struct {
	Positions []math.Point3
	Normals   []math.Vector3
	Triangles [][3]int
}

// Isophotes returns iso-contours of N·L (the cosine between the surface normal and the light
// direction) at count evenly spaced levels in (−1, 1) — the classic G1-vs-G2 discriminator.
func Isophotes(m SurfaceSamples, light math.Vector3, count int) []Segment3 {
	l := normalize(light)
	return contour(m, scalarField(m, func(_ math.Point3, n math.Vector3) float64 {
		return float64(n.Dot(l))
	}), count)
}

// ReflectionLines returns the reflection lines of a parallel-stripe environment seen from eye:
// the eye→point direction is reflected about the surface normal and projected onto a stripe axis,
// and the iso-contours of that projection are the reflected stripe boundaries — the showroom test
// (a kink or break reveals a discontinuity).
func ReflectionLines(m SurfaceSamples, eye math.Point3, stripeAxis math.Vector3, count int) []Segment3 {
	g := normalize(stripeAxis)
	return contour(m, scalarField(m, func(p math.Point3, n math.Vector3) float64 {
		return float64(reflect(normalize(p.VectorTo(eye)), n).Dot(g))
	}), count)
}

// HighlightLines returns the specular-highlight contours: the Blinn half-vector H between the light
// and the eye direction, contoured by N·H — the bright bands a point light paints, whose smoothness
// tracks surface fairness.
func HighlightLines(m SurfaceSamples, eye math.Point3, light math.Vector3, count int) []Segment3 {
	l := normalize(light)
	return contour(m, scalarField(m, func(p math.Point3, n math.Vector3) float64 {
		h := normalize(normalize(p.VectorTo(eye)).Add(l))
		return float64(n.Dot(h))
	}), count)
}

// ZebraTriangleBands returns, per triangle, whether it falls in a DARK zebra band of a distant
// parallel-stripe environment: the constant view direction is reflected about the surface normal and
// projected on the stripe axis, and that projection (a unit-vector dot, in [-1,1]) is quantized into
// count equal bands. A distant (directional) view, not a per-point eye, keeps the stripe frequency
// uniform across the form instead of compressing at grazing angles. Filling each triangle solid by
// its band parity renders the classic zebra map; the band EDGES reveal continuity — a G2 seam keeps
// them flowing, a G1-only seam steps them. count is the number of bands (min 1).
func ZebraTriangleBands(m SurfaceSamples, viewDir, stripeAxis math.Vector3, count int) []bool {
	v, g := normalize(viewDir), normalize(stripeAxis)
	vert := scalarField(m, func(_ math.Point3, n math.Vector3) float64 {
		return float64(reflect(v, n).Dot(g))
	})
	if count < 1 {
		count = 1
	}
	bands := make([]bool, len(m.Triangles))
	for i, tri := range m.Triangles {
		c := (vert[tri[0]] + vert[tri[1]] + vert[tri[2]]) / 3 // centroid projection in [-1,1]
		b := int((c + 1) / 2 * float64(count))                // [-1,1] → [0,count)
		if b >= count {
			b = count - 1
		} else if b < 0 {
			b = 0
		}
		bands[i] = b%2 == 1
	}
	return bands
}

// scalarField evaluates f at every vertex (position + unit normal) of the mesh.
func scalarField(m SurfaceSamples, f func(p math.Point3, n math.Vector3) float64) []float64 {
	out := make([]float64, len(m.Positions))
	for i := range m.Positions {
		out[i] = f(m.Positions[i], normalize(m.Normals[i]))
	}
	return out
}

// contour returns the iso-line segments of a per-vertex scalar field at count levels auto-fitted to
// the field's actual range, by marching each triangle: a level that separates a triangle's vertices
// crosses exactly two edges, and the two crossing points form one segment. A constant field (e.g. a
// flat patch) yields no contours.
func contour(m SurfaceSamples, field []float64, count int) []Segment3 {
	levels := interiorLevels(field, count)
	var out []Segment3
	for _, tri := range m.Triangles {
		a, b, c := tri[0], tri[1], tri[2]
		for _, lv := range levels {
			if seg, ok := triangleCrossing(m.Positions, field, a, b, c, lv); ok {
				out = append(out, seg)
			}
		}
	}
	return out
}

// triangleCrossing returns the iso-line segment of `level` through triangle (a,b,c), or ok=false
// when the level does not cut exactly two of its edges.
func triangleCrossing(pos []math.Point3, field []float64, a, b, c int, level float64) (Segment3, bool) {
	var pts [3]math.Point3
	n := 0
	for _, e := range [3][2]int{{a, b}, {b, c}, {c, a}} {
		if p, ok := edgeCrossing(pos[e[0]], pos[e[1]], field[e[0]], field[e[1]], level); ok && n < 3 {
			pts[n] = p
			n++
		}
	}
	if n != 2 {
		return Segment3{}, false
	}
	return Segment3{A: pts[0], B: pts[1]}, true
}

// edgeCrossing returns where the field equals level along the edge p0→p1 (linear interpolation),
// or ok=false when the endpoints do not straddle the level.
func edgeCrossing(p0, p1 math.Point3, f0, f1, level float64) (math.Point3, bool) {
	d0, d1 := f0-level, f1-level
	if (d0 > 0) == (d1 > 0) || d0 == d1 {
		return math.Point3{}, false
	}
	t := d0 / (f0 - f1)
	return p0.TranslateBy(p0.VectorTo(p1).Scale(math.Scalar(t))), true
}

// interiorLevels returns count contour levels evenly spaced strictly inside the field's actual
// [min, max] range, so a gentle surface still shows count lines and a flat (constant) field shows
// none. count < 1 is treated as 1.
func interiorLevels(field []float64, count int) []float64 {
	if count < 1 {
		count = 1
	}
	lo, hi := fieldRange(field)
	if hi-lo < 1e-12 {
		return nil // constant field: no contours
	}
	out := make([]float64, count)
	for i := 0; i < count; i++ {
		out[i] = lo + (hi-lo)*float64(i+1)/float64(count+1)
	}
	return out
}

// fieldRange returns the min and max of a scalar field (0,0 for an empty field).
func fieldRange(field []float64) (lo, hi float64) {
	if len(field) == 0 {
		return 0, 0
	}
	lo, hi = field[0], field[0]
	for _, v := range field[1:] {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	return lo, hi
}

// normalize returns v scaled to unit length, or v unchanged when it is (near) zero.
func normalize(v math.Vector3) math.Vector3 {
	l := float64(v.Length())
	if l < 1e-12 {
		return v
	}
	return v.Scale(math.Scalar(1 / l))
}

// reflect mirrors direction d about the unit normal n (R = d − 2(d·n)n).
func reflect(d, n math.Vector3) math.Vector3 {
	return d.Sub(n.Scale(2 * d.Dot(n)))
}
