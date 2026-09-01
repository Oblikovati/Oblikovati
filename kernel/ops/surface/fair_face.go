// SPDX-License-Identifier: GPL-2.0-only

package surface

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/internal/retopo"
	"oblikovati.org/kernel/topo"
)

// Fairing a surface body's NURBS face (M36-F04): relax the interior control points to smooth
// curvature wrinkles while holding the boundary band (geom.FairSurface), and rebuild the face on the
// unchanged domain. holdOrder preserves the boundary continuity (0=G0,1=G1,2=G2).

// FairFaceSurface returns a surface body whose NURBS face is faired (interior smoothed, boundary held
// to holdOrder) by strength over iterations. It errors when the body has no NURBS face.
func FairFaceSurface(b *topo.Body, holdOrder int, strength float64, iterations int) (*topo.Body, error) {
	_, s, ok := probe.FirstNurbsFace(b)
	if !ok {
		return nil, fmt.Errorf("surface.FairFaceSurface: body has no NURBS surface face")
	}
	faired := geom.FairSurface(s, holdOrder, strength, iterations)
	return retopo.FullDomainBody(faired, "fair"), nil
}
