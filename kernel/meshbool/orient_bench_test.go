// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "testing"

// benchQuad is a generic float-exact tetrahedron for the orientation benchmarks.
func benchQuad() (a, b, c, d Point) {
	return FromCoords(0.1, 0.2, 0.3), FromCoords(1.7, 0.1, -0.2),
		FromCoords(0.3, 1.9, 0.4), FromCoords(0.6, 0.7, 1.3)
}

// BenchmarkOrient3DFloatExact measures the delegated float fast path taken when all
// four vertices are exact binary64 — the common case for original tessellation
// vertices. Compare against BenchmarkOrient3DRational.
func BenchmarkOrient3DFloatExact(b *testing.B) {
	p, q, r, s := benchQuad()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Orient3D(p, q, r, s)
	}
}

// BenchmarkOrient3DRational measures the pure-rational path (orient3DVal), the cost
// paid for a constructed intersection vertex that is not a binary64.
func BenchmarkOrient3DRational(b *testing.B) {
	p, q, r, s := benchQuad()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = orient3DVal(p, q, r, s).Sign()
	}
}
