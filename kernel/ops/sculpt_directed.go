// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// SculptDirected builds the solid enclosed by a set of PLANAR bounding surfaces, each with a
// keep-side flag naming the direction that faces the volume — Inventor's directed sculpt, where a
// per-surface direction disambiguates the enclosed region of OPEN bounding surfaces (#1881). It
// starts from a bounding cube around the surfaces and cuts it by each surface's directed halfspace;
// the intersection is the enclosed solid. keepPositive[i] keeps the surface's +normal side (its
// −normal side otherwise). Curved or non-convex bounding surfaces are phase C.
func SculptDirected(surfaces []*topo.Body, keepPositive []bool, feat string) (*topo.Body, error) {
	if len(surfaces) < 2 || len(keepPositive) != len(surfaces) {
		return nil, errors.New("sculpt: need at least two bounding surfaces, one direction each")
	}
	solid, err := boundingCube(surfaces, feat)
	if err != nil {
		return nil, err
	}
	for i, surf := range surfaces {
		pl, ok := BodyPlane(surf)
		if !ok {
			return nil, fmt.Errorf("sculpt: bounding surface %d is not planar", i)
		}
		n := pl.Normal()
		if keepPositive[i] {
			n = n.Scale(-1) // HalfSpaceCut keeps the side OPPOSITE the plane normal
		}
		cut, err := geom.NewPlane(pl.Origin, n)
		if err != nil {
			return nil, fmt.Errorf("sculpt: bounding surface %d: %w", i, err)
		}
		if solid, err = brep.HalfSpaceCut(solid, cut); err != nil {
			return nil, fmt.Errorf("sculpt: cut by surface %d: %w", i, err)
		}
	}
	if r := Validate(solid); !r.Valid || !solid.IsSolid() {
		return nil, fmt.Errorf("sculpt: directed surfaces do not bound a solid %v", r.Issues)
	}
	return solid, nil
}

// boundingCube returns a solid box enclosing all the surfaces with a margin, so the directed
// halfspace cuts (not the box) bound the sculpted solid.
func boundingCube(surfaces []*topo.Body, feat string) (*topo.Body, error) {
	pts := make([]math.Point3, 0, 2*len(surfaces))
	for _, s := range surfaces {
		bb := s.RangeBox()
		pts = append(pts, bb.Min, bb.Max)
	}
	box := math.BoxFromPoints(pts...)
	m := float64(box.Diagonal().Length())*0.1 + 1
	mv := math.V3(m, m, m)
	return brep.SolidBlock(box.Min.TranslateBy(mv.Scale(-1)), box.Max.TranslateBy(mv), feat)
}
