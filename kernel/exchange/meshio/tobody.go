// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
)

// ErrNonManifoldImport classifies a mesh import REFUSED because the welded body is non-manifold — an
// edge shared by more than two faces. A non-manifold body is invalid topology that corrupts every
// downstream consumer (boolean input, mass properties, export), so the import fails here rather than
// handing back a broken body with only a warning (#3384). A caller tells this decline from a real I/O
// failure with errors.Is(err, ErrNonManifoldImport).
var ErrNonManifoldImport = errors.New("mesh import: non-manifold body")

// SolidOrSurface welds a triangle soup and converts it to a B-rep body via subd.ToBody:
// a watertight soup becomes a faceted SOLID (b.IsSolid() — downstream features can cut/
// fillet it), an open soup a SURFACE body (left open; not a crash). It fixes inside-out
// winding by rebuilding face-reversed when the signed volume is negative (the swept.go
// pattern). A non-manifold weld is REFUSED with ErrNonManifoldImport (#3384); a merely
// open or inconsistently-oriented result imports with a warning (the body is usable). feat
// is the persistent-naming lineage root (e.g. "import:stl#0").
//
// Example:
//
//	body, warns, err := meshio.SolidOrSurface(raw, "import:stl#0", meshio.DefaultWeldTolerance)
func SolidOrSurface(raw RawMesh, feat string, weldTol float64) (*topo.Body, []string, error) {
	cage, dropped := Weld(raw, weldTol)
	if len(cage.Faces) == 0 {
		return nil, nil, fmt.Errorf("mesh import: no non-degenerate triangles in %d-triangle soup", raw.TriangleCount())
	}
	var warns []string
	if dropped > 0 {
		// Dropped triangles are discarded geometry — surface them like the DWG decoder's
		// per-entity warnings instead of thinning the mesh silently (#1638).
		warns = append(warns, fmt.Sprintf("%d of %d triangles were degenerate after welding and were dropped", dropped, raw.TriangleCount()))
	}
	body := orientedBody(cage, feat)
	r := ops.Validate(body)
	if !r.Manifold {
		// A non-manifold body is not a warn-and-continue case: it is invalid topology that would
		// silently corrupt every downstream consumer. Refuse the import, naming the offending count.
		return nil, nil, fmt.Errorf("%w: %d non-manifold edge(s) shared by more than two faces in a %d-triangle soup; "+
			"a mesh imports only as a 2-manifold body (each edge on at most two faces)",
			ErrNonManifoldImport, nonManifoldEdgeCount(r), raw.TriangleCount())
	}
	return body, append(warns, softValidateWarnings(body, r)...), nil
}

// nonManifoldEdgeCount counts the non-manifold edges the validation reported — the offending value the
// decline names.
func nonManifoldEdgeCount(r ops.ValidationReport) int {
	n := 0
	for _, issue := range r.Issues {
		if len(issue) >= 17 && issue[:17] == "non-manifold edge" {
			n++
		}
	}
	return n
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

// softValidateWarnings reports the NON-FATAL topology issues so the caller can surface them without
// failing the import (the body is usable; the user decides). The non-manifold case is NOT here — it is
// a hard decline in SolidOrSurface (#3384) — so this only covers an inconsistently-oriented solid and
// an open (non-watertight) surface body, which are legitimate importable outcomes. It takes the
// already-computed report so the body is validated once.
func softValidateWarnings(b *topo.Body, r ops.ValidationReport) []string {
	var warns []string
	if b.IsSolid() && !r.OrientationOK {
		warns = append(warns, "imported solid has inconsistent face orientation")
	}
	if !b.IsSolid() {
		warns = append(warns, "imported mesh is not watertight; brought in as an open surface body (not a solid)")
	}
	return warns
}
