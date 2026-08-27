// SPDX-License-Identifier: GPL-2.0-only

package renderer

import "math"

// vecFloat is the element type of the [3]T triples used across this package: ray-tracing
// math stays float32 to match the Vulkan-facing vertex/hit buffers both GPU Intersector
// backends (PBI-333/334) produce and consume, while the camera-transform math stays
// float64. Go had no way to share one implementation across both before generic methods
// landed (Go 1.27) — dot32/cross32/sub32/normalize32 and dot/cross/norm used to be
// separate, identical-shaped functions purely because Go has no overloading.
type vecFloat interface{ ~float32 | ~float64 }

func dot3[T vecFloat](a, b [3]T) T { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

func cross3[T vecFloat](a, b [3]T) [3]T {
	return [3]T{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}

func sub3[T vecFloat](a, b [3]T) [3]T { return [3]T{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }

// normalize3 returns v scaled to unit length, or v unchanged when it is the zero vector.
func normalize3[T vecFloat](v [3]T) [3]T {
	l := T(math.Sqrt(float64(dot3(v, v))))
	if l == 0 {
		return v
	}
	return [3]T{v[0] / l, v[1] / l, v[2] / l}
}
