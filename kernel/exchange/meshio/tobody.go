// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
)

// SolidOrSurface welds a triangle soup and converts it to a B-rep body via subd.ToBody:
// a watertight soup becomes a faceted SOLID (b.IsSolid() — downstream features can cut/
// fillet it), an open soup a SURFACE body (left open; not a crash). It fixes inside-out
// winding by rebuilding face-reversed when the signed volume is negative (the swept.go
// pattern), and returns warnings for a non-manifold or open result (the body still
// exists). feat is the persistent-naming lineage root (e.g. "import:stl#0").
//
// Example:
//
//	body, warns := meshio.SolidOrSurface(raw, "import:stl#0", meshio.DefaultWeldTolerance)
func SolidOrSurface(raw RawMesh, feat string, weldTol float64) (*topo.Body, []string, error) {
	cage := Weld(raw, weldTol)
	if len(cage.Faces) == 0 {
		return nil, nil, fmt.Errorf("mesh import: no non-degenerate triangles in %d-triangle soup", raw.TriangleCount())
	}
	body := orientedBody(cage, feat)
	return body, validateWarnings(body), nil
}

// orientedBody builds the body and rebuilds it face-reversed if it came out inside-out
// (negative signed volume) — only meaningful for a closed (solid) cage; an open cage's
// volume is not used to flip it.
func orientedBody(cage subd.Mesh, feat string) *topo.Body {
	body := subd.ToBody(cage, feat)
	if body.IsSolid() && ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume < 0 {
		return subd.ToBody(reverseFaces(cage), feat)
	}
	return body
}

// reverseFaces returns a copy of the cage with every face's winding reversed, flipping
// the outward sense of a built body (used to correct an inside-out solid).
func reverseFaces(m subd.Mesh) subd.Mesh {
	out := m.Clone()
	for i, f := range out.Faces {
		rev := make([]int, len(f))
		for j := range f {
			rev[j] = f[len(f)-1-j]
		}
		out.Faces[i] = rev
	}
	return out
}

// validateWarnings reports non-fatal topology issues so the caller can surface them
// without failing the import (the body is usable; the user decides).
func validateWarnings(b *topo.Body) []string {
	r := ops.Validate(b)
	var warns []string
	if !r.Manifold {
		warns = append(warns, "imported mesh is non-manifold (an edge shared by more than two faces)")
	}
	if b.IsSolid() && !r.OrientationOK {
		warns = append(warns, "imported solid has inconsistent face orientation")
	}
	if !b.IsSolid() {
		warns = append(warns, "imported mesh is not watertight; brought in as an open surface body (not a solid)")
	}
	return warns
}
