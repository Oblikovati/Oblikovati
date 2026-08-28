// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Dress-up serialization — the SHELL / DRAFT data (M48 #2237 split of serialize_dressup.go). The draft
// pull-direction and neutral-plane encoding and the shell per-face thickness data. The shared reference/
// anchor encoding lives in serialize_dressup.go.

// draftPull reads a serialized pull direction [dx,dy,dz], defaulting to +Z when absent
// (older recipes / the common Z-up mould pull).
func draftPull(p []float64) math.Vector3 {
	if len(p) != 3 {
		return math.V3(0, 0, 1)
	}
	return math.V3(p[0], p[1], p[2])
}

// draftNeutral reads a serialized draft neutral (parting) plane from its origin and normal, or nil when
// absent/degenerate — the implicit lowest-vertex hinge (#1801).
func draftNeutral(origin, normal []float64) *geom.Plane {
	if len(origin) != 3 || len(normal) != 3 {
		return nil
	}
	pl, err := geom.NewPlane(math.P3(origin[0], origin[1], origin[2]), math.V3(normal[0], normal[1], normal[2]))
	if err != nil {
		return nil
	}
	return &pl
}

// neutralData serializes a draft neutral plane to (origin, normal) float triples, or (nil, nil) when
// no neutral plane is set.
func neutralData(pl *geom.Plane) (origin, normal []float64) {
	if pl == nil {
		return nil, nil
	}
	n := pl.Normal()
	return []float64{pl.Origin.X, pl.Origin.Y, pl.Origin.Z}, []float64{float64(n.X), float64(n.Y), float64(n.Z)}
}

// This file holds the YAML codecs for the dress-up family (fillet/chamfer/shell/draft/
// thread) and the reference-key + scalar helpers shared across feature codecs. Edge and
// face reference keys are opaque lineage bytes, so they are base64-encoded (ADR-0020)
// and re-bind to the regenerated topology after recompute.

// FaceThicknessData is one retained face's own wall thickness in a shell (#1864).
type FaceThicknessData struct {
	Face      string  `yaml:"face"`
	Thickness float64 `yaml:"thickness"`
}
