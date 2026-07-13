// SPDX-License-Identifier: GPL-2.0-only

// Package build maps a decoded Inventor .ipt onto the Oblikovati kernel/model: it
// facets the extracted B-rep into a solid body and assembles a part document.
package translate

import (
	"fmt"
	"math"
	"sort"

	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
)

// planeEps is the coplanarity tolerance in cm; SAB coordinates are exact so this is tight.
const planeEps = 1e-6

// SoupFromBrep facets the planar faces of an extracted B-rep into a triangle soup
// (each face fan-triangulated after ordering its coplanar vertices). Curved
// (cone/cylinder) faces are not yet tessellated — a B-rep containing them errors.
func SoupFromBrep(b ipt.Brep) (meshio.RawMesh, error) {
	if len(b.Cones) > 0 {
		return meshio.RawMesh{}, fmt.Errorf("build: %d curved face(s) not yet tessellated (planar solids only)", len(b.Cones))
	}
	var raw meshio.RawMesh
	bodyCenter := centroid(b.Points)
	for _, pl := range b.Planes {
		ring := coplanarRing(b.Points, pl)
		if len(ring) >= 3 {
			orientOutward(ring, bodyCenter)
			fanTriangulate(&raw, ring)
		}
	}
	if raw.TriangleCount() < 4 {
		return meshio.RawMesh{}, fmt.Errorf("build: only %d triangles — not enough to close a solid", raw.TriangleCount())
	}
	return raw, nil
}

// SoupFromMesh adapts Inventor's own stored display tessellation (ipt.GraphicsMesh) into a
// RawMesh: vertices carry over in cm, triangles keep their indices. Unlike SoupFromBrep this
// already has curved faces and holes tessellated, so it needs no per-face reconstruction —
// the mesh consumer welds duplicate boundary vertices between adjacent face patches.
func SoupFromMesh(mesh ipt.Mesh) meshio.RawMesh {
	raw := meshio.RawMesh{Verts: make([]m.Point3, len(mesh.Verts)), Tris: mesh.Tris}
	for i, v := range mesh.Verts {
		raw.Verts[i] = p3(v)
	}
	return raw
}

// BodyFromBrep facets the B-rep and welds it into a watertight solid via
// meshio.SolidOrSurface (which fixes inside-out winding). Used for in-memory validation.
func BodyFromBrep(b ipt.Brep, feat string) (*topo.Body, []string, error) {
	raw, err := SoupFromBrep(b)
	if err != nil {
		return nil, nil, err
	}
	return meshio.SolidOrSurface(raw, feat, meshio.DefaultWeldTolerance)
}

// coplanarRing returns the vertices lying on a plane, ordered into a boundary ring
// (CCW about the plane normal). Robust for a convex face; a holed face needs the true
// loop graph (future work).
func coplanarRing(pts [][3]float64, pl ipt.Plane) [][3]float64 {
	n := normalize(pl.Normal)
	var on [][3]float64
	for _, p := range pts {
		if math.Abs(dot(sub(p, pl.Origin), n)) < planeEps {
			on = append(on, p)
		}
	}
	return orderAroundNormal(on, n)
}

func orderAroundNormal(pts [][3]float64, n [3]float64) [][3]float64 {
	if len(pts) < 3 {
		return pts
	}
	c := centroid(pts)
	u := normalize(sub(pts[0], c))
	w := cross(n, u)
	sort.SliceStable(pts, func(i, j int) bool {
		return angle(pts[i], c, u, w) < angle(pts[j], c, u, w)
	})
	return pts
}

// orientOutward reverses a face ring in place if its winding normal points toward the
// body centre, so every face's triangles wind consistently outward (a clean, manifold
// mesh — SolidOrSurface would otherwise warn about mixed orientation).
func orientOutward(ring [][3]float64, bodyCenter [3]float64) {
	if len(ring) < 3 {
		return
	}
	n := cross(sub(ring[1], ring[0]), sub(ring[2], ring[0]))
	if dot(n, sub(centroid(ring), bodyCenter)) < 0 {
		for i, j := 0, len(ring)-1; i < j; i, j = i+1, j-1 {
			ring[i], ring[j] = ring[j], ring[i]
		}
	}
}

func fanTriangulate(raw *meshio.RawMesh, ring [][3]float64) {
	for i := 1; i+1 < len(ring); i++ {
		addTriangle(raw, ring[0], ring[i], ring[i+1])
	}
}

func addTriangle(raw *meshio.RawMesh, a, b, c [3]float64) {
	base := len(raw.Verts)
	raw.Verts = append(raw.Verts, p3(a), p3(b), p3(c))
	raw.Tris = append(raw.Tris, [3]int{base, base + 1, base + 2})
}

func p3(v [3]float64) m.Point3 { return m.P3(v[0], v[1], v[2]) }

// --- small float3 vector helpers (coordinates are in cm) ---

func sub(a, b [3]float64) [3]float64 { return [3]float64{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func dot(a, b [3]float64) float64    { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func cross(a, b [3]float64) [3]float64 {
	return [3]float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

func normalize(a [3]float64) [3]float64 {
	l := math.Sqrt(dot(a, a))
	if l == 0 {
		return a
	}
	return [3]float64{a[0] / l, a[1] / l, a[2] / l}
}

func centroid(pts [][3]float64) [3]float64 {
	var c [3]float64
	for _, p := range pts {
		c[0], c[1], c[2] = c[0]+p[0], c[1]+p[1], c[2]+p[2]
	}
	n := float64(len(pts))
	return [3]float64{c[0] / n, c[1] / n, c[2] / n}
}

func angle(p, c, u, w [3]float64) float64 {
	d := sub(p, c)
	return math.Atan2(dot(d, w), dot(d, u))
}
