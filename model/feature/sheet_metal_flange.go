// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// Sheet-metal Flange feature (M13-F02). A flange adds a wall on a straight edge of an
// existing sheet, folded up over a cylindrical bend. The wall and its bend are built as one
// solid — the bend+wall CROSS-SECTION (a constant-thickness band: a faceted inner arc of the
// bend radius, an outer arc of radius+thickness, then a straight run for the flange height)
// extruded along the picked edge — then unioned onto the sheet. Thickness and the default
// bend radius come from the active rule (read live from the part's parameters), so a gauge or
// radius edit repropagates here like every other wall.

// flangeBendParamName is the rule's default bend-radius parameter (compdef seeds it). Dup'd
// here so the feature engine reads it without importing compdef.
const flangeBendParamName = "BendRadius"

// flangeFacetStep caps the bend arc's facet size (~10°) — the same smoothness/face-count
// trade the part bend uses.
const flangeFacetStep = stdmath.Pi / 18

// SheetMetalFlangeDefinition is the flange recipe: the edge to flange from, the flange height
// and bend angle (parameter-backed closures), an optional bend-radius override (nil ⇒ the
// rule's BendRadius), and a flip that folds to the opposite side of the sheet.
type SheetMetalFlangeDefinition struct {
	EdgeKey []byte
	Height  func() float64
	Angle   func() float64 // bend angle (radians); nil ⇒ 90°
	Radius  func() float64 // inside bend radius; nil ⇒ rule BendRadius
	Flip    bool
	// Width is how much of the picked edge the wall covers (#1958); the zero value is the whole
	// edge, which is what this feature has always built.
	Width FlangeWidth
	// Options overrides the style's bend properties for this bend alone (#1959); nil ⇒ the style.
	Options *BendOptions
	// AutoMiter extends this wall and the one it corners with until they meet, then cuts MiterGap
	// between them (#1961). Off by default: an existing part's corners stay as they were built.
	AutoMiter bool
	MiterGap  func() float64
	// Position and HeightDatum decide where the wall LANDS (#1957): how far back from the picked
	// edge the bend sits, and what the height is measured from. Both default to what this feature
	// has always built — the bend at the edge, the height from its tangent. See
	// sheet_metal_flange_position.go.
	Position       BendPosition
	PositionOffset func() float64 // explicit distance for the two edge-offset positions
	HeightDatum    HeightDatum
	// EdgeSets flanges SEVERAL edges in one feature, each set with its own edges and Width —
	// Inventor's FlangeDefinition edge-set collection (#2071). When non-empty it SUPERSEDES
	// EdgeKey/Width; every other option (height, angle, radius, flip, position, datum, options,
	// miter) is shared across the sets. The zero value is the single-edge flange this feature has
	// always built.
	EdgeSets []FlangeEdgeSet
}

// FlangeEdgeSet is one edge group of a multi-edge flange (#2071): the edges to flange and the width
// extent bounding their walls (the zero Width spans each whole edge).
type FlangeEdgeSet struct {
	EdgeKeys [][]byte
	Width    FlangeWidth
}

// SheetMetalFlangeFeature folds a wall onto the sheet each recompute.
type SheetMetalFlangeFeature struct {
	def        *SheetMetalFlangeDefinition
	featName   string
	placement  *BendPlacement  // the FIRST bend, for the single-placement PlacedBend interface
	placements []BendPlacement // every bend this recompute placed (one per flanged edge), for the flat pattern
}

// Definition returns the flange recipe.
func (f *SheetMetalFlangeFeature) Definition() *SheetMetalFlangeDefinition { return f.def }

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalFlangeFeature) Kind() string { return "sheet-metal-flange" }

// Recompute resolves the edge, builds the bend+wall solid at the rule gauge, and unions it
// onto the running sheet.
func (f *SheetMetalFlangeFeature) Recompute(in Input) (Output, error) {
	body, err := lastBody(in, "sheet-metal flange")
	if err != nil {
		return Output{}, err
	}
	dims, err := f.resolveDims(in.Params)
	if err != nil {
		return Output{}, err
	}
	sets := f.edgeSets()
	if len(sets) == 0 {
		return Output{}, errors.New("sheet-metal flange: no edge to flange")
	}
	bodies, heals, err := f.flangeAllEdges(in, body, dims, sets)
	if err != nil {
		return Output{}, err
	}
	f.placement = &f.placements[0] // the first bend, for the single-placement PlacedBend interface
	return Output{Bodies: bodies, Heals: heals}, nil
}

// flangeAllEdges folds a wall on every edge of every set onto the running sheet, chaining each
// result into the next, and returns the final bodies plus the reference heals (#2071). It records
// one bend placement per edge as it goes (f.placements).
func (f *SheetMetalFlangeFeature) flangeAllEdges(in Input, body *topo.Body, dims flangeDims, sets []FlangeEdgeSet) ([]*topo.Body, []ReferenceHeal, error) {
	run := in
	f.placements = nil
	var heals []ReferenceHeal
	for _, set := range sets {
		edges, h, err := resolveEdges(body, set.EdgeKeys, nil)
		if err != nil {
			return nil, nil, err
		}
		heals = append(heals, h...)
		for _, edge := range edges {
			if run.Bodies, err = f.flangeOneEdge(run, edge, dims, set.Width); err != nil {
				return nil, nil, err
			}
		}
	}
	return run.Bodies, heals, nil
}

// flangeOneEdge folds one wall onto the running bodies from a single edge with the given width, cuts
// its reliefs, and records its bend placement. It returns the new running body state so the caller
// can chain several edges (and edge sets) through one feature (#2071).
func (f *SheetMetalFlangeFeature) flangeOneEdge(run Input, edge *topo.Edge, dims flangeDims, width FlangeWidth) ([]*topo.Body, error) {
	wall, placement, err := buildFoldedSolidAt(edge, dims.thickness, f.flangeSteps(dims),
		dims.setback, width, f.def.Flip, f.featName)
	if err != nil {
		return nil, err
	}
	// Union the wall onto the sheet (combine planarizes both before the planar boolean).
	bodies, err := combine(run, wall, ops.Join)
	if err != nil {
		return nil, err
	}
	if bodies, err = f.relieve(bodies, edge, dims, placement, width, run); err != nil {
		return nil, err
	}
	f.placements = append(f.placements, placement) // record the bend for the flat pattern (M13-F04)
	return bodies, nil
}

// edgeSets returns the flange's edge sets — the explicit EdgeSets, or the single EdgeKey/Width set
// this feature has always built (#2071).
func (f *SheetMetalFlangeFeature) edgeSets() []FlangeEdgeSet {
	if len(f.def.EdgeSets) > 0 {
		return f.def.EdgeSets
	}
	if len(f.def.EdgeKey) == 0 {
		return nil
	}
	return []FlangeEdgeSet{{EdgeKeys: [][]byte{f.def.EdgeKey}, Width: f.def.Width}}
}

// relieve cuts both styled reliefs this wall calls for (#2072): the notches at its own bend's ends,
// and the corner cut where that bend meets one an earlier wall placed.
func (f *SheetMetalFlangeFeature) relieve(bodies []*topo.Body, edge *topo.Edge, dims flangeDims,
	placement BendPlacement, width FlangeWidth, in Input) ([]*topo.Body, error) {
	bodies, err := f.relieveBend(bodies, edge, dims, width, in.Relief)
	if err != nil {
		return nil, err
	}
	if bodies, err = f.mitre(bodies, placement, in); err != nil {
		return nil, err
	}
	return cutCornerRelief(bodies, placement, in, f.bendTransition(in), featOr(f.featName, "flange"))
}

// mitre fills the corner between this wall and the one it meets, when the flange asks for it
// (#1961). The gap defaults to the style's, which is what the sheet-metal rule's GapSize carries.
func (f *SheetMetalFlangeFeature) mitre(bodies []*topo.Body, placement BendPlacement, in Input) ([]*topo.Body, error) {
	if !f.def.AutoMiter {
		return bodies, nil
	}
	gap := in.MiterGap
	if f.def.MiterGap != nil {
		gap = f.def.MiterGap()
	}
	return mitreCorner(bodies, placement, in, gap, featOr(f.featName, "flange"))
}

// bendTransition is the transition this bend uses: its own override, or the style's when it defers.
func (f *SheetMetalFlangeFeature) bendTransition(in Input) types.BendTransition {
	if f.def.Options != nil && f.def.Options.Transition != types.DefaultBendTransition {
		return f.def.Options.Transition
	}
	return in.Transition
}

// relieveBend cuts the styled relief notches at the ends of this flange's bend that stop short of
// the edge (#2072). A full-width flange reaches both ends of its edge and needs none, which is why
// nothing was relieved before width extents existed (#1958).
func (f *SheetMetalFlangeFeature) relieveBend(bodies []*topo.Body, edge *topo.Edge, dims flangeDims,
	width FlangeWidth, spec ReliefSpec) ([]*topo.Body, error) {
	v0, v1 := edge.StartVertex().Point(), edge.EndVertex().Point()
	from, to, err := width.span(float64(v0.DistanceTo(v1)))
	if err != nil {
		return nil, err
	}
	length := float64(v0.DistanceTo(v1))
	ends := bendReliefEnds(from, to, length)
	return cutBendRelief(bodies, edge, ends, f.def.Options.resolve(spec), dims.thickness, length,
		f.def.Flip, featOr(f.featName, "flange"))
}

// Placement returns the FIRST resolved bend from the last successful recompute (the single-bend
// PlacedBend surface); the flat pattern reads every bend through Placements. ok is false before the
// first recompute.
func (f *SheetMetalFlangeFeature) Placement() (BendPlacement, bool) {
	if f.placement == nil {
		return BendPlacement{}, false
	}
	return *f.placement, true
}

// Placements returns every bend this feature placed — one per flanged edge (#2071) — for the flat
// pattern to lay each out as a tab. A single-edge flange returns one, matching Placement.
func (f *SheetMetalFlangeFeature) Placements() []BendPlacement { return f.placements }

// flangeDims is the resolved, validated set of flange dimensions for one recompute. run is the
// straight wall length after the bend, which is the height once the datum's setback is taken off;
// setback is how far back from the picked edge the section starts (#1957).
type flangeDims struct{ thickness, radius, height, angle, run, setback float64 }

// flangeSteps is the folded section a flange builds: its bend and the wall that follows. Where the
// section STARTS is the bend position's business (d.setback), not a step.
func (f *SheetMetalFlangeFeature) flangeSteps(d flangeDims) []bendRun {
	return []bendRun{{Angle: d.angle, Radius: d.radius, Run: d.run}}
}

// resolveDims reads the live thickness, the bend radius (override or rule), the height, and
// the angle, erroring if any is non-positive.
func (f *SheetMetalFlangeFeature) resolveDims(ps *param.Parameters) (flangeDims, error) {
	t, err := sheetThickness(ps)
	if err != nil {
		return flangeDims{}, err
	}
	d := flangeDims{thickness: t, radius: f.resolveRadius(ps), height: evalFloat(f.def.Height), angle: f.resolveAngle()}
	if d.radius <= 0 || d.height <= 0 || d.angle <= 0 {
		return flangeDims{}, fmt.Errorf("sheet-metal flange: radius/height/angle must be positive (r=%g h=%g a=%g)", d.radius, d.height, d.angle)
	}
	if d.run, err = f.def.HeightDatum.wallRun(d.height, d.radius, d.thickness, d.angle); err != nil {
		return flangeDims{}, err
	}
	d.setback = f.def.Position.setbackFor(d.radius, d.thickness, evalFloat(f.def.PositionOffset))
	return d, nil
}

// resolveRadius returns the bend radius: the override closure, else the rule's BendRadius
// parameter, else 0 (which Recompute rejects).
func (f *SheetMetalFlangeFeature) resolveRadius(ps *param.Parameters) float64 {
	if f.def.Radius != nil {
		return f.def.Radius()
	}
	if ps != nil {
		if p, ok := ps.ByName(flangeBendParamName); ok {
			return p.ModelValue()
		}
	}
	return 0
}

// resolveAngle returns the bend angle, defaulting to a 90° fold.
func (f *SheetMetalFlangeFeature) resolveAngle() float64 {
	if f.def.Angle == nil {
		return stdmath.Pi / 2
	}
	return f.def.Angle()
}

// BendSpecs reports the single bend a flange introduces (its fold), for the flat pattern.
// A nil radius override defers to the rule's default (signalled by a non-positive radius).
func (f *SheetMetalFlangeFeature) BendSpecs(_ float64) []BendSpec {
	radius := 0.0
	if f.def.Radius != nil {
		radius = f.def.Radius()
	}
	// One bend per flanged edge (#2071): a multi-edge flange develops each edge's fold. The
	// dimensions are shared, so every spec is identical — only the count follows the edges.
	n := 0
	for _, set := range f.edgeSets() {
		n += len(set.EdgeKeys)
	}
	if n == 0 {
		n = 1 // a definition without a resolved edge still reports its one fold
	}
	specs := make([]BendSpec, n)
	for i := range specs {
		specs[i] = BendSpec{Angle: f.resolveAngle(), Radius: radius}
	}
	return specs
}

// buildFoldedSolid extrudes a folded-section chain (see sheet_metal_band.go) along the picked
// edge — the shared body of the flange and every hem type (#1956). The reported placement
// describes the FIRST bend, which is the one at the picked edge and so the one the flat pattern
// unfolds about; the developed length of any further folds is the caller's BendSpecs to report.
func buildFoldedSolid(edge *topo.Edge, thickness float64, steps []bendRun, flip bool,
	feat string) (*topo.Body, BendPlacement, error) {
	return buildFoldedSolidAt(edge, thickness, steps, 0, FlangeWidth{}, flip, feat)
}

// buildFoldedSolidAt is buildFoldedSolid with the section SHIFTED setback into the parent material
// — how a flange sits back from its picked edge (#1957). The shifted section starts buried in the
// sheet and the union absorbs the overlap; a backwards RUN cannot do this, because the bend that
// follows it curls forward again and doubles the band over itself.
func buildFoldedSolidAt(edge *topo.Edge, thickness float64, steps []bendRun, setback float64,
	width FlangeWidth, flip bool, feat string) (*topo.Body, BendPlacement, error) {
	if len(steps) == 0 {
		return nil, BendPlacement{}, fmt.Errorf("sheet-metal %s: no bend steps to build", feat)
	}
	v0, v1 := edge.StartVertex().Point(), edge.EndVertex().Point()
	e, err := math.UnitVector3FromVector(v0.VectorTo(v1))
	if err != nil {
		return nil, BendPlacement{}, fmt.Errorf("sheet-metal flange: degenerate edge")
	}
	up, out, err := flangeFrame(edge, v0.Midpoint(v1), e, flip)
	if err != nil {
		return nil, BendPlacement{}, err
	}
	from, to, err := width.span(float64(v0.DistanceTo(v1)))
	if err != nil {
		return nil, BendPlacement{}, err
	}
	plane := planePerp(v0, e)
	poly := foldedSection(steps, out.AsVector(), up.AsVector(), thickness, setback, plane)
	// The bend LINE is the covered sub-span, not the whole edge, so the flat pattern develops a
	// partial wall as the partial tab it is (#1958).
	along := e.AsVector()
	placement := BendPlacement{
		AxisStart: v0.TranslateBy(along.Scale(from)), AxisEnd: v0.TranslateBy(along.Scale(to)),
		Outward: out, Up: up,
		Angle: steps[0].Angle, Radius: steps[0].Radius, Thickness: thickness,
		Length: steps[0].Run, FoldDown: flip,
	}
	return buildPrism(poly, plane, span{near: from, far: to}, 0, feat), placement, nil
}

// foldedSection projects the folded chain into the edge's section plane, shifted setback into the
// parent material (#1957).
func foldedSection(steps []bendRun, out, up math.Vector3, thickness, setback float64,
	plane sketch.Plane) []math.Point2 {
	u, v := plane.XAxis().AsVector(), plane.YAxis().AsVector()
	return bandPolygon(steps, out, up, thickness, func(w math.Vector3) math.Point2 {
		shifted := w.Add(out.Scale(-setback))
		return math.P2(shifted.Dot(u), shifted.Dot(v))
	})
}

// flangeFrame returns the parent face's outward normal (up) and the in-plane outward
// direction (out, away from the sheet) for the edge. The parent face is the larger of the
// two faces the edge bounds (the wide sheet face, not the thin side). flip negates up.
func flangeFrame(edge *topo.Edge, mid math.Point3, e math.UnitVector3, flip bool) (up, out math.UnitVector3, err error) {
	faces := edge.Faces()
	if len(faces) != 2 {
		return up, out, fmt.Errorf("sheet-metal flange: edge bounds %d faces, need 2", len(faces))
	}
	parent := widerFace(faces[0], faces[1])
	pts := faceVertexPoints(parent)
	n := faceNormal(parent, pts)
	if parent.Reversed() {
		n = n.Negate()
	}
	outward := interiorDir(parent, mid, e) // points INTO the sheet
	o, err := math.UnitVector3FromVector(outward.Negate())
	if err != nil {
		return up, out, fmt.Errorf("sheet-metal flange: cannot orient against the edge")
	}
	if flip {
		n = n.Negate()
	}
	return n, o, nil
}

// widerFace returns the face with the larger boundary area (the wide sheet face vs the thin
// edge side).
func widerFace(a, b *topo.Face) *topo.Face {
	if polygonArea3(faceVertexPoints(a)) >= polygonArea3(faceVertexPoints(b)) {
		return a
	}
	return b
}

// polygonArea3 returns the area of a planar polygon in 3D (Newell's method).
func polygonArea3(pts []math.Point3) float64 {
	if len(pts) < 3 {
		return 0
	}
	var n math.Vector3
	for i := range pts {
		a, b := pts[i], pts[(i+1)%len(pts)]
		n = n.Add(math.V3(
			(a.Y-b.Y)*(a.Z+b.Z),
			(a.Z-b.Z)*(a.X+b.X),
			(a.X-b.X)*(a.Y+b.Y),
		))
	}
	return n.Length() / 2
}

// SheetMetalFlangeFeatures adds flange features into the engine.
type SheetMetalFlangeFeatures struct{ engine *PartFeatures }

// NewSheetMetalFlangeFeatures binds the collection to a feature engine.
func NewSheetMetalFlangeFeatures(engine *PartFeatures) *SheetMetalFlangeFeatures {
	return &SheetMetalFlangeFeatures{engine}
}

// Add appends a flange feature, naming it Flange1, Flange2, … .
func (c *SheetMetalFlangeFeatures) Add(def *SheetMetalFlangeDefinition) *PartFeature {
	f := &SheetMetalFlangeFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Flange"))
	f.featName = pf.Name()
	return pf
}

var (
	_ Feature     = (*SheetMetalFlangeFeature)(nil)
	_ PlacedBend  = (*SheetMetalFlangeFeature)(nil)
	_ PlacedBends = (*SheetMetalFlangeFeature)(nil) // #2071: a multi-edge flange places several bends
	_ BendLineage = (*SheetMetalFlangeFeature)(nil)
)
