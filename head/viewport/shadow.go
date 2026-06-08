// SPDX-License-Identifier: GPL-2.0-only

package viewport

import (
	"math"

	"oblikovati/renderer"
)

// SceneBounds returns the axis-aligned min/max of a flattened mesh's shaded + occluder triangle
// positions, and ok=false when there is no geometry. The shadow pass fits its light frustum to
// these bounds so the whole model casts into the shadow map (ADR-0026 §6). Pure Go (no GPU) so
// the framing math is unit-tested.
func SceneBounds(m Mesh) (min, max [3]float32, ok bool) {
	min = [3]float32{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
	max = [3]float32{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
	accumulatePositions(m.TriVerts, &min, &max)
	accumulatePositions(m.OccVerts, &min, &max)
	return min, max, min[0] <= max[0]
}

// DrawListBounds returns the shadow bounds directly from a draw list — the min/max of every
// non-on-top triangle item's positions, the same set SceneBounds reads from a flattened mesh. It
// lets the viewport size the shadow frustum without a throwaway Flatten (which is then done once,
// after the ground plane is appended).
func DrawListBounds(list renderer.DrawList) (min, max [3]float32, ok bool) {
	min = [3]float32{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
	max = [3]float32{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
	for _, it := range list.Items {
		if it.Primitive != renderer.Triangles || it.OnTop {
			continue
		}
		for _, p := range it.Positions {
			widen(&min, &max, float32(p.X), float32(p.Y), float32(p.Z))
		}
	}
	return min, max, min[0] <= max[0]
}

// widen expands the min/max box to include (x,y,z).
func widen(min, max *[3]float32, x, y, z float32) {
	for i, v := range [3]float32{x, y, z} {
		if v < min[i] {
			min[i] = v
		}
		if v > max[i] {
			max[i] = v
		}
	}
}

// accumulatePositions widens min/max by every vertex position (the first 3 of each 16-float
// vertex) in verts.
func accumulatePositions(verts []float32, min, max *[3]float32) {
	for i := 0; i+VertexFloats <= len(verts); i += VertexFloats {
		for a := 0; a < 3; a++ {
			v := verts[i+a]
			if v < min[a] {
				min[a] = v
			}
			if v > max[a] {
				max[a] = v
			}
		}
	}
}

// LightMatrix builds the column-major light-space view-projection that frames the bounds box
// from the direction of a directional light (lightDir points from the scene toward the light).
// It is an orthographic fit centered on the box, sized to its bounding sphere, used both to
// render the shadow map and to project world points into it in the shader (ADR-0026 §6).
func LightMatrix(min, max [3]float32, lightDir [3]float32) [16]float32 {
	center := [3]float32{(min[0] + max[0]) / 2, (min[1] + max[1]) / 2, (min[2] + max[2]) / 2}
	radius := 0.5 * length3([3]float32{max[0] - min[0], max[1] - min[1], max[2] - min[2]})
	if radius < 1e-4 {
		radius = 1
	}
	l := normalize3(lightDir)
	eye := [3]float32{center[0] + l[0]*radius*2, center[1] + l[1]*radius*2, center[2] + l[2]*radius*2}
	view := lookAt(eye, center, pickUp(l))
	proj := ortho(-radius, radius, -radius, radius, 0.01, 4*radius)
	return mul4(proj, view)
}

// pickUp chooses an up vector not parallel to the light direction (so the look-at basis is
// well-defined when the light is near vertical).
func pickUp(l [3]float32) [3]float32 {
	if absf32(l[2]) > 0.99 {
		return [3]float32{0, 1, 0}
	}
	return [3]float32{0, 0, 1}
}
