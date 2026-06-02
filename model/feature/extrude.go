// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"github.com/Oblikovati/oblikovati/build"
	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
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
}

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

// Recompute resolves the profile, builds the prism solid at the extent distance,
// and applies the operation against the running bodies.
func (e *ExtrudeFeature) Recompute(in Input) (Output, error) {
	if e.def.Extent.Type != DistanceExtent {
		return Output{}, build.NotYetImplemented("PBI-092-extent-" + fmt.Sprint(e.def.Extent.Type))
	}
	profiles, err := e.resolveProfiles()
	if err != nil {
		return Output{}, err
	}
	dist := e.def.Extent.distance()
	if dist == 0 {
		return Output{}, errors.New("extrude distance is zero")
	}
	bodies, err := combine(in.Bodies, e.buildTool(profiles, dist), e.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// buildTool extrudes each selected region into a prism and merges the prisms into one
// body. The regions are distinct cells of the same sketch, so they never overlap — a
// shell merge is exactly their union and avoids the unimplemented intersecting Join.
func (e *ExtrudeFeature) buildTool(profiles []*sketch.Profile, dist float64) *topo.Body {
	plane := e.def.Sketch.Plane()
	prisms := make([]*topo.Body, len(profiles))
	for i, p := range profiles {
		prisms[i] = buildPrism(p.OuterLoop().Polygon(), plane, dist, e.prismFeat(i, len(profiles)))
	}
	if len(prisms) == 1 {
		return prisms[0]
	}
	return topo.MergeBodies(topo.NewLineage(topo.Tok(e.featName, "merged", 0)), true, prisms...)
}

// prismFeat namespaces each region's prism lineage; a single-profile extrude keeps the
// bare feature name so its persistent face IDs are unchanged.
func (e *ExtrudeFeature) prismFeat(i, n int) string {
	if n == 1 {
		return e.featName
	}
	return fmt.Sprintf("%s/p%d", e.featName, i)
}

// resolveProfiles re-derives the selected closed regions from the sketch, erroring
// (→ sick) when one is missing or open, or when none is selected.
func (e *ExtrudeFeature) resolveProfiles() ([]*sketch.Profile, error) {
	all := e.def.Sketch.Profiles()
	if len(e.def.ProfileIndices) == 0 {
		return nil, errors.New("extrude: no profile selected")
	}
	out := make([]*sketch.Profile, 0, len(e.def.ProfileIndices))
	for _, idx := range e.def.ProfileIndices {
		if idx < 0 || idx >= all.Count() {
			return nil, fmt.Errorf("extrude: profile %d not found (sketch has %d)", idx, all.Count())
		}
		p := all.Item(idx)
		if !p.IsClosed() {
			return nil, errors.New("extrude: profile is open (cannot form a solid)")
		}
		out = append(out, p)
	}
	return out, nil
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
	def := &ExtrudeDefinition{
		Sketch: skt, ProfileIndices: append([]int(nil), profileIndices...), Operation: op,
		Extent: Extent{Type: DistanceExtent, Distance: distance},
	}
	ef := &ExtrudeFeature{def: def}
	pf := c.engine.Add(ef)
	pf.SetName(c.engine.UniqueName("Extrusion")) // Extrusion1, Extrusion2, … (Inventor's naming)
	ef.featName = pf.name
	return pf
}

// combine applies an operation between the running bodies and a new body. Phase A
// handles the first body / new-body and the non-overlapping boolean cases.
func combine(running []*topo.Body, body *topo.Body, op ops.PartFeatureOperation) ([]*topo.Body, error) {
	if len(running) == 0 || op == ops.NewBody {
		return append(append([]*topo.Body(nil), running...), body), nil
	}
	target := running[len(running)-1]
	res, err := ops.Boolean(op, target, body)
	if err != nil {
		return nil, err
	}
	out := append([]*topo.Body(nil), running[:len(running)-1]...)
	if res != nil && len(res.Faces()) > 0 {
		out = append(out, res)
	}
	return out, nil
}

// buildPrism extrudes a closed polygon by distance along the plane normal, building
// a watertight solid: a bottom cap, a top cap, and one planar side face per profile
// edge, with lineage on each (start/end caps and indexed side walls).
func buildPrism(poly []math.Point2, plane sketch.Plane, dist float64, feat string) *topo.Body {
	n := len(poly)
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok(feat, "body", 0)))
	up := plane.Normal().AsVector().Scale(dist)
	bottom := make([]*topo.Vertex, n)
	top := make([]*topo.Vertex, n)
	for i, p := range poly {
		b := plane.ToModel(p)
		bottom[i] = bld.AddVertex(b, topo.NewLineage(topo.Tok(feat, "vertex", i)))
		top[i] = bld.AddVertex(b.TranslateBy(up), topo.NewLineage(topo.Tok(feat, "vertex", n+i)))
	}
	be, te, ve := prismEdges(bld, bottom, top, feat)
	addCaps(bld, bottom, top, be, te, plane, feat)
	addSides(bld, bottom, be, te, ve, plane, outwardSign(poly), feat)
	return bld.Build()
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

// addCaps builds the bottom (downward) and top (upward) cap faces.
func addCaps(bld *topo.Builder, bottom, top []*topo.Vertex, be, te []*topo.Edge, plane sketch.Plane, feat string) {
	n := len(bottom)
	down := plane.Normal().AsVector().Negate()
	bottomPlane, _ := geom.NewPlane(bottom[0].Point(), down)
	topPlane, _ := geom.NewPlane(top[0].Point(), plane.Normal().AsVector())
	bottomLoop := make([]topo.Use, n)
	topLoop := make([]topo.Use, n)
	for i := 0; i < n; i++ {
		bottomLoop[i] = topo.Rev(be[n-1-i]) // reverse order & direction → outward-down
		topLoop[i] = topo.Fwd(te[i])
	}
	bld.AddFace(bottomPlane, topo.NewLineage(topo.Tok(feat, "start-cap", 0)), topo.OuterLoop(bottomLoop...))
	bld.AddFace(topPlane, topo.NewLineage(topo.Tok(feat, "end-cap", 0)), topo.OuterLoop(topLoop...))
}

// addSides builds one planar side wall per profile edge. sign flips the computed
// outward normal for a clockwise profile (see outwardSign) so every wall faces away
// from the solid's interior.
func addSides(bld *topo.Builder, bottom []*topo.Vertex, be, te, ve []*topo.Edge, plane sketch.Plane, sign float64, feat string) {
	n := len(bottom)
	normal := plane.Normal().AsVector()
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		edgeDir := bottom[i].Point().VectorTo(bottom[j].Point())
		outward, err := math.UnitVector3FromVector(edgeDir.Cross(normal).Scale(sign))
		surf := sideSurface(bottom[i].Point(), outward, err)
		loop := topo.OuterLoop(topo.Fwd(be[i]), topo.Fwd(ve[j]), topo.Rev(te[i]), topo.Rev(ve[i]))
		bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "side", i)), loop)
	}
}

// sideSurface returns the side face's plane (falling back to a degenerate plane if
// the edge was zero-length, which the validator will then flag).
func sideSurface(origin math.Point3, outward math.UnitVector3, err error) geom.Surface {
	if err != nil {
		p, _ := geom.NewPlane(origin, math.V3(0, 0, 1))
		return p
	}
	p, _ := geom.NewPlane(origin, outward.AsVector())
	return p
}
