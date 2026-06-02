// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/scene"
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
