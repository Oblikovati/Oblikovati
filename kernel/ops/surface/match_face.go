// SPDX-License-Identifier: GPL-2.0-only

package surface

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/topo"
)

// Matching one surface body's NURBS face to a target body's face (M36-F05): rebuild the source
// face's surface against the target to the chosen continuity (geom.MatchSurface) and swap it back in
// with transform.ReplaceFaceSurface (loops preserved). Both bodies must carry a single NURBS face whose chosen
// edges correspond along the seam.

// MatchFaceTo returns a copy of src whose NURBS face is matched to target's NURBS face along the
// given edges to the continuity order (0=G0..3=G3). It errors when either body lacks a NURBS face or
// the match is invalid.
func MatchFaceTo(src, target *topo.Body, srcEdge, tgtEdge geom.Boundary, order int) (*topo.Body, error) {
	sf, ss, ok := probe.FirstNurbsFace(src)
	if !ok {
		return nil, fmt.Errorf("surface.MatchFaceTo: source body has no NURBS surface face")
	}
	_, ts, ok := probe.FirstNurbsFace(target)
	if !ok {
		return nil, fmt.Errorf("surface.MatchFaceTo: target body has no NURBS surface face")
	}
	matched, err := geom.MatchSurface(ss, ts, srcEdge, tgtEdge, order)
	if err != nil {
		return nil, fmt.Errorf("surface.MatchFaceTo: %w", err)
	}
	return transform.ReplaceFaceSurface(src, sf.ReferenceKey(), matched)
}
