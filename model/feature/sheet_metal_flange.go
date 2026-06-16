// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
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
	edges, err := resolveEdges(body, [][]byte{f.def.EdgeKey})
	if err != nil {
		return Output{}, err
	}
	wall, placement, err := buildFlangeSolid(edges[0], dims.thickness, dims.radius, dims.height, dims.angle, f.def.Flip, f.featName)
	if err != nil {
		return Output{}, err
	}
	// Union the wall onto the sheet (combine planarizes both before the planar boolean).
	bodies, err := combine(in.Bodies, wall, ops.Join)
	if err != nil {
		return Output{}, err
	}
	f.placement = &placement // record the resolved bend for the flat pattern (M13-F04)
	return Output{Bodies: bodies}, nil
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

// flangeDims is the resolved, validated set of flange dimensions for one recompute.
type flangeDims struct{ thickness, radius, height, angle float64 }

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

// buildFlangeSolid constructs the bend+wall solid on edge: the cross-section band extruded
// along the edge. up is the parent face's outward normal (the fold-toward side); out is the
// in-plane direction away from the sheet. flip folds toward the opposite face. It also
// returns the resolved [BendPlacement] (the bend line + outward direction + dims) so the
// flat pattern can lay this flange out as a tab without re-resolving the edge.
func buildFlangeSolid(edge *topo.Edge, thickness, radius, height, angle float64, flip bool, feat string) (*topo.Body, BendPlacement, error) {
	v0, v1 := edge.StartVertex().Point(), edge.EndVertex().Point()
	e, err := math.UnitVector3FromVector(v0.VectorTo(v1))
	if err != nil {
		return nil, BendPlacement{}, fmt.Errorf("sheet-metal flange: degenerate edge")
	}
	up, out, err := flangeFrame(edge, v0.Midpoint(v1), e, flip)
	if err != nil {
		return nil, BendPlacement{}, err
	}
	plane := planePerp(v0, e)
	u, v := plane.XAxis().AsVector(), plane.YAxis().AsVector()
	proj := func(w math.Vector3) math.Point2 { return math.P2(w.Dot(u), w.Dot(v)) }
	poly := flangeBandPolygon(out.AsVector(), up.AsVector(), thickness, radius, height, angle, proj)
	placement := BendPlacement{
		AxisStart: v0, AxisEnd: v1, Outward: out,
		Angle: angle, Radius: radius, Thickness: thickness, Length: height, FoldDown: flip,
	}
	return buildPrism(poly, plane, span{near: 0, far: v0.DistanceTo(v1)}, 0, feat), placement, nil
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

// flangeBandPolygon returns the bend+wall cross-section as a closed 2D polygon in the section
// plane (projected via proj). The inner arc has radius r about a centre offset up·r from the
// edge; the outer arc has radius r+t; after the bend the wall runs straight for height h. The
// loop is ordered inner-arc → inner-wall-end → outer-wall-end → outer-arc(reversed).
func flangeBandPolygon(out, up math.Vector3, t, r, h, angle float64, proj func(math.Vector3) math.Point2) []math.Point2 {
	centre := up.Scale(r) // bend-axis centre, relative to the edge origin
	dir := func(phi float64) math.Vector3 {
		return up.Scale(-stdmath.Cos(phi)).Add(out.Scale(stdmath.Sin(phi)))
	}
	steps := int(stdmath.Max(2, stdmath.Round(angle/flangeFacetStep)))
	inner := make([]math.Point2, 0, steps+1)
	outer := make([]math.Point2, 0, steps+1)
	for k := 0; k <= steps; k++ {
		phi := angle * float64(k) / float64(steps)
		d := dir(phi)
		inner = append(inner, proj(centre.Add(d.Scale(r))))
		outer = append(outer, proj(centre.Add(d.Scale(r+t))))
	}
	wall := out.Scale(stdmath.Cos(angle)).Add(up.Scale(stdmath.Sin(angle))).Scale(h) // straight-run offset
	innerEnd := proj(centre.Add(dir(angle).Scale(r)).Add(wall))
	outerEnd := proj(centre.Add(dir(angle).Scale(r + t)).Add(wall))

	poly := append([]math.Point2(nil), inner...) // inner arc, edge → bend end
	poly = append(poly, innerEnd, outerEnd)      // across the wall's far end
	for k := len(outer) - 1; k >= 0; k-- {       // outer arc back, bend end → edge
		poly = append(poly, outer[k])
	}
	return poly
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
