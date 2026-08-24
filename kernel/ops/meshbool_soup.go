// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/topo"
)

// Adapter from the kernel topology to the exact mesh-arrangement boolean core
// (kernel/meshbool, ADR-0052). bodyToSoup is the IN half: it tessellates a body to
// a triangle soup meshbool.Boolean can consume. Because the tessellator welds
// vertices along shared edges, the soup is a closed, consistently outward-oriented
// mesh — exactly the operand form the boolean expects. Curved faces are faceted at
// the requested quality (the boolean is a faceted operation, like the planar B-rep
// boolean it will stand beside); planar faces triangulate exactly.

// bodyToSoup returns body b, tessellated at quality q, as an exact triangle soup.
func bodyToSoup(b *topo.Body, q Quality) [][3]meshbool.Point {
	mesh, _ := TessellateBody(b, q)
	return soupFromMesh(mesh)
}

// soupFromMesh converts an indexed tessellation Mesh into a meshbool triangle soup,
// promoting each float64 position to its exact rational Point.
func soupFromMesh(m *Mesh) [][3]meshbool.Point {
	soup := make([][3]meshbool.Point, m.TriangleCount())
	for t := range soup {
		i, j, k := m.Indices[3*t], m.Indices[3*t+1], m.Indices[3*t+2]
		soup[t] = [3]meshbool.Point{
			meshbool.FromPoint3(m.Positions[i]),
			meshbool.FromPoint3(m.Positions[j]),
			meshbool.FromPoint3(m.Positions[k]),
		}
	}
	return soup
}
