// SPDX-License-Identifier: GPL-2.0-only

package viewport

import "oblikovati.org/renderer"

// SceneUBOFloats is the size, in float32s, of the scene-lighting uniform buffer the mesh
// pipeline binds — its std140 layout must match the Scene block in mesh.frag exactly:
//
//	vec4 header;                 // x=ambience y=brightness z=exposure w=lightCount
//	struct { vec4 dir; vec4 color; vec4 pos; } lights[MaxSceneLights];
//
// Each light is 3 vec4s (12 floats), so the total is 4 + 12·MaxSceneLights. Keeping the
// packing in pure Go (no cgo) lets the layout be unit-tested headlessly (ADR-0014); the native
// layer memcpy's the result straight into the host-visible UBO.
const (
	lightFloats    = 12 // dir(4) + color(4) + pos(4)
	headerFloats   = 4
	SceneUBOFloats = headerFloats + lightFloats*renderer.MaxSceneLights
)

// PackLighting lays out a [renderer.SceneLighting] as the std140 float array the scene UBO
// expects. Light direction is stored as the unit vector toward the light (the shader
// re-normalizes); the .w lanes carry the light kind, intensity, and on-flag so one vec4 array
// covers directional/point/spot without a side table.
//
//	floats := viewport.PackLighting(session.SceneLighting())
//	win.SetViewportLighting(floats)
func PackLighting(l renderer.SceneLighting) []float32 {
	out := make([]float32, SceneUBOFloats)
	lights := l.ActiveLights()
	out[0] = l.Ambience
	out[1] = nonZero(l.Brightness, 1)
	out[2] = nonZero(l.Exposure, 1)
	out[3] = float32(len(lights))
	for i, lt := range lights {
		base := headerFloats + i*lightFloats
		out[base+0], out[base+1], out[base+2] = lt.Direction[0], lt.Direction[1], lt.Direction[2]
		out[base+3] = float32(lt.Kind)
		out[base+4], out[base+5], out[base+6] = lt.Color[0], lt.Color[1], lt.Color[2]
		out[base+7] = lt.Intensity
		out[base+8], out[base+9], out[base+10] = lt.Position[0], lt.Position[1], lt.Position[2]
		out[base+11] = 1 // on-flag (ActiveLights already filtered Off lights)
	}
	return out
}

// nonZero returns v, or fallback when v is zero — so an un-set Brightness/Exposure (a zero
// value from a partial rig) does not darken the scene to black.
func nonZero(v, fallback float32) float32 {
	if v == 0 {
		return fallback
	}
	return v
}
