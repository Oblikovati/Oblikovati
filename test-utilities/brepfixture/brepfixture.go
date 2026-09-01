// SPDX-License-Identifier: GPL-2.0-only

// Package brepfixture builds the B-rep bodies kernel tests use as operands.
//
// It exists because kernel/ops is split by operation (#2183): a fixture two sibling packages
// both need cannot live in either one's _test.go files, since Go does not share test helpers
// across packages. Everything here is a plain constructor over kernel/subd and kernel/topo —
// no operation, no assertion — so it can never make a test pass for the wrong reason.
package brepfixture

import (
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Box builds a faceted axis-aligned box with its minimum corner at p and the given extents.
// The result is a subdivision cage converted to a B-rep, so its walls are triangles — the
// operand shape the CSG and mesh-arrangement paths are specified against.
//
// Example: a := brepfixture.Box(math.P3(0, 0, 0), 2, 2, 2) // the unit CSG operand, volume 8
func Box(p math.Point3, sx, sy, sz float64) *topo.Body {
	cage := subd.Box(sx, sy, sz)
	for i := range cage.Verts {
		cage.Verts[i] = cage.Verts[i].TranslateBy(p.AsVector())
	}
	return subd.ToBody(cage, "box")
}
