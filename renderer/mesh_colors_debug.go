// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/scene"
)

// BuildDrawListMeshColors builds a SHADED draw list where each mesh primitive is painted a distinct,
// index-derived color — the "mesh debug colors" render. It lets a viewer map a rendered region back to
// the mesh data: a defect you SEE (a back-face, a crack) to the primitive you diagnose in code.
//   - perTriangle=false: each B-rep FACE a color = meshDebugColor(globalFaceIndex). Face N reproducible.
//   - perTriangle=true:  each TRIANGLE a color = meshDebugColor(globalTriIndex). Triangles never share a
//     vertex (so the color is per-triangle), which finds an individual triangle in the mesh.
//
// The global index runs across all bodies in face/triangle order, so a primitive's color is stable.
func BuildDrawListMeshColors(bodies []*topo.Body, cam scene.Camera, q ops.Quality, perTriangle bool) DrawList {
	shading := PassSetFor(ShadedWithEdges).Faces
	var items []DrawItem
	index := 0
	for _, b := range bodies {
		if !visible(cam, b.RangeBox()) {
			index += primitiveCount(b, q, perTriangle) // keep the global index stable under cull
			continue
		}
		if it, ok := meshColorItem(b, q, perTriangle, shading, &index); ok {
			items = append(items, it)
		}
	}
	return DrawList{Items: items}
}

// meshColorItem accumulates one body's per-face/per-triangle colored geometry into a single draw item,
// advancing the running primitive index.
func meshColorItem(b *topo.Body, q ops.Quality, perTriangle bool, shading Shading, index *int) (DrawItem, bool) {
	var pos []math.Point3
	var nrm []math.Vector3
	var cols [][4]float32
	var idx []int
	for _, f := range b.Faces() {
		m := tessellate.TessellateFace(f, q)
		if perTriangle {
			addTriangleColored(&pos, &nrm, &cols, &idx, m, index)
		} else {
			addFaceColored(&pos, &nrm, &cols, &idx, m, meshDebugColor(*index))
			*index++
		}
	}
	if len(idx) == 0 {
		return DrawItem{}, false
	}
	return DrawItem{
		Primitive: Triangles, Positions: pos, Normals: nrm, Indices: idx, Colors: cols,
		Color: defaultSurfaceColor, Roughness: 0.6, Opacity: 1, Shading: shading, ObjectID: b.ID(),
	}, true
}

// addFaceColored appends a face's mesh with one shared color for all its vertices.
func addFaceColored(pos *[]math.Point3, nrm *[]math.Vector3, cols *[][4]float32, idx *[]int, m *ops.Mesh, c [4]float32) {
	base := len(*pos)
	for i := range m.Positions {
		*pos, *nrm, *cols = append(*pos, m.Positions[i]), append(*nrm, m.Normals[i]), append(*cols, c)
	}
	for _, vi := range m.Indices {
		*idx = append(*idx, base+vi)
	}
}

// addTriangleColored appends a face's mesh with one color PER TRIANGLE — each triangle gets its own
// three (un-shared) vertices so no two triangles share a color.
func addTriangleColored(pos *[]math.Point3, nrm *[]math.Vector3, cols *[][4]float32, idx *[]int, m *ops.Mesh, index *int) {
	for t := 0; 3*t+2 < len(m.Indices); t++ {
		c := meshDebugColor(*index)
		*index++
		for k := range 3 {
			v := m.Indices[3*t+k]
			*idx = append(*idx, len(*pos))
			*pos, *nrm, *cols = append(*pos, m.Positions[v]), append(*nrm, m.Normals[v]), append(*cols, c)
		}
	}
}

// primitiveCount returns how many face/triangle indices a body consumes, so the running index stays
// aligned when the body is frustum-culled.
func primitiveCount(b *topo.Body, q ops.Quality, perTriangle bool) int {
	if !perTriangle {
		return len(b.Faces())
	}
	n := 0
	for _, f := range b.Faces() {
		n += tessellate.TessellateFace(f, q).TriangleCount()
	}
	return n
}

// meshDebugColor maps a primitive index to a distinct, reproducible RGBA. Hues are spaced by the
// golden ratio so consecutive primitives get well-separated colors; mid saturation/value keeps them
// readable.
func meshDebugColor(i int) [4]float32 {
	hue := stdmath.Mod(float64(i)*0.61803398875, 1.0)
	r, g, b := hsvToRGB(hue, 0.62, 0.92)
	return [4]float32{float32(r), float32(g), float32(b), 1}
}

// hsvToRGB converts h,s,v in [0,1] to r,g,b in [0,1].
func hsvToRGB(h, s, v float64) (r, g, b float64) {
	i := stdmath.Floor(h * 6)
	f := h*6 - i
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)
	switch int(i) % 6 {
	case 0:
		return v, t, p
	case 1:
		return q, v, p
	case 2:
		return p, v, t
	case 3:
		return p, q, v
	case 4:
		return t, p, v
	default:
		return v, p, q
	}
}
