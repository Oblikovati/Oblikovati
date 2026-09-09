// SPDX-License-Identifier: GPL-2.0-only

package math_test

import (
	"testing"

	"oblikovati.org/math"
)

// The cost side of ADR-0064. Every product in this package is explicitly rounded, which costs
// NOTHING on amd64 (the compiler never contracted there, so the conversion emits no instruction)
// and costs one instruction per product-sum on arm64, where an FMADDD becomes an FMULD and an
// FADDD. These benchmarks are the arithmetic floor at its densest — a dot product is three
// products and two adds, a 4x4 multiply is sixty-four and forty-eight — so they bound the loss:
// nothing in the kernel is a higher ratio of contractible arithmetic to everything else.
//
//	go test ./math -run XXX -bench . -benchtime 3s
var (
	dotSink    math.Scalar
	vectorSink math.Vector3
	matrixSink math.Matrix4
)

func BenchmarkVector3Dot(b *testing.B) {
	v, o := math.V3(0.3, -1.7, 2.9), math.V3(-0.11, 5.3, 0.017)
	for i := 0; i < b.N; i++ {
		dotSink += v.Dot(o)
	}
}

func BenchmarkVector3Cross(b *testing.B) {
	v, o := math.V3(0.3, -1.7, 2.9), math.V3(-0.11, 5.3, 0.017)
	for i := 0; i < b.N; i++ {
		vectorSink = v.Cross(o)
	}
}

func BenchmarkMatrix4Mul(b *testing.B) {
	m, n := benchTransform(0.7), benchTransform(0.3)
	for i := 0; i < b.N; i++ {
		matrixSink = m.Mul(n)
	}
}

func BenchmarkMatrix4TransformPoint(b *testing.B) {
	m := benchTransform(0.7)
	p := math.P3(0.3, -1.7, 2.9)
	for i := 0; i < b.N; i++ {
		dotSink += m.TransformPoint(p).X
	}
}

// benchTransform is a rotation about an oblique axis composed with a translation and a scale — a
// dense matrix, so no zero cell lets the compiler skip a product.
func benchTransform(angle math.Scalar) math.Matrix4 {
	axis, err := math.NewUnitVector3(0.3, 0.5, 0.81)
	if err != nil {
		panic(err)
	}
	r := math.Rotation4(angle, axis, math.P3(0.2, -0.4, 1.1))
	return math.Translation4(math.V3(1, 2, 3)).Mul(r).Mul(math.Scale4(1.5, 0.9, 1.1))
}
