// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A boundary-less body (a whole sphere: one face, no vertices or edges) must report its true range box
// from the surface, not an empty box — else boolean classification judges it disjoint from any tool
// (Oblikovati/Oblikovati#1334).
func TestRangeBoxBoundarylessSphere(t *testing.T) {
	sphere, _ := geom.NewSphere(math.P3(1, 2, 3), 5)
	bld := NewBuilder(true, NewLineage(Tok("s", "body", 0)))
	bld.AddFace(sphere, NewLineage(Tok("s", "face", 0)))
	box := bld.Build().RangeBox()
	min, max := box.Min, box.Max
	for _, c := range []struct {
		got, want float64
	}{
		{float64(min.X), -4}, {float64(max.X), 6},
		{float64(min.Y), -3}, {float64(max.Y), 7},
		{float64(min.Z), -2}, {float64(max.Z), 8},
	} {
		if stdmath.Abs(c.got-c.want) > 1e-9 {
			t.Errorf("range box bound %.4f, want %.4f (sphere centre (1,2,3) r=5)", c.got, c.want)
		}
	}
}
