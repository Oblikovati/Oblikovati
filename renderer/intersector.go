// SPDX-License-Identifier: GPL-2.0-only

package renderer

import "math"

// Ray is one ray-tracing query: an origin and a (not necessarily normalized) direction,
// with the valid hit range [TMin, TMax] measured in units of Direction.
type Ray struct {
	Origin, Direction [3]float32
	TMin, TMax        float32
}

// Hit is the result of a ray query: the nearest intersection along the ray, or a
// zero-value Hit (Hit == false) when nothing was struck within [TMin, TMax].
type Hit struct {
	Hit                     bool
	T                       float32
	Point, Normal           [3]float32
	InstanceID, PrimitiveID uint32
}

// Triangle is one triangle-soup primitive: three world-space vertices plus the owning
// instance/primitive identifiers a [Hit] reports back — the same identifiers the
// hardware BLAS/TLAS (PBI-333) and the software BVH (PBI-334) both carry, so a caller
// can resolve a hit to a body/face regardless of which backend found it.
type Triangle struct {
	V0, V1, V2              [3]float32
	InstanceID, PrimitiveID uint32
}

// Intersector answers nearest-hit ray queries against the current scene's triangle
// geometry — the seam behind which the hardware (PBI-333, vkCmdTraceRaysKHR/ray query)
// and software (PBI-334, compute-shader BVH) ray-tracing backends both sit, so
// path-integration/BSDF code (F03/F04) stays backend-agnostic and CPU-testable
// (ADR-0014, ADR-0053). Toggling the hardware-RT checkbox swaps which Intersector the
// path tracer holds; nothing above this seam changes.
type Intersector interface {
	// TraceRay returns the nearest hit along ray within [ray.TMin, ray.TMax].
	TraceRay(ray Ray) Hit
}

// IntersectorFactory builds a fresh Intersector over a triangle soup. Each backend
// (FakeIntersector here; the hardware/software GPU backends in PBI-333/334) implements
// one, so a shared contract-test suite (renderer/rttest) can exercise all of them
// identically.
type IntersectorFactory func(triangles []Triangle) Intersector

// FakeIntersector is a deterministic, in-memory Intersector: brute-force
// Möller–Trumbore ray-triangle intersection with no GPU, BVH, or acceleration
// structure. It is both the test double for code that only needs *an* Intersector, and
// the CPU-reference oracle PBI-333/334 cross-check their real GPU backends against
// (ADR-0014).
type FakeIntersector struct {
	Triangles []Triangle
}

// NewFakeIntersector builds a FakeIntersector over triangles.
func NewFakeIntersector(triangles []Triangle) *FakeIntersector {
	return &FakeIntersector{Triangles: triangles}
}

// TraceRay linearly scans every triangle and returns the nearest hit — O(n) by design;
// this type exists for correctness and testability, never performance.
func (f *FakeIntersector) TraceRay(ray Ray) Hit {
	best := Hit{}
	bestT := ray.TMax
	for _, tri := range f.Triangles {
		t, n, ok := rayTriangle(ray, tri)
		if !ok || t < ray.TMin || t >= bestT {
			continue
		}
		bestT = t
		best = Hit{
			Hit: true, T: t, Normal: n,
			Point:       addScaled(ray.Origin, ray.Direction, t),
			InstanceID:  tri.InstanceID,
			PrimitiveID: tri.PrimitiveID,
		}
	}
	return best
}

// rayTriangle is the Möller–Trumbore ray-triangle intersection test: returns the ray
// parameter t and the triangle's (unnormalized-input-independent, always outward for a
// CCW-wound triangle as seen from the ray origin side) geometric normal, or ok=false for
// a miss or a ray parallel to the triangle's plane.
func rayTriangle(ray Ray, tri Triangle) (t float32, normal [3]float32, ok bool) {
	e1 := sub32(tri.V1, tri.V0)
	e2 := sub32(tri.V2, tri.V0)
	pvec := cross32(ray.Direction, e2)
	det := dot32(e1, pvec)
	const epsilon = 1e-8
	if det > -epsilon && det < epsilon {
		return 0, [3]float32{}, false
	}
	invDet := 1 / det
	tvec := sub32(ray.Origin, tri.V0)
	u := dot32(tvec, pvec) * invDet
	if u < 0 || u > 1 {
		return 0, [3]float32{}, false
	}
	qvec := cross32(tvec, e1)
	v := dot32(ray.Direction, qvec) * invDet
	if v < 0 || u+v > 1 {
		return 0, [3]float32{}, false
	}
	t = dot32(e2, qvec) * invDet
	return t, normalize32(cross32(e1, e2)), true
}

// sub32 / cross32 / dot32 / normalize32 are [3]float32 vector helpers, distinct from
// transform.go's [3]float64 sub/cross/dot/norm (Go has no overloading, so these can't
// share names) — ray-tracing math stays float32 to match the Vulkan-facing vertex/hit
// buffers both GPU Intersector backends (PBI-333/334) will produce and consume.
func sub32(a, b [3]float32) [3]float32 { return [3]float32{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }

func cross32(a, b [3]float32) [3]float32 {
	return [3]float32{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}

func dot32(a, b [3]float32) float32 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

func addScaled(a, dir [3]float32, t float32) [3]float32 {
	return [3]float32{a[0] + dir[0]*t, a[1] + dir[1]*t, a[2] + dir[2]*t}
}

func normalize32(v [3]float32) [3]float32 {
	l := dot32(v, v)
	if l == 0 {
		return v
	}
	inv := float32(1) / float32(math.Sqrt(float64(l)))
	return [3]float32{v[0] * inv, v[1] * inv, v[2] * inv}
}
