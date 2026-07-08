// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/kernel/topo"
)

// This file wires the wire-level geometric selectors (featureargs.GeomFaceSel / GeomEdgeSel) to
// the kernel's geometric references (topo.GeometricFaceRef / GeometricEdgeRef). An external author
// cannot mint lineage reference keys and a located key does not survive the feature's recompute, so
// a hole/fillet/chamfer/shell/draft it authors selects its faces/edges by GEOMETRY; the feature
// rebinds them to body topology every recompute (ADR-0040, model bindGeomFaces/bindGeomEdges).

// geomFaceRef converts one wire face selector to a kernel geometric face reference.
func geomFaceRef(sel featureargs.GeomFaceSel) (topo.GeometricFaceRef, error) {
	centroid, err := point3(sel.Centroid, "geometric face: centroid")
	if err != nil {
		return topo.GeometricFaceRef{}, err
	}
	normal, err := vec3(sel.Normal, "geometric face: normal")
	if err != nil {
		return topo.GeometricFaceRef{}, err
	}
	return topo.GeometricFaceRef{Centroid: centroid, Normal: normal}, nil
}

// geomFaceRefs converts a list of wire face selectors to kernel geometric face references.
func geomFaceRefs(sels []featureargs.GeomFaceSel) ([]topo.GeometricFaceRef, error) {
	refs := make([]topo.GeometricFaceRef, len(sels))
	for i, sel := range sels {
		ref, err := geomFaceRef(sel)
		if err != nil {
			return nil, err
		}
		refs[i] = ref
	}
	return refs, nil
}

// geomEdgeRefs converts a list of wire edge selectors to kernel geometric edge references.
func geomEdgeRefs(sels []featureargs.GeomEdgeSel) ([]topo.GeometricEdgeRef, error) {
	refs := make([]topo.GeometricEdgeRef, len(sels))
	for i, sel := range sels {
		midpoint, err := point3(sel.Midpoint, "geometric edge: midpoint")
		if err != nil {
			return nil, err
		}
		direction, err := vec3(sel.Direction, "geometric edge: direction")
		if err != nil {
			return nil, err
		}
		refs[i] = topo.GeometricEdgeRef{Midpoint: midpoint, Direction: direction}
	}
	return refs, nil
}
