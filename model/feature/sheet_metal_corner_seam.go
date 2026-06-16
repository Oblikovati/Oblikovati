// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Sheet-metal Corner Seam feature (M13-F02). Where two flange walls meet at a corner, the
// seam controls the relief between them. The gap seam cuts a square notch of the given gap
// along the shared corner edge — relieving the corner so the two walls leave a controlled
// gap (no interference, room for welding/fastening). The cut removes a parallelogram
// cross-section (gap along each adjacent face) swept along the edge, distinct from the corner
// CHAMFER's triangular bevel. Overlap/edge seam types build on this and are a follow-up.

// SeamType discriminates the corner-seam relief.
type SeamType int

const (
	// GapSeam cuts a square relief leaving a gap between the two corner walls (the default).
	GapSeam SeamType = iota
)

// ParseSeamType resolves a wire spelling to a seam type.
func ParseSeamType(s string) (SeamType, bool) {
	switch s {
	case "", "gap":
		return GapSeam, true
	}
	return 0, false
}

// SheetMetalCornerSeamDefinition is the corner-seam recipe: the corner edges (the shared
// through-thickness edges where flanges meet), the gap, and the seam type.
type SheetMetalCornerSeamDefinition struct {
	EdgeKeys [][]byte
	Gap      func() float64
	Type     SeamType
}

// SheetMetalCornerSeamFeature relieves the sheet's corner seams each recompute.
type SheetMetalCornerSeamFeature struct {
	def      *SheetMetalCornerSeamDefinition
	featName string
}

// Definition returns the corner-seam recipe.
func (f *SheetMetalCornerSeamFeature) Definition() *SheetMetalCornerSeamDefinition {
	return f.def
}

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalCornerSeamFeature) Kind() string { return "sheet-metal-corner-seam" }

// Recompute resolves the corner edges and cuts a gap relief at each.
func (f *SheetMetalCornerSeamFeature) Recompute(in Input) (Output, error) {
	body, err := lastBody(in, "sheet-metal corner seam")
	if err != nil {
		return Output{}, err
	}
	gap := evalFloat(f.def.Gap)
	if gap <= 0 {
		return Output{}, fmt.Errorf("sheet-metal corner seam: gap must be positive, got %g", gap)
	}
	if len(f.def.EdgeKeys) == 0 {
		return Output{}, fmt.Errorf("sheet-metal corner seam: no corner edges selected")
	}
	edges, err := resolveEdges(body, f.def.EdgeKeys)
	if err != nil {
		return Output{}, err
	}
	work, edges := planarizeForEdges(body, edges, f.featName)
	tools, err := seamCutters(edges, gap, f.featName)
	if err != nil {
		return Output{}, err
	}
	res, err := cutAll(work, tools)
	if err != nil {
		return Output{}, fmt.Errorf("sheet-metal corner seam: %w", err)
	}
	return Output{Bodies: replaceBody(in.Bodies, body, res)}, nil
}

// seamCutters builds the gap-relief notch cutter for each corner edge.
func seamCutters(edges []*topo.Edge, gap float64, feat string) ([]*topo.Body, error) {
	tools := make([]*topo.Body, 0, len(edges))
	for i, edge := range edges {
		tool, err := seamCutter(edge, gap, fmt.Sprintf("%s/s%d", feat, i))
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// seamCutter builds the square relief notch removed at a corner edge: a parallelogram of gap
// along each adjacent face's interior, swept along the edge with a small overhang for a clean
// boolean.
func seamCutter(edge *topo.Edge, gap float64, feat string) (*topo.Body, error) {
	fr, err := edgeCornerFrame(edge, "sheet-metal corner seam")
	if err != nil {
		return nil, err
	}
	// A parallelogram corner notch: origin → gap·t1 → gap·(t1+t2) → gap·t2 (vs the chamfer's
	// triangle), so the relief is a square gap rather than a bevel.
	poly := []math.Point2{
		{X: 0, Y: 0},
		fr.proj(fr.t1.Scale(gap)),
		fr.proj(fr.t1.Add(fr.t2).Scale(gap)),
		fr.proj(fr.t2.Scale(gap)),
	}
	oh := gap * 0.1
	return buildPrism(poly, fr.plane, span{near: -oh, far: fr.length + oh}, 0, feat), nil
}

// SheetMetalCornerSeamFeatures adds corner-seam features into the engine.
type SheetMetalCornerSeamFeatures struct{ engine *PartFeatures }

// NewSheetMetalCornerSeamFeatures binds the collection to a feature engine.
func NewSheetMetalCornerSeamFeatures(engine *PartFeatures) *SheetMetalCornerSeamFeatures {
	return &SheetMetalCornerSeamFeatures{engine}
}

// Add appends a corner-seam feature, naming it CornerSeam1, … .
func (c *SheetMetalCornerSeamFeatures) Add(def *SheetMetalCornerSeamDefinition) *PartFeature {
	f := &SheetMetalCornerSeamFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("CornerSeam"))
	f.featName = pf.Name()
	return pf
}

var _ Feature = (*SheetMetalCornerSeamFeature)(nil)
