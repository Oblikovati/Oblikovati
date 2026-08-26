// SPDX-License-Identifier: GPL-2.0-only

package benchgen

import (
	"fmt"
	"math"
	"path"

	"oblikovati.org/kernel/ops"
	obkmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// buildTierPart creates a part document holding one extruded n-gon prism standing in
// for a tier's geometry, recomputed to a realized body so it tessellates and renders.
// The n-gon's side count (spec.Sides) is the poly-weight knob: a hexagon for fasteners,
// a many-sided prism for machined/system parts (dense facets in place of fillets/bores).
func buildTierPart(ws *doc.Workspace, name string, spec TierSpec) (*doc.Document, error) {
	d, err := compdef.AddPart(ws, name, false)
	if err != nil {
		return nil, fmt.Errorf("benchgen: add part %q: %w", name, err)
	}
	def, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		return nil, fmt.Errorf("benchgen: part %q content is %T, want *PartComponentDefinition", name, d.Content())
	}
	sk := def.Sketches().Add(sketch.XYPlane())
	if err := sketchRegularPolygon(sk, spec.Sides, spec.RadiusCm); err != nil {
		return nil, err
	}
	height := spec.HeightCm
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return height })
	def.Recompute()
	return d, nil
}

// sketchRegularPolygon lays a closed, fully-determined regular n-gon of the given
// radius on the sketch — the profile every tier part extrudes. It rejects degenerate
// inputs (a profile must have at least three sides and a positive radius) with the
// offending value, since an open or zero-area loop yields no extrudable profile.
func sketchRegularPolygon(sk *sketch.Sketch, sides int, radiusCm float64) error {
	if sides < 3 {
		return fmt.Errorf("benchgen: polygon needs >=3 sides, got %d", sides)
	}
	if radiusCm <= 0 {
		return fmt.Errorf("benchgen: polygon radius must be > 0 cm, got %g", radiusCm)
	}
	corners := make([]*sketch.Point, sides)
	for i := range sides {
		theta := 2 * math.Pi * float64(i) / float64(sides)
		corners[i] = sk.Points().Add(obkmath.P2(radiusCm*math.Cos(theta), radiusCm*math.Sin(theta)))
	}
	for i := range sides {
		sk.Lines().Add(corners[i], corners[(i+1)%sides])
	}
	return nil
}

// buildPool builds spec.UniqueMeshes distinct part documents for one tier under
// dirPrefix, returning them in pool order. These are the unique definitions the tree
// round-robin places to reach the tier's placement count — so memory holds one body
// per pool entry however many times it is placed (the flyweight the benchmark probes).
func buildPool(ws *doc.Workspace, dirPrefix string, spec TierSpec) ([]*doc.Document, error) {
	pool := make([]*doc.Document, spec.UniqueMeshes)
	for i := 0; i < spec.UniqueMeshes; i++ {
		name := path.Join(dirPrefix, spec.Tier.String(), fmt.Sprintf("%s_%06d", spec.Tier, i)) + doc.Part.Extension()
		d, err := buildTierPart(ws, name, spec)
		if err != nil {
			return nil, err
		}
		pool[i] = d
	}
	return pool, nil
}
