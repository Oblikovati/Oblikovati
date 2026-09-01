// SPDX-License-Identifier: GPL-2.0-only

package brepfixture

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// CountCylinderFaces counts the body's cylindrical faces — how a fillet, a boolean or a
// draft test asserts that the walls it expected were actually built.
//
// Example: if brepfixture.CountCylinderFaces(res) != 4 { /* a rounded edge is missing */ }
func CountCylinderFaces(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			n++
		}
	}
	return n
}
