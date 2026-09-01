// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// SplitSide selects which side(s) of the cutting plane a solid split keeps: both pieces (a true
// split into separate bodies), or only the +normal / −normal side (a Trim Solid).
type SplitSide uint8

const (
	SplitBoth     SplitSide = iota // keep both pieces (split into bodies)
	SplitPositive                  // keep the side the plane normal points toward (trim)
	SplitNegative                  // keep the opposite side (trim)
)

// SplitSolidDefinition is the recipe for a solid split: the cutting tool, which side(s) to keep,
// and FacesOnly — the reference's Split Faces mode, which imprints the cut onto the body's faces
// without removing material (#330). Re-resolved each recompute.
//
// Tool names what the split cuts with (#1891): the work plane in Plane, or the surface at
// ToolIndex. See split_tool.go.
type SplitSolidDefinition struct {
	Tool      SplitToolKind
	Plane     *WorkPlane
	ToolIndex int
	Keep      SplitSide
	FacesOnly bool
	// Sketch/ProfileIndex drive the SplitByPath tool (#2068): the profile's boundary, projected onto
	// the running solids along the sketch plane normal, scores their faces (no material removed).
	// Re-resolved each recompute, so editing the sketch reshapes the split.
	Sketch       *sketch.Sketch
	ProfileIndex int
}

// SplitSolidFeature divides the running solid bodies by a plane (the reference's Split
// feature): each solid is cut into the pieces on each side of the plane and Keep filters
// them — or, FacesOnly, each solid keeps its volume and only its faces split.
type SplitSolidFeature struct{ def *SplitSolidDefinition }

// Definition returns the split recipe.
func (f *SplitSolidFeature) Definition() *SplitSolidDefinition { return f.def }

// Kind implements [Feature].
func (f *SplitSolidFeature) Kind() string { return "splitSolid" }

// SplitType reports the frozen discriminator of what this split does (api/types, #330).
func (f *SplitSolidFeature) SplitType() types.SplitType {
	switch {
	case f.def.FacesOnly:
		return types.SplitFacesSplit
	case f.def.Keep == SplitBoth:
		return types.SplitBodySplit
	default:
		return types.TrimSolidSplit
	}
}

// Recompute resolves the cutting plane and splits every running solid body by it, keeping the
// requested side(s) — or imprinting faces only. A lost plane → Sick; surface (non-solid)
// bodies pass through unchanged.
func (f *SplitSolidFeature) Recompute(in Input) (Output, error) {
	if f.def.Tool == SplitByPath {
		return f.recomputePath(in)
	}
	plane, err := f.def.cuttingPlane(in.Bodies)
	if err != nil {
		return Output{}, err
	}
	var out []*topo.Body
	for _, b := range in.Bodies {
		if !b.IsSolid() {
			out = append(out, b)
			continue
		}
		pieces, err := f.splitOne(b, plane)
		if err != nil {
			return Output{}, err
		}
		out = append(out, pieces...)
	}
	return Output{Bodies: out}, nil
}

// splitOne applies the definition's mode to one solid: a faces-only imprint, or the
// side-filtered body split.
func (f *SplitSolidFeature) splitOne(b *topo.Body, plane geom.Plane) ([]*topo.Body, error) {
	if f.def.FacesOnly {
		imprinted, err := ops.SplitFacesByPlane(b, plane)
		if err != nil {
			return nil, err
		}
		return []*topo.Body{imprinted}, nil
	}
	pieces, err := ops.SplitSolidByPlane(b, plane)
	if err != nil {
		return nil, err
	}
	return keepSplitSides(pieces, plane, f.def.Keep), nil
}

// recomputePath scores every running solid's faces along the projected sketch path (#2068), leaving
// the volume untouched. Surface (non-solid) bodies pass through unchanged.
func (f *SplitSolidFeature) recomputePath(in Input) (Output, error) {
	path, dir, err := f.def.projectedPath()
	if err != nil {
		return Output{}, err
	}
	var out []*topo.Body
	for _, b := range in.Bodies {
		if !b.IsSolid() {
			out = append(out, b)
			continue
		}
		scored, err := ops.SplitFacesByPath(b, path, dir)
		if err != nil {
			return Output{}, err
		}
		out = append(out, scored)
	}
	return Output{Bodies: out}, nil
}

// projectedPath resolves the split's profile to a closed model-space polyline plus the projection
// direction (the sketch plane normal). The profile boundary is sampled to a polygon, so a profile
// made of arcs/splines is imprinted as its faceted approximation — the plane split is exact, this is
// not (#2068). The loop is closed (first point repeated) so the extruded tool sheet has no open end.
func (d *SplitSolidDefinition) projectedPath() ([]math.Point3, math.Vector3, error) {
	if d.Sketch == nil {
		return nil, math.Vector3{}, errors.New("split: the path tool needs a sketch to project")
	}
	profs := d.Sketch.Profiles()
	if d.ProfileIndex < 0 || d.ProfileIndex >= profs.Count() {
		return nil, math.Vector3{}, fmt.Errorf("split: profile %d out of range (the sketch has %d)",
			d.ProfileIndex, profs.Count())
	}
	poly := profs.Item(d.ProfileIndex).OuterLoop().Polygon()
	if len(poly) < 2 {
		return nil, math.Vector3{}, fmt.Errorf("split: profile %d has %d points, need at least 2",
			d.ProfileIndex, len(poly))
	}
	plane := d.Sketch.Plane()
	path := make([]math.Point3, 0, len(poly)+1)
	for _, p := range poly {
		m := plane.ToModel(p)
		if len(path) > 0 && float64(path[len(path)-1].DistanceTo(m)) < 1e-9 {
			continue // drop a repeated point so no sheet quad is degenerate
		}
		path = append(path, m)
	}
	if len(path) < 2 {
		return nil, math.Vector3{}, fmt.Errorf("split: profile %d collapses to a single point", d.ProfileIndex)
	}
	if float64(path[0].DistanceTo(path[len(path)-1])) > 1e-9 {
		path = append(path, path[0]) // close the loop unless the polygon already does
	}
	return path, plane.Normal().AsVector(), nil
}

// AddSplitFacesByPath adds a faces-only split that scores the running solids along a sketch
// profile projected onto them (#2068), removing no material.
func (c *ModifyFeatures) AddSplitFacesByPath(sk *sketch.Sketch, profileIndex int) *PartFeature {
	pf := c.engine.Add(&SplitSolidFeature{def: &SplitSolidDefinition{
		Tool: SplitByPath, FacesOnly: true, Sketch: sk, ProfileIndex: profileIndex}})
	pf.SetName(c.engine.UniqueName("Split"))
	return pf
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
		c := query.BodyGeometryProperties(p, ops.DefaultQuality()).Centroid
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

// AddSplitFaces adds a faces-only split: the plane imprints onto every running solid's
// faces without removing material (#330).
func (c *ModifyFeatures) AddSplitFaces(plane *WorkPlane) *PartFeature {
	pf := c.engine.Add(&SplitSolidFeature{def: &SplitSolidDefinition{Plane: plane, FacesOnly: true}})
	pf.SetName(c.engine.UniqueName("Split"))
	return pf
}

// AddSplitByDefinition adds a split from a fully-built recipe — the path the surface tools and
// the recipe codec take, since they choose the tool rather than always passing a work plane
// (#1891).
//
//	mods.AddSplitByDefinition(&SplitSolidDefinition{Tool: SplitBySurfaceBody, ToolIndex: 1})
func (c *ModifyFeatures) AddSplitByDefinition(def *SplitSolidDefinition) *PartFeature {
	pf := c.engine.Add(&SplitSolidFeature{def: def})
	pf.SetName(c.engine.UniqueName("Split"))
	return pf
}
