// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Sheet-metal Corner Seam feature (M13-F02). Where two flange walls meet at a corner, the
// seam controls how they are finished — Inventor's four CornerTypeEnum styles (#1964). The GAP
// seam cuts a square notch of the given gap along the shared corner edge, relieving the corner
// so the two walls leave a controlled gap (no interference, room for welding/fastening). The cut
// removes a parallelogram cross-section (gap along each adjacent face) swept along the edge,
// distinct from the corner CHAMFER's triangular bevel.
//
// The OVERLAP / reverse-overlap / no-overlap styles lap or butt the two walls instead of gapping
// them. On this single-body sheet only the gap style changes the solid — the others differ in
// which wall laps over the other, which a watertight union cannot show — so their solid geometry
// is carried here (type, percent, relief) but modelled in a follow-up (#2085); until then a
// non-gap seam is recorded and reported, and leaves the corner unrelieved.

// SeamType discriminates the corner-seam finish. It aliases the API's CornerSeamType so the enum
// is defined once (ADR-0018); the long-standing GapSeam call sites keep working.
type SeamType = types.CornerSeamType

const (
	// GapSeam leaves a gap between the two corner walls (the default) — Inventor's ripped corner.
	GapSeam = types.CornerSeamGap
	// OverlapSeam / ReverseOverlapSeam lap one wall over the other; NoOverlapSeam butts them.
	OverlapSeam        = types.CornerSeamOverlap
	ReverseOverlapSeam = types.CornerSeamReverseOverlap
	NoOverlapSeam      = types.CornerSeamNoOverlap
)

// codeCornerSeamUnmodeled marks a non-gap seam whose lap/miter solid is not modelled yet (#2085).
const codeCornerSeamUnmodeled diag.Code = "sheet-metal.corner-seam-unmodeled"

// ParseSeamType resolves a wire spelling to a seam type — gap/overlap/reverseOverlap/noOverlap,
// with "" meaning gap. It delegates to the API enum so the vocabulary has one home.
func ParseSeamType(s string) (SeamType, bool) {
	return types.ParseCornerSeamType(s)
}

// SheetMetalCornerSeamDefinition is the corner-seam recipe: the corner edges (the shared
// through-thickness edges where flanges meet), the gap, the seam type, and — for the lap styles —
// how far one wall laps (Overlap, a percentage), the seam-root relief, and how the gap is measured.
// The overlap/relief/definition fields are recorded and round-tripped; their solid geometry is a
// follow-up (#2085), so only the gap style cuts material today.
type SheetMetalCornerSeamDefinition struct {
	EdgeKeys [][]byte
	Gap      func() float64
	Type     SeamType
	// Overlap is the lap percentage (0–100) for the overlap / reverse-overlap styles.
	Overlap float64
	// ReliefShape / ReliefSize describe the relief cut at the seam root; ReliefSize == nil ⇒ none.
	ReliefShape types.CornerReliefShape
	ReliefSize  func() float64
	// DefinitionType is how the gap is measured — max-distance (default) or face-edge distance.
	DefinitionType types.CornerSeamDefinitionType
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
	edges, heals, err := resolveEdges(body, f.def.EdgeKeys, nil)
	if err != nil {
		return Output{}, err
	}
	// The gap style cuts a notch on this one body; the butt/lap styles finish the corner where two
	// flange walls meet, so they build from the walls' bends instead (#2085) — and fall back to an
	// honest report when there is no bent corner to finish.
	if f.def.Type != GapSeam {
		return f.finishLapSeam(in, edges, gap, heals)
	}
	return f.cutGapSeam(in, body, edges, gap, heals)
}

// cutGapSeam removes the square gap notch at each resolved corner edge and returns the relieved
// body — the one seam style that changes this single-body sheet's solid.
func (f *SheetMetalCornerSeamFeature) cutGapSeam(in Input, body *topo.Body, edges []*topo.Edge,
	gap float64, heals []ReferenceHeal) (Output, error) {
	work, edges := planarizeForEdges(body, edges, f.featName)
	tools, err := seamCutters(edges, gap, f.featName)
	if err != nil {
		return Output{}, err
	}
	res, err := cutAll(work, tools, in.Diag)
	if err != nil {
		return Output{}, fmt.Errorf("sheet-metal corner seam: %w", err)
	}
	return Output{Bodies: replaceBody(in.Bodies, body, res), Heals: heals}, nil
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
