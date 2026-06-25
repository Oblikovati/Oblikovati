// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// ExtrudeDefinition is the recipe for an extrude (the Definition of the triangle):
// a sketch profile, the operation against existing material, the extent, and a
// taper. It re-derives the profile from its sketch each recompute (so sketch edits
// flow through), going sick if the profile is gone or open.
type ExtrudeDefinition struct {
	Sketch         *sketch.Sketch
	ProfileIndices []int // one or more sketch regions, extruded together into one feature
	Operation      ops.PartFeatureOperation
	Extent         Extent
	Taper          float64 // draft angle (radians); 0 in phase A (planar sides)
}

// ExtrudeFeature turns a profile into a prism and combines it with the running
// body state. It is the reference sketched feature (PBI-092).
type ExtrudeFeature struct {
	def      *ExtrudeDefinition
	featName string
	tool     *topo.Body // last prism built, exposed so a pattern can replicate this feature
}

// ToolBody returns the prism this feature last combined into the model — the clean tool a
// pattern replicates at each occurrence (more robust than diffing before/after bodies,
// especially for curved geometry). It is nil before the first recompute.
func (e *ExtrudeFeature) ToolBody() *topo.Body { return e.tool }

// Definition returns the extrude recipe (round-trippable, serializable).
func (e *ExtrudeFeature) Definition() *ExtrudeDefinition { return e.def }

// Kind implements [Feature].
func (e *ExtrudeFeature) Kind() string { return "extrude" }

// DistanceValue returns the current extent distance (database units) — the value a
// feature editor shows when re-opening the extrude.
func (e *ExtrudeFeature) DistanceValue() float64 { return e.def.Extent.distance() }

// SetDistance replaces the extent with a constant distance, keeping the extent type and
// direction — the feature editor's distance field writes through here. Mark the feature
// dirty and recompute afterwards for the change to take effect.
func (e *ExtrudeFeature) SetDistance(d float64) {
	e.def.Extent.Distance = func() float64 { return d }
}

// Operation returns the boolean operation applied against the existing bodies.
func (e *ExtrudeFeature) Operation() ops.PartFeatureOperation { return e.def.Operation }

// SetOperation changes the boolean operation (join/cut/intersect/new-body).
func (e *ExtrudeFeature) SetOperation(op ops.PartFeatureOperation) { e.def.Operation = op }

// Extent returns the feature's termination (type/direction/distances/targets); SetExtent
// replaces it — the feature editor and Extrude tool write the chosen mode through here.
func (e *ExtrudeFeature) Extent() Extent           { return e.def.Extent }
func (e *ExtrudeFeature) SetExtent(ext Extent)     { e.def.Extent = ext }
func (e *ExtrudeFeature) Taper() float64           { return e.def.Taper }
func (e *ExtrudeFeature) SetTaper(radians float64) { e.def.Taper = radians }

// Recompute resolves the profile, computes the extent span (distance/through-all/to-face/
// …), builds the prism solid the span sweeps, and applies the operation against the
// running bodies.
func (e *ExtrudeFeature) Recompute(in Input) (Output, error) {
	profiles, err := e.resolveProfiles()
	if err != nil {
		return Output{}, err
	}
	plane := e.def.Sketch.Plane()
	sp, err := e.resolveSpan(in.Bodies, plane, outerPolygons(profiles))
	if err != nil {
		return Output{}, err
	}
	if sp.depth() == 0 {
		return Output{}, errors.New("extrude: the extent has zero depth")
	}
	e.tool = e.buildTool(profiles, plane, sp)
	bodies, err := combine(in.Bodies, e.tool, e.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// buildTool extrudes each selected region into a prism over the span and merges the
// prisms into one body. The regions are distinct cells of the same sketch, so they never
// overlap — a shell merge is exactly their union and avoids the intersecting Join.
func (e *ExtrudeFeature) buildTool(profiles []*sketch.Profile, plane sketch.Plane, sp span) *topo.Body {
	return buildProfilePrisms(profiles, plane, sp, e.def.Taper, e.featName)
}

// outerPolygons returns each profile's outer-loop polygon, the input the span resolver
// (to-next ray-casting) and the prism builder both consume.
func outerPolygons(profiles []*sketch.Profile) [][]math.Point2 {
	out := make([][]math.Point2, len(profiles))
	for i, p := range profiles {
		out[i] = p.OuterLoop().Polygon()
	}
	return out
}

// resolveProfiles re-derives the selected closed regions from the sketch (the shared
// resolver), erroring (→ sick) when one is missing or open, or when none is selected.
func (e *ExtrudeFeature) resolveProfiles() ([]*sketch.Profile, error) {
	return resolveClosedProfiles(e.def.Sketch, e.def.ProfileIndices, "extrude")
}

// ExtrudeFeatures is the collection of extrude features, adding into the engine.
type ExtrudeFeatures struct {
	engine *PartFeatures
}

// NewExtrudeFeatures binds the collection to a feature engine.
func NewExtrudeFeatures(engine *PartFeatures) *ExtrudeFeatures {
	return &ExtrudeFeatures{engine: engine}
}

// AddByDistanceExtent adds an extrude of a single sketch region, growing distance (a
// closure, typically a parameter) under the given operation.
func (c *ExtrudeFeatures) AddByDistanceExtent(skt *sketch.Sketch, profileIndex int, op ops.PartFeatureOperation, distance func() float64) *PartFeature {
	return c.AddByDistanceExtentProfiles(skt, []int{profileIndex}, op, distance)
}

// AddByDistanceExtentProfiles adds an extrude of one or more sketch regions, merged
// into one body — the multi-region selection the Extrude tool gathers with Ctrl+click.
func (c *ExtrudeFeatures) AddByDistanceExtentProfiles(skt *sketch.Sketch, profileIndices []int, op ops.PartFeatureOperation, distance func() float64) *PartFeature {
	return c.AddExtrude(skt, profileIndices, op, Extent{Type: DistanceExtent, Distance: distance}, 0)
}

// AddExtrude adds an extrude of one or more regions with a fully-specified extent
// (distance / through-all / to-face / from-to / to-next / distance-from-face), boolean
// operation, and taper (draft) angle — the general constructor the Extrude tool and the
// automation API drive. The distance constructors above delegate here.
func (c *ExtrudeFeatures) AddExtrude(skt *sketch.Sketch, profileIndices []int, op ops.PartFeatureOperation, extent Extent, taper float64) *PartFeature {
	return c.AddExtrudeFeature(&ExtrudeDefinition{
		Sketch: skt, ProfileIndices: append([]int(nil), profileIndices...),
		Operation: op, Extent: extent, Taper: taper,
	})
}

// AddExtrudeFeature registers and names an extrude built from def — the seam shared by the
// arg-based AddExtrude above and the Extrude tool, which builds the same def for both the
// live preview ([NewExtrudeFeature]) and commit so the previewed geometry is exactly what
// OK creates.
func (c *ExtrudeFeatures) AddExtrudeFeature(def *ExtrudeDefinition) *PartFeature {
	ef := NewExtrudeFeature(def)
	pf := c.engine.Add(ef)
	pf.SetName(c.engine.UniqueName("Extrusion")) // Extrusion1, Extrusion2, … (Inventor's naming)
	ef.featName = pf.name
	return pf
}

// NewExtrudeFeature builds an extrude feature value from a definition WITHOUT adding it to
// any engine — the unattached, unnamed [Feature] the live preview evaluates speculatively
// (see [PartFeatures.PreviewResult]). AddExtrudeFeature wraps this, then registers and names it.
func NewExtrudeFeature(def *ExtrudeDefinition) *ExtrudeFeature { return &ExtrudeFeature{def: def} }

// combine applies an operation between the running bodies and a new body. Phase A
// handles the first body / new-body and the non-overlapping boolean cases.
func combine(running []*topo.Body, body *topo.Body, op ops.PartFeatureOperation) ([]*topo.Body, error) {
	if len(running) == 0 || op == ops.NewBody {
		return append(append([]*topo.Body(nil), running...), body), nil
	}
	target := running[len(running)-1]
	// First try the EXACT curved boolean on the still-analytic operands when the target is a BARE analytic
	// primitive (exactly one curved face — a revolved torus, an extruded cylinder/cone, a sphere) and the
	// tool is all-planar (a prism): those keep their curved faces through the M2 curved boolean
	// (#1334/#1335). The single-curved-face gate is deliberately tight — a composite curved body (a washer's
	// two cylinder walls, a filleted edge) is NOT a primitive the half-space cut handles, so it stays on the
	// faceted planar path; CurvedBoolean can over-match such a body and cut it wrongly.
	if curvedFaceCount(target) == 1 && curvedFaceCount(body) == 0 {
		if res, ok := ops.CurvedBoolean(op, target, body); ok {
			return appendCombined(running, res), nil
		}
	}
	// Otherwise re-facet any analytic curved face into a planar B-rep before the planar boolean — it hangs
	// on a full periodic curved face it cannot consume (#129). A standalone primitive that is never
	// combined keeps its analytic face for thread/chamfer/fillet.
	target = planarized(target, "combine-target")
	body = planarized(body, "combine-tool")
	res, err := ops.Boolean(op, target, body)
	if err != nil {
		return nil, err
	}
	return appendCombined(running, res), nil
}

// appendCombined replaces the running target with the boolean result, dropping an empty result.
func appendCombined(running []*topo.Body, res *topo.Body) []*topo.Body {
	out := append([]*topo.Body(nil), running[:len(running)-1]...)
	if res != nil && len(res.Faces()) > 0 {
		out = append(out, res)
	}
	return out
}

// buildPrism extrudes a closed polygon over the span (near→far offsets along the plane
// normal) into a watertight solid: a near cap, a far cap, and one planar side face per
// profile edge. A non-zero taper offsets the far loop by depth·tan(taper) so the sides
// draft outward (positive) or inward (negative); the far cap stays perpendicular to the
// normal, and each side stays a planar trapezoid.
func buildPrism(poly []math.Point2, plane sketch.Plane, sp span, taper float64, feat string) *topo.Body {
	n := len(poly)
	normal := plane.Normal().AsVector()
	topPoly := taperedLoop(poly, sp.depth(), taper)
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok(feat, "body", 0)))
	bottom := make([]*topo.Vertex, n)
	top := make([]*topo.Vertex, n)
	for i := 0; i < n; i++ {
		b := plane.ToModel(poly[i]).TranslateBy(normal.Scale(sp.near))
		t := plane.ToModel(topPoly[i]).TranslateBy(normal.Scale(sp.far))
		bottom[i] = bld.AddVertex(b, topo.NewLineage(topo.Tok(feat, "vertex", i)))
		top[i] = bld.AddVertex(t, topo.NewLineage(topo.Tok(feat, "vertex", n+i)))
	}
	be, te, ve := prismEdges(bld, bottom, top, feat)
	addCaps(bld, bottom, top, be, te, normal, feat)
	addSides(bld, bottom, top, be, te, ve, outwardSign(poly), feat)
	return bld.Build()
}

// taperedLoop returns the far-loop polygon: poly unchanged when taper is 0, else each
// vertex offset along its outward bisector by depth·tan(taper) (positive widens).
func taperedLoop(poly []math.Point2, depth, taper float64) []math.Point2 {
	if taper == 0 {
		return poly
	}
	delta := depth * stdmath.Tan(taper) * outwardSign(poly)
	return offsetPolygon2D(poly, delta)
}

// offsetPolygon2D offsets each vertex outward by delta along the angle bisector of its two
// incident edges (a simple miter offset, exact for convex corners).
func offsetPolygon2D(poly []math.Point2, delta float64) []math.Point2 {
	n := len(poly)
	out := make([]math.Point2, n)
	for i := 0; i < n; i++ {
		prev := poly[(i-1+n)%n]
		next := poly[(i+1)%n]
		nIn := edgeNormal2D(prev, poly[i])
		nOut := edgeNormal2D(poly[i], next)
		bisect := math.V2(nIn.X+nOut.X, nIn.Y+nOut.Y)
		u, err := math.UnitVector2FromVector(bisect)
		if err != nil {
			out[i] = poly[i]
			continue
		}
		scale := delta / cosHalf(nIn, nOut)
		out[i] = math.P2(poly[i].X+u.AsVector().X*scale, poly[i].Y+u.AsVector().Y*scale)
	}
	return out
}

// edgeNormal2D returns the left-hand unit normal of edge a→b (points outward for a CCW loop).
func edgeNormal2D(a, b math.Point2) math.Vector2 {
	d := math.V2(b.X-a.X, b.Y-a.Y)
	u, err := math.UnitVector2FromVector(math.V2(d.Y, -d.X))
	if err != nil {
		return math.V2(0, 0)
	}
	return u.AsVector()
}

// cosHalf returns the cosine of the half-angle between two edge normals, the miter
// factor that keeps the bisector offset's perpendicular distance equal to delta (clamped
// so a near-straight corner does not blow up the offset).
func cosHalf(nIn, nOut math.Vector2) float64 {
	c := stdmath.Sqrt(stdmath.Max(0, (1+nIn.X*nOut.X+nIn.Y*nOut.Y)/2))
	if c < 0.2 {
		return 0.2
	}
	return c
}

// outwardSign returns +1 when the profile polygon is wound counter-clockwise in the
// sketch plane and −1 when clockwise. Side-wall normals are built as edgeDir×normal,
// which points away from the interior only for a CCW loop; profile detection does not
// guarantee a winding, so a clockwise loop must flip the side normals to stay coherent
// with the (fixed up/down) caps — otherwise the prism is "inside-out" and breaks
// rendering, volume, and downstream booleans.
func outwardSign(poly []math.Point2) float64 {
	area := 0.0
	for i, n := 0, len(poly); i < n; i++ {
		j := (i + 1) % n
		area += poly[i].X*poly[j].Y - poly[j].X*poly[i].Y
	}
	if area < 0 {
		return -1
	}
	return 1
}

// prismEdges builds the bottom, top and vertical edges and returns them.
func prismEdges(bld *topo.Builder, bottom, top []*topo.Vertex, feat string) (be, te, ve []*topo.Edge) {
	n := len(bottom)
	be, te, ve = make([]*topo.Edge, n), make([]*topo.Edge, n), make([]*topo.Edge, n)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		be[i] = bld.AddEdge(geom.NewLineSegment(bottom[i].Point(), bottom[j].Point()), bottom[i], bottom[j], topo.NewLineage(topo.Tok(feat, "bottom-edge", i)))
		te[i] = bld.AddEdge(geom.NewLineSegment(top[i].Point(), top[j].Point()), top[i], top[j], topo.NewLineage(topo.Tok(feat, "top-edge", i)))
		ve[i] = bld.AddEdge(geom.NewLineSegment(bottom[i].Point(), top[i].Point()), bottom[i], top[i], topo.NewLineage(topo.Tok(feat, "side-edge", i)))
	}
	return be, te, ve
}

// addCaps builds the near (downward) and far (upward) cap faces, perpendicular to the
// extrude normal at each end.
func addCaps(bld *topo.Builder, bottom, top []*topo.Vertex, be, te []*topo.Edge, normal math.Vector3, feat string) {
	n := len(bottom)
	bottomPlane, _ := geom.NewPlane(bottom[0].Point(), normal.Negate())
	topPlane, _ := geom.NewPlane(top[0].Point(), normal)
	bottomLoop := make([]topo.Use, n)
	topLoop := make([]topo.Use, n)
	for i := 0; i < n; i++ {
		bottomLoop[i] = topo.Rev(be[n-1-i]) // reverse order & direction → outward-down
		topLoop[i] = topo.Fwd(te[i])
	}
	bld.AddFace(bottomPlane, topo.NewLineage(topo.Tok(feat, "start-cap", 0)), topo.OuterLoop(bottomLoop...))
	bld.AddFace(topPlane, topo.NewLineage(topo.Tok(feat, "end-cap", 0)), topo.OuterLoop(topLoop...))
}

// addSides builds one planar side wall per profile edge, through the wall's corners so
// the face tilts correctly when the extrude is tapered. sign flips the outward normal for
// a clockwise profile (see outwardSign) so every wall faces away from the interior.
func addSides(bld *topo.Builder, bottom, top []*topo.Vertex, be, te, ve []*topo.Edge, sign float64, feat string) {
	n := len(bottom)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		surf := sideSurface(bottom[i].Point(), bottom[j].Point(), top[i].Point(), sign)
		loop := topo.OuterLoop(topo.Fwd(be[i]), topo.Fwd(ve[j]), topo.Rev(te[i]), topo.Rev(ve[i]))
		bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "side", i)), loop)
	}
}

// sideSurface returns the side face's plane through corners b0,b1,t0 (b0→b1 along the
// profile edge, b0→t0 up the wall), oriented outward by sign. Falls back to a degenerate
// plane on a zero-area corner, which the validator then flags.
func sideSurface(b0, b1, t0 math.Point3, sign float64) geom.Surface {
	normal, err := math.UnitVector3FromVector(b0.VectorTo(b1).Cross(b0.VectorTo(t0)).Scale(sign))
	if err != nil {
		p, _ := geom.NewPlane(b0, math.V3(0, 0, 1))
		return p
	}
	p, _ := geom.NewPlane(b0, normal.AsVector())
	return p
}
