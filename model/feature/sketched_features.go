// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// The remaining sketched features carry their full Definition (the triangle) and
// extent/operation surface. Revolve and coil generate real (faceted) solids of
// revolution via the shared swept-solid primitive; sweep/loft generation is below.
// Curved profile edges and exact analytic/NURBS surfaces are a later refinement —
// phase A approximates the swept surfaces as planar facets, a real watertight solid.

// revolveSegments is the angular facet count for a full revolution; partial angles
// use a proportional share (so a 90° revolve gets ~1/4 the facets).
const revolveSegments = 64

// RevolveDefinition is the recipe for a revolve: a profile spun about an axis. The axis is, in
// precedence: an explicit work axis; else a specific centerline (AxisCenterline on its sketch);
// else the profile sketch's single centerline (auto).
type RevolveDefinition struct {
	Sketch               *sketch.Sketch
	ProfileIndex         int
	Axis                 *WorkAxis
	AxisCenterline       *sketch.Line   // a specific centerline to revolve about
	AxisCenterlineSketch *sketch.Sketch // the centerline's sketch (for its plane)
	Angle                func() float64 // 0 ⇒ full revolution
	// Angle2 is the second-direction sweep (radians, opposite sense), the
	// reference two-directional revolve (M08 PBI-093, #313). nil/0 ⇒ one-way.
	Angle2    func() float64
	Operation ops.PartFeatureOperation
}

// RevolveFeature spins a profile about an axis.
type RevolveFeature struct {
	def      *RevolveDefinition
	featName string
	tool     *topo.Body // last solid of revolution, exposed so a pattern can replicate it
}

func (r *RevolveFeature) Definition() *RevolveDefinition { return r.def }
func (r *RevolveFeature) Kind() string                   { return "revolve" }

// Operation and ToolBody let a pattern/mirror replicate this feature with the right boolean
// (see [ToolFeature]).
func (r *RevolveFeature) Operation() ops.PartFeatureOperation { return r.def.Operation }
func (r *RevolveFeature) ToolBody() *topo.Body                { return r.tool }

// Recompute resolves the profile, spins it about the axis into a faceted solid of
// revolution, and applies the operation against the running bodies.
func (r *RevolveFeature) Recompute(in Input) (Output, error) {
	prof, err := resolveSingleProfile(r.def.Sketch, r.def.ProfileIndex, "revolve")
	if err != nil {
		return Output{}, err
	}
	axis, err := r.revolveAxis()
	if err != nil {
		return Output{}, err
	}
	r.tool, err = r.buildRevolveTool(prof, axis)
	if err != nil {
		return Output{}, err
	}
	bodies, err := combine(in.Bodies, r.tool, r.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// buildRevolveTool spins the profile into the solid of revolution, resolving the swept
// span from the definition. The actual revolution is the shared [buildRevolveSolid] (the
// assembly-context revolve, #735, reuses it on an assembly-space profile).
func (r *RevolveFeature) buildRevolveTool(prof *sketch.Profile, axis *WorkAxis) (*topo.Body, error) {
	angle, start := revolveSpan(r.def)
	return buildRevolveSolid(prof, r.def.Sketch.Plane(), axis, angle, start, featOr(r.featName, "revolve"))
}

// buildRevolveSolid revolves a profile (already projected onto plane) about axis over the
// swept angle starting at start. Behind OBK_ANALYTIC_CURVES a full 360° revolve of a
// rectilinear profile becomes a TRUE analytic solid (cylinder walls + disk/annulus caps) so
// thread/chamfer/fillet attach to its revolved cylindrical faces (#129); every other case
// (partial angle, an oblique/curved profile edge, or the gate off) keeps the faceted swept
// solid. Booleans re-facet an analytic revolve body on demand (combine → planarized).
func buildRevolveSolid(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis, angle, start float64, feat string) (*topo.Body, error) {
	// Analytic only for a full revolve of a STRAIGHT-edged profile: those edges revolve to exact
	// cylinder/cone/plane faces. A profile with an arc/spline (e.g. a sphere) would have its sampled
	// chords turn into many tiny cone facets — worse than the faceted swept solid — so it stays
	// faceted until curved meridian edges (torus, #129 follow-up) are supported.
	if fullRevolution(angle) && isStraightLoop(prof.OuterLoop()) {
		mer := meridianFromProfile(prof, plane, axis)
		if body, err := brep.SolidOfRevolution(axis.Origin(), axis.Direction().AsVector(), mer, feat); err == nil && body != nil {
			return body, nil
		}
	}
	sections, closed := revolveSectionsFrom(prof, plane, axis, angle, start)
	return sweptSolid(sections, closed, feat)
}

// revolveSpan resolves the total swept angle and its start offset: a
// two-directional revolve spans [-angle2, +angle1]. A combined span reaching a
// full turn collapses to the full revolution (start irrelevant — the solid is
// rotationally complete).
func revolveSpan(def *RevolveDefinition) (total, start float64) {
	a1, a2 := callOrZero(def.Angle), callOrZero(def.Angle2)
	if a2 <= 0 {
		return a1, 0
	}
	if a1 <= 0 { // full + a second direction is still just full
		return 0, 0
	}
	if a1+a2 >= 2*stdmath.Pi-1e-9 {
		return 0, 0
	}
	return a1 + a2, -a2
}

// isStraightLoop reports whether every entity of a profile loop is a straight line segment (so its
// revolution is an exact cylinder/cone/plane), as opposed to an arc/spline/circle.
func isStraightLoop(l sketch.Loop) bool {
	for _, pe := range l.Entities() {
		if _, ok := pe.Entity.(*sketch.Line); !ok {
			return false
		}
	}
	return len(l.Entities()) > 0
}

// fullRevolution reports whether an angle is a complete turn (0 ⇒ full, like revolveSections).
func fullRevolution(angle float64) bool { return angle <= 0 || angle >= 2*stdmath.Pi-1e-9 }

// meridianFromProfile projects the profile's outer loop into the axis's (radius, height) half-plane
// — radius = perpendicular distance from the axis, height = signed distance along it — the
// cross-section brep.SolidOfRevolution revolves.
func meridianFromProfile(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis) []math.Point2 {
	o, a := axis.Origin(), axis.Direction().AsVector()
	poly := modelPolygon(prof, plane)
	out := make([]math.Point2, len(poly))
	for i, p := range poly {
		v := o.VectorTo(p)
		z := v.Dot(a)
		radial := v.Sub(a.Scale(z))
		out[i] = math.P2(radial.Length(), z)
	}
	return out
}

// revolveAxis resolves the axis of revolution: an explicit work axis if set, otherwise the
// sketch's single centerline (Inventor's "revolve about the sketch centerline"). No axis and no
// (or several) centerlines → Sick.
func (r *RevolveFeature) revolveAxis() (*WorkAxis, error) {
	if r.def.Axis != nil {
		return r.def.Axis, nil
	}
	if r.def.AxisCenterline != nil {
		return centerlineAxis(r.def.AxisCenterline, r.def.AxisCenterlineSketch)
	}
	return sketchCenterlineAxis(r.def.Sketch, "revolve")
}

// sketchCenterlineAxis resolves a sketch's single centerline into a revolve axis (Inventor's
// "revolve about the sketch centerline"). No centerline, or more than one, is ambiguous and
// reported as feature health — shared by the part and assembly-context revolves (#735).
func sketchCenterlineAxis(sk *sketch.Sketch, feat string) (*WorkAxis, error) {
	cls := sk.Centerlines()
	if len(cls) == 0 {
		return nil, fmt.Errorf("%s: no axis of revolution (set an axis or add a sketch centerline)", feat)
	}
	if len(cls) > 1 {
		return nil, fmt.Errorf("%s: ambiguous axis — the sketch has multiple centerlines; pick one", feat)
	}
	return centerlineAxis(cls[0], sk)
}

// centerlineAxis turns a centerline line on its sketch into a transient axis of revolution.
func centerlineAxis(line *sketch.Line, sk *sketch.Sketch) (*WorkAxis, error) {
	o, d := line.Axis3D(sk.Plane())
	dir, err := math.UnitVector3FromVector(d)
	if err != nil {
		return nil, errors.New("revolve: the centerline is degenerate")
	}
	return &WorkAxis{origin: o, dir: dir}, nil
}

// revolveSectionsFrom is revolveSections starting at a signed angular offset —
// the two-directional revolve sweeps [start, start+angle] (#313).
func revolveSectionsFrom(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis, angle, start float64) ([][]math.Point3, bool) {
	return revolvePointsAbout(modelPolygon(prof, plane), axis, angle, start)
}

// revolvePointsAbout sweeps a base section's points about the axis from start through angle,
// returning the rotated cross-sections (revolveSegments facets per full turn) and whether the
// revolution is full. Shared by the part revolve and the sheet-metal contour roll.
func revolvePointsAbout(base []math.Point3, axis *WorkAxis, angle, start float64) ([][]math.Point3, bool) {
	full := angle <= 0 || angle >= 2*stdmath.Pi-1e-9
	k, step := revolveSegments, 2*stdmath.Pi/float64(revolveSegments)
	if full {
		start = 0
	} else {
		segs := stdmath.Max(3, stdmath.Round(revolveSegments*angle/(2*stdmath.Pi)))
		k, step = int(segs)+1, angle/segs
	}
	sections := make([][]math.Point3, k)
	for s := 0; s < k; s++ {
		m := math.Rotation4(start+step*float64(s), axis.Direction(), axis.Origin())
		sec := make([]math.Point3, len(base))
		for i, p := range base {
			sec[i] = m.TransformPoint(p)
		}
		sections[s] = sec
	}
	return sections, full
}

// modelPolygon returns a profile's outer-loop polygon mapped into model space.
func modelPolygon(prof *sketch.Profile, plane sketch.Plane) []math.Point3 {
	poly := prof.OuterLoop().Polygon()
	out := make([]math.Point3, len(poly))
	for i, p := range poly {
		out[i] = plane.ToModel(p)
	}
	return out
}

// resolveSingleProfile re-derives one closed region from a sketch, erroring (→ sick)
// when it is missing or open.
func resolveSingleProfile(skt *sketch.Sketch, index int, feat string) (*sketch.Profile, error) {
	all := skt.Profiles()
	if index < 0 || index >= all.Count() {
		return nil, fmt.Errorf("%s: profile %d not found (sketch has %d)", feat, index, all.Count())
	}
	p := all.Item(index)
	if !p.IsClosed() {
		return nil, fmt.Errorf("%s: profile is open (cannot form a solid)", feat)
	}
	return p, nil
}

// featOr returns name if set, else fallback — the lineage feature id for generated
// topology (a unique per-feature name keeps reference keys distinct).
func featOr(name, fallback string) string {
	if name == "" {
		return fallback
	}
	return name
}

// RevolveFeatures adds revolves into the engine.
type RevolveFeatures struct{ engine *PartFeatures }

// NewRevolveFeatures binds the collection to an engine.
func NewRevolveFeatures(engine *PartFeatures) *RevolveFeatures { return &RevolveFeatures{engine} }

// Add adds a revolve of the profile about axis through angle (nil ⇒ full).
func (c *RevolveFeatures) Add(skt *sketch.Sketch, profileIndex int, axis *WorkAxis, angle func() float64, op ops.PartFeatureOperation) *PartFeature {
	def := &RevolveDefinition{Sketch: skt, ProfileIndex: profileIndex, Axis: axis, Angle: angle, Operation: op}
	rf := &RevolveFeature{def: def}
	pf := c.engine.Add(rf)
	pf.SetName(c.engine.UniqueName("Revolution"))
	rf.featName = pf.name
	return pf
}

// AddTwoDirectional adds a revolve sweeping angle forward and angle2 in the
// opposite sense about axis — the reference two-directional revolve (#313).
func (c *RevolveFeatures) AddTwoDirectional(skt *sketch.Sketch, profileIndex int, axis *WorkAxis, angle, angle2 func() float64, op ops.PartFeatureOperation) *PartFeature {
	def := &RevolveDefinition{Sketch: skt, ProfileIndex: profileIndex, Axis: axis, Angle: angle, Angle2: angle2, Operation: op}
	rf := &RevolveFeature{def: def}
	pf := c.engine.Add(rf)
	pf.SetName(c.engine.UniqueName("Revolution"))
	rf.featName = pf.name
	return pf
}

// AddAboutCenterline adds a revolve that spins the profile about the sketch's own centerline
// (the common case: profile + centerline in one sketch). The sketch must hold exactly one.
func (c *RevolveFeatures) AddAboutCenterline(skt *sketch.Sketch, profileIndex int, angle func() float64, op ops.PartFeatureOperation) *PartFeature {
	return c.Add(skt, profileIndex, nil, angle, op)
}

// AddAboutCenterlineLine adds a revolve about a SPECIFIC centerline (on axisSketch) — used when
// the chosen axis isn't simply the profile sketch's lone centerline (several centerlines, or one
// in another sketch).
func (c *RevolveFeatures) AddAboutCenterlineLine(profileSketch *sketch.Sketch, profileIndex int, axisSketch *sketch.Sketch, axisLine *sketch.Line, angle func() float64, op ops.PartFeatureOperation) *PartFeature {
	def := &RevolveDefinition{
		Sketch: profileSketch, ProfileIndex: profileIndex,
		AxisCenterline: axisLine, AxisCenterlineSketch: axisSketch, Angle: angle, Operation: op,
	}
	rf := &RevolveFeature{def: def}
	pf := c.engine.Add(rf)
	pf.SetName(c.engine.UniqueName("Revolution"))
	rf.featName = pf.name
	return pf
}

// CoilDefinition is the recipe for a coil (helical sweep).
type CoilDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int
	Axis         *WorkAxis
	Pitch        func() float64
	Revolutions  func() float64
	// Height is the total axial rise — any TWO of pitch/revolutions/height
	// specify the coil (the reference's pitch+height and revolution+height
	// modes, M08 PBI-096 #316); all three is overdetermined.
	Height    func() float64
	Taper     float64
	Operation ops.PartFeatureOperation
	// Variable-pitch rail + end conditions (M06-F09, #624; coil_variable.go).
	PitchRows []CoilPitchRow
	StartEnd  CoilEndCondition
	EndEnd    CoilEndCondition
}

// CoilFeature sweeps a profile along a helix.
type CoilFeature struct {
	def      *CoilDefinition
	featName string
	tool     *topo.Body // last helical solid, exposed so a pattern can replicate it
}

func (c *CoilFeature) Definition() *CoilDefinition { return c.def }
func (c *CoilFeature) Kind() string                { return "coil" }

// Operation and ToolBody let a pattern/mirror replicate this feature (see [ToolFeature]).
func (c *CoilFeature) Operation() ops.PartFeatureOperation { return c.def.Operation }
func (c *CoilFeature) ToolBody() *topo.Body                { return c.tool }

// Recompute resolves the profile and sweeps it along a helix about the axis (pitch per
// revolution × revolutions) into a faceted solid, then applies the operation.
func (c *CoilFeature) Recompute(in Input) (Output, error) {
	prof, err := resolveSingleProfile(c.def.Sketch, c.def.ProfileIndex, "coil")
	if err != nil {
		return Output{}, err
	}
	if c.def.Axis == nil {
		return Output{}, errors.New("coil: no axis")
	}
	rise, totalTurns, err := coilRail(c.def)
	if err != nil {
		return Output{}, err
	}
	sections := coilSections(prof, c.def.Sketch.Plane(), c.def.Axis, rise, totalTurns, c.def.Taper)
	c.tool, err = sweptSolid(sections, false, featOr(c.featName, "coil"))
	if err != nil {
		return Output{}, err
	}
	bodies, err := combine(in.Bodies, c.tool, c.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// coilSections places the profile along the helix rail: at each step it is
// rotated about the axis by the running angle and translated along the axis
// by the rail's rise at that angle (constant pitch or the M06-F09 pitch
// table — the rise closure carries either).
func coilSections(prof *sketch.Profile, plane sketch.Plane, axis *WorkAxis, rise func(float64) float64, revolutions, taper float64) [][]math.Point3 {
	base := modelPolygon(prof, plane)
	axisVec := axis.Direction().AsVector()
	k := int(stdmath.Max(3, stdmath.Round(revolveSegments*revolutions)))
	total := 2 * stdmath.Pi * revolutions
	sections := make([][]math.Point3, k+1)
	for s := 0; s <= k; s++ {
		angle := total * float64(s) / float64(k)
		rot := math.Rotation4(angle, axis.Direction(), axis.Origin())
		h := rise(angle)
		sec := make([]math.Point3, len(base))
		for i, p := range base {
			q := rot.TransformPoint(p).TranslateBy(axisVec.Scale(math.Scalar(h)))
			sec[i] = coilTaperPoint(q, axis, taper, h)
		}
		sections[s] = sec
	}
	return sections
}

// coilTaperPoint moves a section point radially away from the axis by
// tan(taper)·rise — the tapered coil whose helix radius grows with height
// (M08 PBI-096, #316). Zero taper or zero rise is the identity.
func coilTaperPoint(p math.Point3, axis *WorkAxis, taper, rise float64) math.Point3 {
	if taper == 0 || rise == 0 {
		return p
	}
	a := axis.Direction().AsVector()
	v := axis.Origin().VectorTo(p)
	radial := v.Sub(a.Scale(v.Dot(a)))
	l := float64(radial.Length())
	if l == 0 {
		return p // a point ON the axis has no radial direction to taper along
	}
	off := stdmath.Tan(taper) * rise
	return p.TranslateBy(radial.Scale(math.Scalar(off / l)))
}

// CoilFeatures adds coils into the engine.
type CoilFeatures struct{ engine *PartFeatures }

// NewCoilFeatures binds the collection to an engine.
func NewCoilFeatures(engine *PartFeatures) *CoilFeatures { return &CoilFeatures{engine} }

// AddDefinition adds a coil from a fully-populated definition (height mode,
// taper, variable pitch — #316/#624).
func (c *CoilFeatures) AddDefinition(def *CoilDefinition) *PartFeature {
	cf := &CoilFeature{def: def}
	pf := c.engine.Add(cf)
	pf.SetName(c.engine.UniqueName("Coil"))
	cf.featName = pf.name
	return pf
}

// Add adds a coil of the profile about axis with the given pitch (per revolution),
// number of revolutions, taper, and boolean operation.
func (c *CoilFeatures) Add(skt *sketch.Sketch, profileIndex int, axis *WorkAxis, pitch, revolutions func() float64, taper float64, op ops.PartFeatureOperation) *PartFeature {
	def := &CoilDefinition{
		Sketch: skt, ProfileIndex: profileIndex, Axis: axis,
		Pitch: pitch, Revolutions: revolutions, Taper: taper, Operation: op,
	}
	cf := &CoilFeature{def: def}
	pf := c.engine.Add(cf)
	pf.SetName(c.engine.UniqueName("Coil"))
	cf.featName = pf.name
	return pf
}

// RibDefinition is the recipe for a rib: a thin wall from an open profile (a sketch path).
type RibDefinition struct {
	Sketch       *sketch.Sketch
	ProfileIndex int            // index into the sketch's open paths
	Thickness    func() float64 // wall thickness, centered on the path
	Depth        func() float64 // signed extent along the sketch-plane normal
	// ToNext extends the wall until it fully lands on the existing material
	// (the reference "to-next" rib, M08 PBI-096 #316); Depth then only picks
	// the direction (its sign; nil/0 ⇒ +normal).
	ToNext    bool
	Operation ops.PartFeatureOperation
}

// RibFeature thickens an open sketch profile (a path) into a wall: the path is offset by
// ±Thickness/2 in the sketch plane to a closed band, then extruded Depth along the plane
// normal and combined with the running body. (Inventor's "to-next" bounding against the
// part is a refinement — Depth gives the finite extent today.)
type RibFeature struct {
	def  *RibDefinition
	tool *topo.Body // last rib wall, exposed so a pattern can replicate it
}

func (r *RibFeature) Definition() *RibDefinition { return r.def }
func (r *RibFeature) Kind() string               { return "rib" }

// Operation and ToolBody let a pattern/mirror replicate this feature (see [ToolFeature]).
func (r *RibFeature) Operation() ops.PartFeatureOperation { return r.def.Operation }
func (r *RibFeature) ToolBody() *topo.Body                { return r.tool }

func (r *RibFeature) Recompute(in Input) (Output, error) {
	pts, err := r.pathPoints()
	if err != nil {
		return Output{}, err
	}
	if r.def.Thickness == nil {
		return Output{}, errors.New("rib: thickness must be set")
	}
	t := r.def.Thickness()
	if t <= 0 {
		return Output{}, fmt.Errorf("rib: need positive thickness, got t=%g", t)
	}
	d, err := r.ribDepth(in, pts)
	if err != nil {
		return Output{}, err
	}
	band := ensureCCW2(thickenPath(pts, t))
	r.tool = buildPrism(band, r.def.Sketch.Plane(), orderedSpan(0, d), 0, "rib")
	bodies, err := combine(in.Bodies, r.tool, r.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// ribDepth resolves the wall extent: the explicit signed depth, or with
// ToNext the distance to the FARTHEST first-hit of the existing material
// among the path's points — the wall must fully land, so the deepest ray
// governs (extrude's to-next takes the nearest; a rib that stopped there
// would leave part of the wall hanging).
func (r *RibFeature) ribDepth(in Input, pts []math.Point2) (float64, error) {
	if !r.def.ToNext {
		d := callOrZero(r.def.Depth)
		if d == 0 {
			return 0, errors.New("rib: need a non-zero depth (or set toNext)")
		}
		return d, nil
	}
	sign := 1.0
	if callOrZero(r.def.Depth) < 0 {
		sign = -1
	}
	return ribToNextDepth(in.Bodies, r.def.Sketch.Plane(), pts, sign)
}

// ribToNextDepth ray-casts each path point along the (signed) plane normal
// into the existing material and returns the farthest first-hit as the signed
// depth; a point with no material ahead is a precise error.
func ribToNextDepth(bodies []*topo.Body, plane sketch.Plane, pts []math.Point2, sign float64) (float64, error) {
	if len(bodies) == 0 {
		return 0, errors.New("rib: to-next needs existing material")
	}
	dir := plane.Normal().AsVector().Scale(math.Scalar(sign))
	deepest := 0.0
	for i, p := range pts {
		origin := plane.ToModel(p)
		hit, ok := nearestBodyHit(bodies, origin, dir)
		if !ok {
			return 0, fmt.Errorf("rib: to-next found no material ahead of path point %d (%v)", i, p)
		}
		if hit > deepest {
			deepest = hit
		}
	}
	return sign * deepest, nil
}

// nearestBodyHit is the closest positive ray hit over all bodies.
func nearestBodyHit(bodies []*topo.Body, origin math.Point3, dir math.Vector3) (float64, bool) {
	best, found := stdmath.Inf(1), false
	for _, b := range bodies {
		if _, t, ok := ops.RayCastFaces(b, origin, dir, ops.DefaultQuality()); ok && t > math.DefaultTolerance && t < best {
			best, found = t, true
		}
	}
	return best, found
}

// pathPoints resolves the rib's open profile (a sketch path) to its ordered points.
func (r *RibFeature) pathPoints() ([]math.Point2, error) {
	paths := r.def.Sketch.Paths()
	if r.def.ProfileIndex < 0 || r.def.ProfileIndex >= len(paths) {
		return nil, fmt.Errorf("rib: path index %d out of range (%d open paths)", r.def.ProfileIndex, len(paths))
	}
	pts := paths[r.def.ProfileIndex].Points()
	if len(pts) < 2 {
		return nil, errors.New("rib: the open profile needs at least two points")
	}
	return pts, nil
}

// RibFeatures adds ribs into the engine.
type RibFeatures struct{ engine *PartFeatures }

// NewRibFeatures binds the collection to an engine.
func NewRibFeatures(engine *PartFeatures) *RibFeatures { return &RibFeatures{engine} }

// AddDefinition adds a rib from a fully-populated definition (to-next, #316).
func (c *RibFeatures) AddDefinition(def *RibDefinition) *PartFeature {
	pf := c.engine.Add(&RibFeature{def: def})
	pf.SetName(c.engine.UniqueName("Rib"))
	return pf
}

// Add adds a rib that thickens the sketch's open profile (by index) into a wall of the given
// thickness, extruded the signed depth along the sketch-plane normal, joined to the part.
func (c *RibFeatures) Add(skt *sketch.Sketch, profileIndex int, thickness, depth func() float64, op ops.PartFeatureOperation) *PartFeature {
	def := &RibDefinition{Sketch: skt, ProfileIndex: profileIndex, Thickness: thickness, Depth: depth, Operation: op}
	pf := c.engine.Add(&RibFeature{def: def})
	pf.SetName(c.engine.UniqueName("Rib"))
	return pf
}

// thickenPath offsets a polyline by ±t/2 into a closed band polygon (left side forward,
// right side back) so it can be extruded as a wall.
func thickenPath(pts []math.Point2, t float64) []math.Point2 {
	h := math.Scalar(t / 2)
	n := len(pts)
	band := make([]math.Point2, 0, 2*n)
	for i := 0; i < n; i++ { // left side
		band = append(band, pts[i].TranslateBy(vertexNormal2(pts, i).Scale(h)))
	}
	for i := n - 1; i >= 0; i-- { // right side, reversed → closed loop
		band = append(band, pts[i].TranslateBy(vertexNormal2(pts, i).Scale(-h)))
	}
	return band
}

// vertexNormal2 is the unit in-plane normal at vertex i (averaged perpendicular of the
// adjacent segments).
func vertexNormal2(pts []math.Point2, i int) math.Vector2 {
	var sum math.Vector2
	if i > 0 {
		sum = sum.Add(segNormal2(pts[i-1], pts[i]))
	}
	if i < len(pts)-1 {
		sum = sum.Add(segNormal2(pts[i], pts[i+1]))
	}
	if l := float64(sum.Length()); l > 0 {
		return sum.Scale(math.Scalar(1 / l))
	}
	return math.V2(0, 0)
}

// segNormal2 is the unit left normal of segment a→b.
func segNormal2(a, b math.Point2) math.Vector2 {
	d := a.VectorTo(b)
	n := math.V2(-d.Y, d.X)
	if l := float64(n.Length()); l > 0 {
		return n.Scale(math.Scalar(1 / l))
	}
	return math.V2(0, 0)
}

// ensureCCW2 returns the polygon wound counter-clockwise (positive signed area) so the
// prism builder gives it outward normals.
func ensureCCW2(poly []math.Point2) []math.Point2 {
	var area float64
	for i := range poly {
		j := (i + 1) % len(poly)
		area += float64(poly[i].X*poly[j].Y - poly[j].X*poly[i].Y)
	}
	if area >= 0 {
		return poly
	}
	rev := make([]math.Point2, len(poly))
	for i, p := range poly {
		rev[len(poly)-1-i] = p
	}
	return rev
}
