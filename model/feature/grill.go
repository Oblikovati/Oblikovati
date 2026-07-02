// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Grill (M20-F10 #863) is a plastic ventilation feature: a boundary region of a thin wall is
// cut open and bridged by a structure of ribs, spars and islands that stay solid (the reference
// GrillFeature). The structure is drawn as the inner loops of the boundary profile(s): a sketch
// rectangle (the vent) with rib/spar/island shapes inside it resolves to one profile whose
// outer loop is the boundary and whose holes are the kept structure. Cutting that profile
// through the wall removes the open area and leaves the structure bridging the vent.
//
// Geometrically every kept element (rib/spar/island) is a hole of the cut profile; they share
// one cut depth. A draft angle tapers the vent walls.

// GrillDefinition is the grill recipe: the boundary vent profile(s) (whose inner loops are the
// bridging structure), in one sketch, and a draft angle on the cut walls.
type GrillDefinition struct {
	Sketch     *sketch.Sketch
	Boundaries []int
	Draft      float64
}

// GrillFeature cuts the vent(s), leaving each boundary profile's inner-loop structure bridging.
type GrillFeature struct {
	def      *GrillDefinition
	featName string
}

// Definition returns the grill recipe.
func (g *GrillFeature) Definition() *GrillDefinition { return g.def }

// Kind implements [Feature].
func (g *GrillFeature) Kind() string { return "grill" }

// Recompute cuts each boundary's vent through the running wall, leaving the rib/spar/island
// structure bridging it. The vent of a boundary is boundary − union(bars), where the bars are
// the sketch's closed loops lying inside that boundary; computing it as a boundary solid drilled
// by each bar (a sequence of booleans = union subtraction) is robust even when the bars cross —
// unlike the boundary profile's inner loops, which the even–odd region finder mis-forms for
// overlapping structure (#863).
func (g *GrillFeature) Recompute(in Input) (Output, error) {
	if _, err := lastBody(in, "grill"); err != nil {
		return Output{}, err
	}
	if len(g.def.Boundaries) == 0 {
		return Output{}, fmt.Errorf("grill: no boundary profile selected")
	}
	boundaries, err := resolveClosedProfiles(g.def.Sketch, g.def.Boundaries, "grill boundary")
	if err != nil {
		return Output{}, err
	}
	plane := g.def.Sketch.Plane()
	sp := throughSpan(in.Bodies, plane)
	loops := g.def.Sketch.ClosedLoops()
	cutTool := ventTool(boundaries, loops, plane, sp, g.def.Draft, featOr(g.featName, "grill"), in.Diag)
	bodies, err := combine(in, cutTool, ops.Cut)
	if err != nil {
		return Output{}, fmt.Errorf("grill: %w", err)
	}
	return Output{Bodies: bodies}, nil
}

// ventTool builds the merged cut tool: for each boundary, a solid prism of its outer loop drilled
// by every closed loop (bar) lying inside it, giving boundary − union(bars). rec collects the
// bar-drilling booleans' fallback diagnostics (#1601; nil discards).
func ventTool(boundaries []*sketch.Profile, loops []sketch.Loop, plane sketch.Plane, sp span, draft float64, feat string, rec *diag.Recorder) *topo.Body {
	tools := make([]*topo.Body, 0, len(boundaries))
	for i, b := range boundaries {
		name := feat
		if len(boundaries) > 1 {
			name = fmt.Sprintf("%s/b%d", feat, i)
		}
		outer := b.OuterLoop().Polygon()
		solid := buildPrism(outer, plane, sp, draft, name)
		if bars := loopsInside(loops, outer); len(bars) > 0 {
			solid = drillProfileHoles(solid, bars, plane, sp, draft, name, rec)
		}
		tools = append(tools, solid)
	}
	if len(tools) == 1 {
		return tools[0]
	}
	return topo.MergeBodies(topo.NewLineage(topo.Tok(feat, "merged", 0)), true, tools...)
}

// loopsInside returns the loops lying strictly inside the boundary polygon (the bars) — excluding
// the boundary loop itself and anything outside it.
func loopsInside(loops []sketch.Loop, boundary []math.Point2) []sketch.Loop {
	var inside []sketch.Loop
	for _, l := range loops {
		if loopStrictlyInside(l.Polygon(), boundary) {
			inside = append(inside, l)
		}
	}
	return inside
}

// loopStrictlyInside reports whether every vertex of poly lies inside boundary (so the boundary
// loop, whose vertices lie on its own edge, is excluded).
func loopStrictlyInside(poly, boundary []math.Point2) bool {
	if len(poly) == 0 {
		return false
	}
	for _, v := range poly {
		if !pointInPolygon2D(v, boundary) {
			return false
		}
	}
	return true
}

// pointInPolygon2D is the even–odd ray-cast test.
func pointInPolygon2D(p math.Point2, poly []math.Point2) bool {
	in := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		yi, yj := poly[i].Y, poly[j].Y
		if (yi > p.Y) != (yj > p.Y) {
			if p.X < poly[i].X+(p.Y-yi)/(yj-yi)*(poly[j].X-poly[i].X) {
				in = !in
			}
		}
	}
	return in
}

// throughSpan spans the whole running material along the sketch-plane normal (plus a margin on
// each side) so the vent cut passes cleanly through the wall whichever side the sketch sits on.
func throughSpan(bodies []*topo.Body, plane sketch.Plane) span {
	lo, hi := normalExtent(bodies, plane)
	return span{near: lo - throughAllMargin, far: hi + throughAllMargin}
}

// GrillFeatures adds grill features into the engine.
type GrillFeatures struct{ engine *PartFeatures }

// NewGrillFeatures binds the collection to a feature engine.
func NewGrillFeatures(engine *PartFeatures) *GrillFeatures { return &GrillFeatures{engine: engine} }

// Add creates a grill from its definition.
func (c *GrillFeatures) Add(def *GrillDefinition) *PartFeature {
	g := &GrillFeature{def: def}
	pf := c.engine.Add(g)
	pf.SetName(c.engine.UniqueName("Grill"))
	g.featName = pf.name
	return pf
}
