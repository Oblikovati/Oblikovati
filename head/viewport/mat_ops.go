// SPDX-License-Identifier: GPL-2.0-only

package viewport

import "math"

// This file holds the small column-major 4×4 / vec3 helpers the shadow light-matrix math needs
// (ADR-0026 §6). They target Vulkan clip space (z ∈ [0,1]); the shadow render and the in-shader
// projection both use the result, so they stay self-consistent.

// mul4 multiplies two column-major 4×4 matrices (a·b).
func mul4(a, b [16]float32) [16]float32 {
	var out [16]float32
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			var s float32
			for k := 0; k < 4; k++ {
				s += a[k*4+row] * b[col*4+k]
			}
			out[col*4+row] = s
		}
	}
	return out
}

// lookAt builds a right-handed column-major view matrix (−Z forward) looking from eye to center.
func lookAt(eye, center, up [3]float32) [16]float32 {
	f := normalize3([3]float32{center[0] - eye[0], center[1] - eye[1], center[2] - eye[2]})
	s := normalize3(cross3(f, up))
	u := cross3(s, f)
	return [16]float32{
		s[0], u[0], -f[0], 0,
		s[1], u[1], -f[1], 0,
		s[2], u[2], -f[2], 0,
		-dot3(s, eye), -dot3(u, eye), dot3(f, eye), 1,
	}
}

// ortho builds a right-handed column-major orthographic projection into Vulkan's z ∈ [0,1].
func ortho(l, r, b, t, n, f float32) [16]float32 {
	return [16]float32{
		2 / (r - l), 0, 0, 0,
		0, 2 / (t - b), 0, 0,
		0, 0, -1 / (f - n), 0,
		-(r + l) / (r - l), -(t + b) / (t - b), -n / (f - n), 1,
	}
}

func cross3(a, b [3]float32) [3]float32 {
	return [3]float32{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

func dot3(a, b [3]float32) float32 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

func length3(v [3]float32) float32 { return float32(math.Sqrt(float64(dot3(v, v)))) }

func normalize3(v [3]float32) [3]float32 {
	l := length3(v)
	if l < 1e-8 {
		return [3]float32{0, 0, 1}
	}
	return [3]float32{v[0] / l, v[1] / l, v[2] / l}
}

func absf32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
