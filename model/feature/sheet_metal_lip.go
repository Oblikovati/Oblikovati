// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"
	"slices"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
)

// Sheet-metal Lip feature (M13-F03). A lip is a stiffening edge return: a short flange folded up
// over a bend, then curled 180° back on itself (a J/hem-with-stand-off profile) so the free edge
// is safe and rigid. It is built like a flange — one constant-thickness cross-section band
// extruded along the picked edge — but the band continues past the flange wall through a second
// (return) bend and a return wall. The band turns the same way through both bends, so the inner
// surface stays the inner radius and the outer stays +thickness throughout (no surface swap).

// lipFacetStep caps each arc's facet size (~10°), matching the flange/bend faceting.
const lipFacetStep = stdmath.Pi / 18

// SheetMetalLipDefinition is the lip recipe: the edge to build on, the flange wall height, the
// flange and return bend angles/radii (parameter-backed; nil ⇒ defaults), the return wall
// length, and a flip that folds to the opposite side of the sheet.
type SheetMetalLipDefinition struct {
	EdgeKey      []byte
	Height       func() float64 // flange wall height before the return
	Angle        func() float64 // flange bend angle (radians); nil ⇒ 90°
	Radius       func() float64 // inside bend radius; nil ⇒ rule BendRadius
	ReturnLength func() float64 // return wall length after the 180° curl
	Flip         bool
}

// SheetMetalLipFeature folds a stiffening lip onto the sheet each recompute.
type SheetMetalLipFeature struct {
	def      *SheetMetalLipDefinition
	featName string
}

// Definition returns the lip recipe.
func (f *SheetMetalLipFeature) Definition() *SheetMetalLipDefinition { return f.def }

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalLipFeature) Kind() string { return "sheet-metal-lip" }

// Recompute resolves the edge and dimensions, builds the lip band solid, and unions it on.
func (f *SheetMetalLipFeature) Recompute(in Input) (Output, error) {
	body, err := lastBody(in, "sheet-metal lip")
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
	wall, err := buildLipSolid(edges[0], dims, f.def.Flip, f.featName)
	if err != nil {
		return Output{}, err
	}
	bodies, err := combine(in, wall, ops.Join)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies, Heals: heals}, nil
}

// lipDims is the resolved, validated set of lip dimensions for one recompute. The return bend
// uses the same inside radius as the flange; its angle is a half turn (180°).
type lipDims struct{ thickness, radius, height, angle, returnLen float64 }

// resolveDims reads thickness (live), the bend radius (override or rule), the flange height,
// angle, and the return length, erroring if any required value is non-positive.
func (f *SheetMetalLipFeature) resolveDims(ps *param.Parameters) (lipDims, error) {
	t, err := sheetThickness(ps)
	if err != nil {
		return lipDims{}, err
	}
	d := lipDims{
		thickness: t, radius: f.resolveRadius(ps), height: evalFloat(f.def.Height),
		angle: f.resolveAngle(), returnLen: evalFloat(f.def.ReturnLength),
	}
	if d.radius <= 0 || d.height <= 0 || d.angle <= 0 || d.returnLen <= 0 {
		return lipDims{}, fmt.Errorf("sheet-metal lip: radius/height/angle/returnLength must be positive (r=%g h=%g a=%g L=%g)", d.radius, d.height, d.angle, d.returnLen)
	}
	return d, nil
}

// resolveRadius returns the bend radius: the override closure, else the rule's BendRadius
// parameter, else 0 (which Recompute rejects).
func (f *SheetMetalLipFeature) resolveRadius(ps *param.Parameters) float64 {
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

// resolveAngle returns the flange bend angle, defaulting to a 90° fold.
func (f *SheetMetalLipFeature) resolveAngle() float64 {
	if f.def.Angle == nil {
		return stdmath.Pi / 2
	}
	return f.def.Angle()
}

// buildLipSolid constructs the lip band solid on edge: the cross-section band (flange bend +
// wall + 180° return bend + return wall) extruded along the edge.
func buildLipSolid(edge *topo.Edge, d lipDims, flip bool, feat string) (*topo.Body, error) {
	v0, v1 := edge.StartVertex().Point(), edge.EndVertex().Point()
	e, err := math.UnitVector3FromVector(v0.VectorTo(v1))
	if err != nil {
		return nil, fmt.Errorf("sheet-metal lip: degenerate edge")
	}
	up, out, err := flangeFrame(edge, v0.Midpoint(v1), e, flip)
	if err != nil {
		return nil, err
	}
	plane := planePerp(v0, e)
	u, v := plane.XAxis().AsVector(), plane.YAxis().AsVector()
	proj := func(w math.Vector3) math.Point2 { return math.P2(w.Dot(u), w.Dot(v)) }
	poly := lipBandPolygon(out.AsVector(), up.AsVector(), d, proj)
	return buildPrism(poly, plane, span{near: 0, far: v0.DistanceTo(v1)}, 0, feat), nil
}

// lipTurtle walks the band's inner surface, emitting the inner and outer (inner + normal·t)
// boundary points. It tracks a position and heading in the (out, up) section plane; the outer
// side is the heading rotated −90°, so a consistent left turn keeps the inner radius inside.
type lipTurtle struct {
	pos, tang math.Vector2 // inner-surface point and unit heading, in (out, up) components
	t         float64      // band thickness
	inner     []math.Vector2
	outer     []math.Vector2
}

// emit records the current inner point and its outer offset.
func (lt *lipTurtle) emit() {
	n := math.V2(lt.tang.Y, -lt.tang.X) // heading rotated −90° ⇒ outer side
	lt.inner = append(lt.inner, lt.pos)
	lt.outer = append(lt.outer, lt.pos.Add(n.Scale(lt.t)))
}

// straight advances the turtle by len along its heading.
func (lt *lipTurtle) straight(length float64) {
	lt.emit()
	lt.pos = lt.pos.Add(lt.tang.Scale(length))
	lt.emit()
}

// rotate2 rotates a 2D vector by angle (counter-clockwise).
func rotate2(v math.Vector2, angle float64) math.Vector2 {
	c, s := stdmath.Cos(angle), stdmath.Sin(angle)
	return math.V2(v.X*c-v.Y*s, v.X*s+v.Y*c)
}

// lipBandPolygon returns the lip cross-section as a closed 2D polygon in the section plane: the
// flange bend (radius r, angle a), the flange wall (height h), the 180° return bend (radius r),
// and the return wall (length L) — then projected via proj. The loop is inner-forward then
// outer-reversed.
func lipBandPolygon(out, up math.Vector3, d lipDims, proj func(math.Vector3) math.Point2) []math.Point2 {
	lt := &lipTurtle{pos: math.V2(0, 0), tang: math.V2(1, 0), t: d.thickness} // start at the edge, heading +out
	lt.runPath(d)
	poly := make([]math.Point2, 0, len(lt.inner)+len(lt.outer))
	to3 := func(c math.Vector2) math.Point2 { return proj(out.Scale(c.X).Add(up.Scale(c.Y))) }
	for _, p := range lt.inner {
		poly = append(poly, to3(p))
	}
	for _, v := range slices.Backward(lt.outer) {
		poly = append(poly, to3(v))
	}
	return poly
}

// runPath drives the turtle through the lip's four segments (all left turns, so the band's inner
// surface stays the inner radius throughout).
func (lt *lipTurtle) runPath(d lipDims) {
	lt.arcSeg(d.radius, d.angle)    // flange bend
	lt.straight(d.height)           // flange wall
	lt.arcSeg(d.radius, stdmath.Pi) // 180° return
	lt.straight(d.returnLen)        // return wall
}

// arcSeg sweeps a left arc of inside radius r through angle, emitting faceted inner/outer points
// and advancing the turtle's position and heading to the arc's end.
func (lt *lipTurtle) arcSeg(r, angle float64) {
	centre := lt.pos.Add(math.V2(-lt.tang.Y, lt.tang.X).Scale(r))
	rel0, tang0 := lt.pos.Sub(centre), lt.tang
	steps := int(stdmath.Max(2, stdmath.Round(angle/lipFacetStep)))
	for k := 0; k <= steps; k++ {
		phi := angle * float64(k) / float64(steps)
		lt.pos = centre.Add(rotate2(rel0, phi))
		lt.tang = rotate2(tang0, phi)
		lt.emit()
	}
}

// SheetMetalLipFeatures adds lip features into the engine.
type SheetMetalLipFeatures struct{ engine *PartFeatures }

// NewSheetMetalLipFeatures binds the collection to a feature engine.
func NewSheetMetalLipFeatures(engine *PartFeatures) *SheetMetalLipFeatures {
	return &SheetMetalLipFeatures{engine}
}

// Add appends a lip feature, naming it Lip1, Lip2, … . (The dress-up "lip" bead is a separate,
// solid-modeling feature; this is the sheet-metal edge return.)
func (c *SheetMetalLipFeatures) Add(def *SheetMetalLipDefinition) *PartFeature {
	f := &SheetMetalLipFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Lip"))
	f.featName = pf.Name()
	return pf
}

var _ Feature = (*SheetMetalLipFeature)(nil)
