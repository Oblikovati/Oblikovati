// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

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
	// Position and HeightDatum decide where the wall LANDS (#1957): how far back from the picked
	// edge the bend sits, and what the height is measured from. Both default to what this feature
	// has always built — the bend at the edge, the height from its tangent. See
	// sheet_metal_flange_position.go.
	Position       BendPosition
	PositionOffset func() float64 // explicit distance for the two edge-offset positions
	HeightDatum    HeightDatum
}

// SheetMetalFlangeFeature folds a wall onto the sheet each recompute.
type SheetMetalFlangeFeature struct {
	def       *SheetMetalFlangeDefinition
	featName  string
	placement *BendPlacement // resolved bend geometry from the last recompute (for the flat pattern)
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
	edges, heals, err := resolveEdges(body, [][]byte{f.def.EdgeKey}, nil)
	if err != nil {
		return Output{}, err
	}
	wall, placement, err := buildFoldedSolidAt(edges[0], dims.thickness, f.flangeSteps(dims),
		dims.setback, f.def.Width, f.def.Flip, f.featName)
	if err != nil {
		return Output{}, err
	}
	// Union the wall onto the sheet (combine planarizes both before the planar boolean).
	bodies, err := combine(in, wall, ops.Join)
	if err != nil {
		return Output{}, err
	}
	f.placement = &placement // record the resolved bend for the flat pattern (M13-F04)
	return Output{Bodies: bodies, Heals: heals}, nil
}

// Placement returns the resolved bend geometry captured by the last successful recompute,
// for the flat pattern to lay this flange out as a tab. ok is false before the first
// recompute.
func (f *SheetMetalFlangeFeature) Placement() (BendPlacement, bool) {
	if f.placement == nil {
		return BendPlacement{}, false
	}
	return *f.placement, true
}

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
	return []BendSpec{{Angle: f.resolveAngle(), Radius: radius}}
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

var _ Feature = (*SheetMetalFlangeFeature)(nil)
