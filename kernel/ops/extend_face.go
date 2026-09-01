// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/internal/retopo"
	"oblikovati.org/kernel/topo"
)

// Extending a surface body's NURBS face (M36-F11): lengthen the face's surface past an edge with
// geom.ExtendSurface and rebuild the face over the grown domain (fullDomainBody), so the face shows
// the extension. The continuation order (1=linear/tangent G1, 2=curvature G2) controls how the
// appended span meets the original boundary.

// ExtendFaceSurface returns a surface body whose NURBS face is extended past `edge` by `distance`
// with the given continuation order. It errors when the body has no NURBS face or the extend is
// invalid.
func ExtendFaceSurface(b *topo.Body, edge geom.Boundary, distance float64, order int) (*topo.Body, error) {
	_, s, ok := probe.FirstNurbsFace(b)
	if !ok {
		return nil, fmt.Errorf("ops.ExtendFaceSurface: body has no NURBS surface face")
	}
	ext, err := geom.ExtendSurface(s, edge, distance, order)
	if err != nil {
		return nil, fmt.Errorf("ops.ExtendFaceSurface: %w", err)
	}
	return retopo.FullDomainBody(ext, "extend"), nil
}
