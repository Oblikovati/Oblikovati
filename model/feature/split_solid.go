// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// SplitSide selects which side(s) of the cutting plane a solid split keeps: both pieces (a true
// split into separate bodies), or only the +normal / −normal side (a Trim Solid).
type SplitSide uint8

const (
	SplitBoth     SplitSide = iota // keep both pieces (split into bodies)
	SplitPositive                  // keep the side the plane normal points toward (trim)
	SplitNegative                  // keep the opposite side (trim)
)

// SplitSolidDefinition is the recipe for a solid split: a cutting work plane and which side(s)
// to keep, re-resolved each recompute.
type SplitSolidDefinition struct {
	Plane *WorkPlane
	Keep  SplitSide
}

// SplitSolidFeature divides the running solid bodies by a plane (Inventor's Split Solid / Trim
// Solid): each solid is cut into the pieces on each side of the plane, and Keep filters them.
type SplitSolidFeature struct{ def *SplitSolidDefinition }

// Definition returns the split recipe.
func (f *SplitSolidFeature) Definition() *SplitSolidDefinition { return f.def }

// Kind implements [Feature].
func (f *SplitSolidFeature) Kind() string { return "splitSolid" }

// Recompute resolves the cutting plane and splits every running solid body by it, keeping the
// requested side(s). A lost plane → Sick; surface (non-solid) bodies pass through unchanged.
func (f *SplitSolidFeature) Recompute(in Input) (Output, error) {
	if f.def.Plane == nil {
		return Output{}, errors.New("split: no cutting plane")
	}
	plane, err := geomPlaneOf(f.def.Plane)
	if err != nil {
		return Output{}, err
	}
	var out []*topo.Body
	for _, b := range in.Bodies {
		if !b.IsSolid() {
			out = append(out, b)
			continue
		}
		pieces, err := ops.SplitSolidByPlane(b, plane)
		if err != nil {
			return Output{}, err
		}
		out = append(out, keepSplitSides(pieces, plane, f.def.Keep)...)
	}
	return Output{Bodies: out}, nil
}

// geomPlaneOf converts a work plane's sketch plane to a kernel plane for the split.
func geomPlaneOf(wp *WorkPlane) (geom.Plane, error) {
	sp := wp.Plane()
	gp, err := geom.NewPlane(sp.Origin(), sp.Normal().AsVector())
	if err != nil {
		return geom.Plane{}, fmt.Errorf("split: cutting plane is degenerate: %w", err)
	}
	return gp, nil
}

// keepSplitSides filters the pieces by which side of the plane each piece's centroid lies on
// (SplitBoth keeps all). Used to implement Trim Solid (keep one side).
func keepSplitSides(pieces []*topo.Body, plane geom.Plane, keep SplitSide) []*topo.Body {
	if keep == SplitBoth {
		return pieces
	}
	var out []*topo.Body
	for _, p := range pieces {
		c := ops.BodyGeometryProperties(p, ops.DefaultQuality()).Centroid
		side := float64(plane.Origin.VectorTo(c).Dot(plane.Normal()))
		if (keep == SplitPositive && side > 0) || (keep == SplitNegative && side < 0) {
			out = append(out, p)
		}
	}
	return out
}

// AddSplitSolid adds a feature that splits the running solids by a work plane, keeping the
// requested side(s).
func (c *ModifyFeatures) AddSplitSolid(plane *WorkPlane, keep SplitSide) *PartFeature {
	pf := c.engine.Add(&SplitSolidFeature{def: &SplitSolidDefinition{Plane: plane, Keep: keep}})
	pf.SetName(c.engine.UniqueName("Split"))
	return pf
}
