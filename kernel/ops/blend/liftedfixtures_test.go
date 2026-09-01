// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
)

// Fixture builders restated from kernel/ops' test package: Go cannot share a _test.go
// helper across packages, and the fillet family moved out from under them. This is the
// test scaffolding sonar.cpd.exclusions already accounts for.

// zPrism extrudes a closed XY polygon between z0 and z1 into a solid prism (winding per the proven
// brep prismBody helper: bottom reversed, top forward, quad sides).
func zPrism(poly []m.Point2, z0, z1 float64, feat string) *topo.Body {
	n := len(poly)
	verts := make([]m.Point3, 0, n*2)
	for _, p := range poly {
		verts = append(verts, m.P3(p.X, p.Y, m.Scalar(z0)))
	}
	for _, p := range poly {
		verts = append(verts, m.P3(p.X, p.Y, m.Scalar(z1)))
	}
	bottom, top := make([]int, n), make([]int, n)
	for i := range poly {
		bottom[i] = n - 1 - i
		top[i] = n + i
	}
	faces := [][]int{bottom, top}
	for i := range poly {
		next := (i + 1) % n
		faces = append(faces, []int{i, next, next + n, i + n})
	}
	return subd.ToBody(subd.Mesh{Verts: verts, Faces: faces}, feat)
}
