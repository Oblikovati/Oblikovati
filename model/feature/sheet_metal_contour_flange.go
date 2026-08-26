// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"
	"slices"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Sheet-metal Contour Flange feature (M13-F02). A contour flange sweeps a user-drawn OPEN
// profile (the wall's cross-section — any chain of straight segments) along a sheet edge,
// instead of the straight-wall-plus-arc the plain Flange computes. The profile is the inner
// (top-face-side) contour; thickening it by the rule's material thickness and extruding the
// resulting band along the edge gives the flange, unioned onto the sheet. The profile's first
// point sits at the edge, so the band's start segment is the sheet's edge cross-section and
// the union is watertight. Thickness is read live from the rule, like every other wall.

// SheetMetalContourFlangeDefinition is the contour-flange recipe: the edge to sweep along, the
// open profile sketch (the cross-section), and a flip that folds to the opposite side.
type SheetMetalContourFlangeDefinition struct {
	EdgeKey []byte
	Profile *sketch.Sketch
	Flip    bool
	// Width is how much of the edge the swept wall covers (#1958); the zero value is the whole
	// edge, which is what this feature has always built.
	Width FlangeWidth
	// Operation is how the wall joins the model (#1961): Join unions it onto the running sheet (the
	// default and what this feature always did), NewBody starts a body of its own.
	Operation ops.PartFeatureOperation
	// Radius rounds the profile's corners into bends; nil ⇒ the rule's BendRadius. A contour
	// flange's corners ARE bends, and they were swept sharp before this.
	Radius func() float64
}

// SheetMetalContourFlangeFeature sweeps the profile onto the sheet each recompute.
type SheetMetalContourFlangeFeature struct {
	def      *SheetMetalContourFlangeDefinition
	featName string
}

// Definition returns the contour-flange recipe.
func (f *SheetMetalContourFlangeFeature) Definition() *SheetMetalContourFlangeDefinition {
	return f.def
}

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalContourFlangeFeature) Kind() string { return "sheet-metal-contour-flange" }

// Recompute resolves the edge and profile, builds the thickened band, and unions it on.
func (f *SheetMetalContourFlangeFeature) Recompute(in Input) (Output, error) {
	body, err := lastBody(in, "sheet-metal contour flange")
	if err != nil {
		return Output{}, err
	}
	t, err := sheetThickness(in.Params)
	if err != nil {
		return Output{}, err
	}
	profile, err := f.bentProfile(in)
	if err != nil {
		return Output{}, err
	}
	edges, heals, err := resolveEdges(body, [][]byte{f.def.EdgeKey}, nil)
	if err != nil {
		return Output{}, err
	}
	wall, err := buildContourFlangeSolid(edges[0], profile, t, f.def.Width, f.def.Flip, f.featName)
	if err != nil {
		return Output{}, err
	}
	bodies, err := combine(in, wall, f.operation())
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies, Heals: heals}, nil
}

// bentProfile reads the drawn profile and rounds its corners into bends, which is what they are —
// a contour flange swept with sharp corners is geometry no press brake can make.
func (f *SheetMetalContourFlangeFeature) bentProfile(in Input) ([]math.Point2, error) {
	profile, err := openProfilePoints(f.def.Profile)
	if err != nil {
		return nil, err
	}
	return roundProfileCorners(profile, f.bendRadius(in))
}

// BendSpecs reports one bend per rounded interior corner of the contour profile, for the flat
// pattern (#2076). Each corner of a contour flange IS a bend — it was swept sharp until #1961
// rounded it to the bend radius — so the flat blank must develop each corner's bend allowance, or
// it comes out short by the whole of it and any nest or DXF cut from it is undersized. The swept
// angle is the corner's own turn; the radius follows the flange convention (the override, else 0 to
// defer to the rule's default, which compdef.developBends fills in). Before the first recompute the
// profile still reads from the definition, so this needs no captured state.
//
// Example:
//
//	specs := contourFlange.BendSpecs(thickness) // one BendSpec per non-collinear profile corner
func (f *SheetMetalContourFlangeFeature) BendSpecs(_ float64) []BendSpec {
	profile, err := openProfilePoints(f.def.Profile)
	if err != nil {
		return nil
	}
	radius := 0.0
	if f.def.Radius != nil {
		radius = f.def.Radius()
	}
	return contourCornerBends(profile, radius)
}

// operation is how this wall joins the model; the zero value is Join, which is what the feature
// has always done.
func (f *SheetMetalContourFlangeFeature) operation() ops.PartFeatureOperation {
	if f.def.Operation == ops.NewBody {
		return ops.NewBody
	}
	return ops.Join
}

// bendRadius is the radius the profile's corners are rounded to: the feature's override, else the
// rule's BendRadius parameter — the same order a plain flange resolves it in.
func (f *SheetMetalContourFlangeFeature) bendRadius(in Input) float64 {
	if f.def.Radius != nil {
		return f.def.Radius()
	}
	if in.Params != nil {
		if p, ok := in.Params.ByName(flangeBendParamName); ok {
			return p.ModelValue()
		}
	}
	return 0
}

// buildContourFlangeSolid builds the contour-flange solid: the profile band mapped onto the
// edge's section frame and extruded along the edge.
func buildContourFlangeSolid(edge *topo.Edge, profile []math.Point2, thickness float64,
	width FlangeWidth, flip bool, feat string) (*topo.Body, error) {
	v0, v1 := edge.StartVertex().Point(), edge.EndVertex().Point()
	e, err := math.UnitVector3FromVector(v0.VectorTo(v1))
	if err != nil {
		return nil, fmt.Errorf("sheet-metal contour flange: degenerate edge")
	}
	up, out, err := flangeFrame(edge, v0.Midpoint(v1), e, flip)
	if err != nil {
		return nil, err
	}
	from, to, err := width.span(float64(v0.DistanceTo(v1)))
	if err != nil {
		return nil, err
	}
	plane := planePerp(v0, e)
	u, v := plane.XAxis().AsVector(), plane.YAxis().AsVector()
	// Map a profile point (px along out, py along up) into the section plane's 2D frame.
	at := func(p math.Point2) math.Point2 {
		w := out.AsVector().Scale(float64(p.X)).Add(up.AsVector().Scale(float64(p.Y)))
		return math.P2(math.Scalar(w.Dot(u)), math.Scalar(w.Dot(v)))
	}
	band := contourBand(profile, thickness, at)
	return buildPrism(band, plane, span{near: from, far: to}, 0, feat), nil
}

// contourBand returns the closed cross-section band for the profile mapped into the section
// plane via at.
func contourBand(profile []math.Point2, thickness float64, at func(math.Point2) math.Point2) []math.Point2 {
	band := profileBand2D(profile, thickness)
	for i, p := range band {
		band[i] = at(p)
	}
	return band
}

// mitreVector is the offset direction at a corner between two unit normals: the vector whose
// projection on each is 1, so an offset by d clears both faces by d. Segments that double back
// (opposing normals) have no mitre at all — the profile would have to be offset to infinity — so
// the incoming normal is used and the corner is left un-mitred rather than blown up.
func mitreVector(n1, n2 math.Vector2) math.Vector2 {
	denom := 1 + float64(n1.Dot(n2))
	if denom < 1e-9 {
		return n1
	}
	return n1.Add(n2).Scale(math.Scalar(1 / denom))
}

// profileBand2D returns the closed constant-thickness band of an open profile: the inner
// contour, then its outer offset (toward the material side) reversed. Shared by the contour
// flange and the lofted flange.
func profileBand2D(profile []math.Point2, thickness float64) []math.Point2 {
	outer := offsetProfile(profile, -thickness)
	band := make([]math.Point2, 0, len(profile)+len(outer))
	band = append(band, profile...)
	for _, o := range slices.Backward(outer) {
		band = append(band, o)
	}
	return band
}

// offsetProfile offsets an open polyline by d along each segment's left normal, mitring at
// interior vertices (a first-order join — good for the straight contours this feature takes).
func offsetProfile(pts []math.Point2, d float64) []math.Point2 {
	if len(pts) < 2 {
		return pts
	}
	out := make([]math.Point2, len(pts))
	for i := range pts {
		out[i] = pts[i].TranslateBy(vertexNormal(pts, i).Scale(math.Scalar(d)))
	}
	return out
}

// vertexNormal returns the offset direction at vertex i: the unit left-normal at an endpoint, and
// at an interior vertex the MITRE vector — the one whose perpendicular distance to BOTH adjacent
// segments is the offset, not merely its own length.
//
// It used to return the normalised average of the two normals, which is a unit vector, so offsetting
// by the gauge moved the corner only gauge·cos(half the turn) away from each face: the band came out
// thin at every corner, by 29% at a right angle. The mitre m = (n1+n2)/(1+n1·n2) satisfies
// m·n1 = m·n2 = 1, which is what "offset by the gauge" has to mean on both faces at once.
func vertexNormal(pts []math.Point2, i int) math.Vector2 {
	left := func(a, b math.Point2) math.Vector2 {
		dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
		l := stdmath.Hypot(dx, dy)
		if l == 0 {
			return math.V2(0, 0)
		}
		return math.V2(math.Scalar(-dy/l), math.Scalar(dx/l))
	}
	switch {
	case i == 0:
		return left(pts[0], pts[1])
	case i == len(pts)-1:
		return left(pts[i-1], pts[i])
	default:
		return mitreVector(left(pts[i-1], pts[i]), left(pts[i], pts[i+1]))
	}
}

// openProfilePoints walks a sketch's line entities into a single ordered open polyline,
// erroring when the sketch is empty or not one open chain.
func openProfilePoints(sk *sketch.Sketch) ([]math.Point2, error) {
	if sk == nil {
		return nil, fmt.Errorf("sheet-metal contour flange: no profile sketch")
	}
	lines := sk.Lines()
	n := lines.Count()
	deg := map[*sketch.Point]int{}
	profileLines := 0
	for i := range n {
		l := lines.Item(i)
		if l.IsCenterline() { // an axis/centerline (e.g. a contour-roll axis) is not part of the profile
			continue
		}
		profileLines++
		deg[l.StartPoint()]++
		deg[l.EndPoint()]++
	}
	if profileLines == 0 {
		return nil, fmt.Errorf("sheet-metal contour flange: profile sketch has no profile lines")
	}
	start := chainStart(deg)
	if start == nil {
		return nil, fmt.Errorf("sheet-metal contour flange: profile must be one open chain")
	}
	return walkOpenChain(lines, n, profileLines, start)
}

// chainStart returns the open-chain end (degree-1 vertex) nearest the sketch origin — the
// profile's edge-attachment point by convention — or nil when no open end exists (a closed
// loop or a branching set). Choosing the nearest-origin end makes the walk deterministic
// regardless of map iteration order.
func chainStart(deg map[*sketch.Point]int) *sketch.Point {
	var best *sketch.Point
	bestD2 := stdmath.Inf(1)
	for p, d := range deg {
		if d != 1 {
			continue
		}
		pos := p.Position()
		d2 := float64(pos.X)*float64(pos.X) + float64(pos.Y)*float64(pos.Y)
		if d2 < bestD2 {
			best, bestD2 = p, d2
		}
	}
	return best
}

// walkOpenChain follows the connected profile segments from start to the far end, collecting
// the ordered vertex positions, erroring unless they form one simple chain of all the
// profileLines (centerlines are skipped).
func walkOpenChain(lines *sketch.Lines, n, profileLines int, start *sketch.Point) ([]math.Point2, error) {
	used := make([]bool, n)
	pts := []math.Point2{start.Position()}
	cur := start
	for range profileLines {
		next, idx := nextSegment(lines, n, used, cur)
		if next == nil {
			return nil, fmt.Errorf("sheet-metal contour flange: profile is not a single connected chain")
		}
		used[idx] = true
		pts = append(pts, next.Position())
		cur = next
	}
	return pts, nil
}

// nextSegment finds an unused, non-centerline line touching cur and returns its other endpoint
// and index.
func nextSegment(lines *sketch.Lines, n int, used []bool, cur *sketch.Point) (*sketch.Point, int) {
	for i := range n {
		if used[i] {
			continue
		}
		l := lines.Item(i)
		if l.IsCenterline() {
			continue
		}
		switch cur {
		case l.StartPoint():
			return l.EndPoint(), i
		case l.EndPoint():
			return l.StartPoint(), i
		}
	}
	return nil, -1
}

// SheetMetalContourFlangeFeatures adds contour-flange features into the engine.
type SheetMetalContourFlangeFeatures struct{ engine *PartFeatures }

// NewSheetMetalContourFlangeFeatures binds the collection to a feature engine.
func NewSheetMetalContourFlangeFeatures(engine *PartFeatures) *SheetMetalContourFlangeFeatures {
	return &SheetMetalContourFlangeFeatures{engine}
}

// Add appends a contour-flange feature, naming it ContourFlange1, … .
func (c *SheetMetalContourFlangeFeatures) Add(def *SheetMetalContourFlangeDefinition) *PartFeature {
	f := &SheetMetalContourFlangeFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("ContourFlange"))
	f.featName = pf.Name()
	return pf
}

var (
	_ Feature     = (*SheetMetalContourFlangeFeature)(nil)
	_ BendLineage = (*SheetMetalContourFlangeFeature)(nil) // #2076: its corners develop as bends
)
