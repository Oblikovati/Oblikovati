// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	stdmath "math"

	"oblikovati/math"
	"oblikovati/scene"
)

// GPU-facing camera transforms in float32, column-major (the layout GLSL/SPIR-V
// `mat4` expects). math.Matrix4 is affine-only by contract — perspective lives here,
// at the renderer layer, never in the kernel. These are pure functions so the matrix
// math is unit-tested headlessly (ADR-0014), and both the Vulkan viewport and the
// offscreen image oracle consume the same ViewProjection.

// ViewProjection returns proj·view as a column-major [16]float32, ready to upload as
// a push constant / UBO. near and far are the clip planes in world units (near > 0,
// far > near). It maps a world point to Vulkan clip space (y-down, depth 0..1).
func ViewProjection(cam scene.Camera, near, far float64) [16]float32 {
	return mul4(projection(cam, near, far), view(cam))
}

// Project maps a world point to viewport pixel coordinates (origin top-left, x right,
// y down), using the same view·projection the viewport uploads — so an overlay drawn at
// (x, y) sits exactly over the 3D point (e.g. a dimension's value label over its anchor).
// ok is false when the point is at or behind the camera (clip w ≤ 0, not on screen).
func Project(cam scene.Camera, near, far float64, p math.Point3) (x, y float64, ok bool) {
	vp := ViewProjection(cam, near, far)
	v := [4]float64{p.X, p.Y, p.Z, 1}
	var clip [4]float64
	for r := 0; r < 4; r++ {
		clip[r] = float64(vp[0*4+r])*v[0] + float64(vp[1*4+r])*v[1] +
			float64(vp[2*4+r])*v[2] + float64(vp[3*4+r])*v[3]
	}
	if clip[3] <= 0 {
		return 0, 0, false
	}
	ndcX, ndcY := clip[0]/clip[3], clip[1]/clip[3] // Vulkan clip y already points down
	return (ndcX*0.5 + 0.5) * float64(cam.Width), (ndcY*0.5 + 0.5) * float64(cam.Height), true
}

// view is the right-handed world→eye transform (camera looks down its local −Z),
// column-major.
func view(cam scene.Camera) [16]float32 {
	f := norm(sub(cam.Target, cam.Eye))
	s := norm(cross(f, [3]float64{cam.Up.X, cam.Up.Y, cam.Up.Z}))
	u := cross(s, f)
	e := [3]float64{cam.Eye.X, cam.Eye.Y, cam.Eye.Z}
	return [16]float32{
		f32(s[0]), f32(u[0]), f32(-f[0]), 0,
		f32(s[1]), f32(u[1]), f32(-f[1]), 0,
		f32(s[2]), f32(u[2]), f32(-f[2]), 0,
		f32(-dot(s, e)), f32(-dot(u, e)), f32(dot(f, e)), 1,
	}
}

// projection is a right-handed perspective with Vulkan conventions: the Y axis is
// flipped (clip-space y points down) and depth maps to [0, 1]. Column-major.
func projection(cam scene.Camera, near, far float64) [16]float32 {
	aspect := float64(cam.Width) / float64(cam.Height)
	g := 1 / stdmath.Tan(cam.FOV/2) // focal length from vertical FOV
	zr := far / (near - far)
	return [16]float32{
		f32(g / aspect), 0, 0, 0,
		0, f32(-g), 0, 0,
		0, 0, f32(zr), -1,
		0, 0, f32(near * zr), 0,
	}
}

// mul4 multiplies two column-major 4×4 matrices (a·b).
func mul4(a, b [16]float32) [16]float32 {
	var out [16]float32
	for c := 0; c < 4; c++ {
		for r := 0; r < 4; r++ {
			out[c*4+r] = a[0*4+r]*b[c*4+0] + a[1*4+r]*b[c*4+1] +
				a[2*4+r]*b[c*4+2] + a[3*4+r]*b[c*4+3]
		}
	}
	return out
}

// sub returns a−b as a plain coordinate triple (world-space displacement).
func sub(a, b math.Point3) [3]float64 { return [3]float64{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }

func f32(v float64) float32 { return float32(v) }

func dot(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

func cross(a, b [3]float64) [3]float64 {
	return [3]float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

func norm(v [3]float64) [3]float64 {
	l := stdmath.Sqrt(dot(v, v))
	if l == 0 {
		return v
	}
	return [3]float64{v[0] / l, v[1] / l, v[2] / l}
}
