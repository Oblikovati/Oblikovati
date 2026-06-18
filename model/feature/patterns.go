// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Pattern and mirror features replicate the running solid into placed copies. Each
// occurrence carries a rigid transform (a grid step, a rotation about an axis, a
// sketch-point offset, or a plane reflection); recompute makes a real copy of every
// running body per active occurrence via ops.TransformBody (with a distinct lineage
// so each copy has its own reference keys) and appends them as placed solids.
//
// The element bookkeeping — count driven by parameters, with per-element suppression
// — is preserved. Copies are appended as separate bodies rather than boolean-unioned
// into one (the general intersecting boolean is M20·F01); for the common
// disjoint-pattern case each placed copy is an independent validated solid. Patterning
// only the source feature's contribution (vs. the whole running solid) needs
// per-feature geometry provenance and is a follow-up.

// PatternElement is one occurrence in a pattern, individually suppressible.
type PatternElement struct {
	Index      int
	Suppressed bool
}

// patternBase holds the element list and the persistent per-index suppression set
// shared by every pattern kind. clipped is the transient set of occurrences a boundary
// excludes this recompute (computed, not user-set; see [PatternOptions]).
type patternBase struct {
	suppressed map[int]bool
	clipped    map[int]bool
	elements   []PatternElement
}

// skip reports whether occurrence k is left out of the result — either suppressed by the
// user or clipped by a pattern boundary.
func (p *patternBase) skip(k int) bool { return p.suppressed[k] || p.clipped[k] }

// rebuild regenerates the element list for count occurrences, preserving any
// per-element suppression by index.
func (p *patternBase) rebuild(count int) {
	if count < 1 {
		count = 1
	}
	if p.suppressed == nil {
		p.suppressed = map[int]bool{}
	}
	p.elements = make([]PatternElement, count)
	for i := range p.elements {
		p.elements[i] = PatternElement{Index: i, Suppressed: p.suppressed[i]}
	}
}

// ElementCount returns the number of pattern occurrences (including suppressed).
func (p *patternBase) ElementCount() int { return len(p.elements) }

// Elements returns a snapshot of the occurrences.
func (p *patternBase) Elements() []PatternElement {
	return append([]PatternElement(nil), p.elements...)
}

// ActiveCount returns the number of non-suppressed occurrences.
func (p *patternBase) ActiveCount() int {
	n := 0
	for _, e := range p.elements {
		if !e.Suppressed {
			n++
		}
	}
	return n
}

// SetElementSuppressed toggles suppression of occurrence i (effective next recompute).
func (p *patternBase) SetElementSuppressed(i int, s bool) {
	if p.suppressed == nil {
		p.suppressed = map[int]bool{}
	}
	p.suppressed[i] = s
}

// placeCopies makes a real copy of every running body for each active occurrence
// (skipping element 0, the original, and any suppressed element), transformed by that
// occurrence's matrix. Each copy gets a distinct lineage so its reference keys do not
// collide with the source's or another copy's.
func (p *patternBase) placeCopies(bodies []*topo.Body, transforms []math.Matrix4, feat string) ([]*topo.Body, error) {
	var copies []*topo.Body
	for k := 1; k < len(transforms); k++ {
		if p.skip(k) {
			continue
		}
		for bi, b := range bodies {
			c, err := ops.TransformBody(b, transforms[k], copyLineage(feat, k, bi))
			if err != nil {
				return nil, err
			}
			copies = append(copies, c)
		}
	}
	return copies, nil
}

// copyLineage returns a lineage map that tags a copy with its occurrence and source
// body indices, keeping every placed copy's keys distinct and stable across recompute.
func copyLineage(feat string, element, bodyIndex int) func(topo.Lineage) topo.Lineage {
	return func(l topo.Lineage) topo.Lineage {
		return topo.NewLineage(append(l.Tokens(),
			topo.Tok(feat, "occurrence", element), topo.Tok(feat, "seed", bodyIndex))...)
	}
}

// appendCopies returns the running bodies with the placed copies appended.
func appendCopies(bodies, copies []*topo.Body) []*topo.Body {
	out := append([]*topo.Body(nil), bodies...)
	return append(out, copies...)
}

// groupTool pairs a resolved source-feature tool body with its boolean operation, so a pattern
// can re-apply a whole GROUP of material features (e.g. a join boss + a hole) at each occurrence.
type groupTool struct {
	body *topo.Body
	op   ops.PartFeatureOperation
}

// replicate produces the patterned body state. When every source feature added or removed
// material (a cut/join/intersect), it re-applies those sources' tools — in feature order — at
// each active occurrence with their own booleans, so patterning a hole cuts N holes in one body,
// patterning a boss unions N bosses into one body, and patterning a join+hole GROUP places N
// connected bosses each holed (Oblikovati/Oblikovati#128) rather than scattering whole-body
// copies. A boolean source whose tool cannot be recovered (a deferred feature that built no
// geometry, or a degenerate delta) replicates nothing — copying the whole running body would
// wrongly multiply it. Only a new-body/base source falls back to placing whole-body copies.
func (p *patternBase) replicate(in Input, sources []ID, transforms []math.Matrix4, feat string) (Output, error) {
	tools, boolean := resolveGroup(in, sources)
	if boolean {
		if len(tools) == 0 {
			return Output{Bodies: in.Bodies}, nil
		}
		return p.replicateTools(in.Bodies, tools, transforms, feat)
	}
	copies, err := p.placeCopies(in.Bodies, transforms, feat)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: appendCopies(in.Bodies, copies)}, nil
}

// replicateTools re-applies a group of source tools, transformed by each active occurrence,
// against the running result (the original sits at occurrence 0 already). Applying the tools in
// source order at every grid cell keeps a multi-feature group connected (#128).
func (p *patternBase) replicateTools(bodies []*topo.Body, tools []groupTool, transforms []math.Matrix4, feat string) (Output, error) {
	if len(bodies) == 0 {
		return Output{Bodies: bodies}, nil
	}
	// Re-facet a curved tool (an extruded-circle ANALYTIC cylinder, #129) into a planar B-rep
	// before the boolean — exactly as combine() does. The planar B-rep boolean hangs/explodes on
	// a full periodic cylinder face, so a circular pattern of a Ø-hole cut used to blow up into
	// tens of thousands of edges (#129). A faceted tool is left unchanged.
	for i := range tools {
		tools[i].body = planarized(tools[i].body, feat)
	}
	running := append([]*topo.Body(nil), bodies...)
	last := len(running) - 1
	running[last] = planarized(running[last], feat) // ditto for a curved running target
	for k := 1; k < len(transforms); k++ {
		if p.skip(k) {
			continue
		}
		next, err := applyGroupAt(running[last], tools, transforms[k], feat, k)
		if err != nil {
			return Output{}, err
		}
		running[last] = next
	}
	return Output{Bodies: running}, nil
}

// applyGroupAt booleans every tool of the group at occurrence k against the running body, in
// source order, transformed into place with a per-tool lineage so reference keys stay distinct.
func applyGroupAt(running *topo.Body, tools []groupTool, xf math.Matrix4, feat string, k int) (*topo.Body, error) {
	for bi, t := range tools {
		tk, err := ops.TransformBody(t.body, xf, copyLineage(feat, k, bi))
		if err != nil {
			return nil, err
		}
		res, err := ops.Boolean(t.op, running, tk)
		if err != nil {
			return nil, err
		}
		if res != nil && len(res.Faces()) > 0 {
			running = res
		}
	}
	return running, nil
}

// resolveGroup classifies the pattern's source features. boolean is true when every source
// applies a boolean op (cut/join/intersect) — a material group the pattern re-applies at each
// occurrence; it is false when any source is a new-body placement, so the caller copies whole
// bodies. When a boolean source's tool cannot be resolved, tools is empty but boolean stays true,
// so the caller no-ops the pattern rather than multiplying whole bodies.
func resolveGroup(in Input, sources []ID) (tools []groupTool, boolean bool) {
	if in.SourceTool == nil || len(sources) == 0 {
		return nil, false
	}
	tools = make([]groupTool, 0, len(sources))
	for _, id := range sources {
		tool, op, ok := in.SourceTool(id)
		if !booleanReplicable(op) {
			return nil, false
		}
		if !ok {
			return nil, true
		}
		tools = append(tools, groupTool{body: tool, op: op})
	}
	return tools, true
}

// booleanReplicable reports whether an operation is one the pattern re-applies as a boolean
// (a material change), as opposed to a new-body placement.
func booleanReplicable(op ops.PartFeatureOperation) bool {
	return op == ops.Cut || op == ops.Join || op == ops.Intersect
}

// RectangularPatternDefinition replicates the running solid in a 2D grid. StepX/StepY
// are the per-occurrence offset vectors (direction × spacing); element (ix,iy) sits at
// StepX·ix + StepY·iy, with (0,0) the original.
type RectangularPatternDefinition struct {
	SourceFeatures []ID
	CountX         func() int
	CountY         func() int
	StepX          math.Vector3
	StepY          math.Vector3
	Options        PatternOptions // spacing/compute/orientation/positioning + boundary (M20-F18)
}

// RectangularPatternFeature is a 2D grid pattern.
type RectangularPatternFeature struct {
	patternBase
	def *RectangularPatternDefinition
}

func (r *RectangularPatternFeature) Definition() *RectangularPatternDefinition { return r.def }
func (r *RectangularPatternFeature) Kind() string                              { return "rectangular-pattern" }

func (r *RectangularPatternFeature) Recompute(in Input) (Output, error) {
	nx, ny := callOr1(r.def.CountX), callOr1(r.def.CountY)
	r.rebuild(nx * ny)
	stepX := r.def.Options.rectStep(r.def.StepX, nx)
	stepY := r.def.Options.rectStep(r.def.StepY, ny)
	transforms := rectTransforms(nx, ny, stepX, stepY)
	seed, ok := seedCentre(in.Bodies)
	r.clipped = r.def.Options.clippedOccurrences(transforms, seed, ok)
	return r.replicate(in, r.def.SourceFeatures, transforms, "rect-pattern")
}

// rectTransforms returns the grid of occurrence transforms in row-major (ix + iy·nx)
// order; element 0 is the identity (the original occurrence).
func rectTransforms(nx, ny int, stepX, stepY math.Vector3) []math.Matrix4 {
	out := make([]math.Matrix4, 0, nx*ny)
	for iy := 0; iy < ny; iy++ {
		for ix := 0; ix < nx; ix++ {
			offset := stepX.Scale(float64(ix)).Add(stepY.Scale(float64(iy)))
			out = append(out, math.Translation4(offset))
		}
	}
	return out
}

// CircularPatternDefinition replicates the running solid about an axis; Count
// occurrences are spread at Angle/Count increments (Angle is the total sweep).
type CircularPatternDefinition struct {
	SourceFeatures []ID
	Count          func() int
	Angle          func() float64
	AxisPoint      math.Point3
	AxisDir        math.Vector3
	Options        PatternOptions // spacing/compute/orientation/positioning + boundary (M20-F18)
}

// CircularPatternFeature is a circular pattern.
type CircularPatternFeature struct {
	patternBase
	def *CircularPatternDefinition
}

func (c *CircularPatternFeature) Definition() *CircularPatternDefinition { return c.def }
func (c *CircularPatternFeature) Kind() string                           { return "circular-pattern" }

func (c *CircularPatternFeature) Recompute(in Input) (Output, error) {
	count := callOr1(c.def.Count)
	c.rebuild(count)
	inc := c.def.Options.circIncrement(callOrZero(c.def.Angle), count)
	transforms, err := circTransforms(count, inc, c.def.AxisPoint, c.def.AxisDir)
	if err != nil {
		return Output{}, err
	}
	seed, ok := seedCentre(in.Bodies)
	c.clipped = c.def.Options.clippedOccurrences(transforms, seed, ok)
	return c.replicate(in, c.def.SourceFeatures, transforms, "circ-pattern")
}

// circTransforms returns count occurrence rotations about the axis stepping by inc radians
// per occurrence; element 0 is the identity. The increment is chosen by the spacing type
// ([PatternOptions.circIncrement]).
func circTransforms(count int, inc float64, axisPoint math.Point3, axisDir math.Vector3) ([]math.Matrix4, error) {
	dir, err := math.UnitVector3FromVector(axisDir)
	if err != nil {
		return nil, err
	}
	out := make([]math.Matrix4, count)
	for k := 0; k < count; k++ {
		out[k] = math.Rotation4(inc*float64(k), dir, axisPoint)
	}
	return out, nil
}

// SketchDrivenPatternDefinition places one occurrence per sketch point. Points are the
// occurrence positions; element k is offset by Points[k] − Points[0] from the source.
type SketchDrivenPatternDefinition struct {
	SourceFeatures []ID
	Points         func() []math.Point3
}

// SketchDrivenPatternFeature places occurrences at sketch points.
type SketchDrivenPatternFeature struct {
	patternBase
	def *SketchDrivenPatternDefinition
}

func (s *SketchDrivenPatternFeature) Definition() *SketchDrivenPatternDefinition { return s.def }
func (s *SketchDrivenPatternFeature) Kind() string                               { return "sketch-driven-pattern" }

func (s *SketchDrivenPatternFeature) Recompute(in Input) (Output, error) {
	points := callPoints(s.def.Points)
	s.rebuild(len(points))
	transforms := sketchTransforms(points)
	return s.replicate(in, s.def.SourceFeatures, transforms, "sketch-pattern")
}

// sketchTransforms returns a translation per sketch point relative to the first point
// (element 0, the source location → identity).
func sketchTransforms(points []math.Point3) []math.Matrix4 {
	if len(points) == 0 {
		return []math.Matrix4{math.Identity4()}
	}
	out := make([]math.Matrix4, len(points))
	for k, p := range points {
		out[k] = math.Translation4(points[0].VectorTo(p))
	}
	return out
}

// MirrorDefinition reflects the running solid across a plane (one mirrored occurrence).
// MirrorPlaneKey identifies the plane for persistence/identity; Origin/Normal give its
// geometry for the reflection.
type MirrorDefinition struct {
	SourceFeatures []ID
	MirrorPlaneKey []byte
	Origin         math.Point3
	Normal         math.Vector3
}

// MirrorFeature mirrors the running solid across a plane.
type MirrorFeature struct {
	patternBase
	def *MirrorDefinition
}

func (m *MirrorFeature) Definition() *MirrorDefinition { return m.def }
func (m *MirrorFeature) Kind() string                  { return "mirror" }

func (m *MirrorFeature) Recompute(in Input) (Output, error) {
	m.rebuild(1) // a mirror produces a single reflected occurrence
	normal, err := math.UnitVector3FromVector(m.def.Normal)
	if err != nil {
		return Output{}, err
	}
	transforms := []math.Matrix4{math.Identity4(), math.Reflection4(m.def.Origin, normal)}
	return m.replicate(in, m.def.SourceFeatures, transforms, "mirror")
}

// PatternFeatures adds pattern/mirror features into the engine.
type PatternFeatures struct{ engine *PartFeatures }

// NewPatternFeatures binds the collection to an engine.
func NewPatternFeatures(engine *PartFeatures) *PatternFeatures { return &PatternFeatures{engine} }

// AddRectangular adds a grid pattern stepping by stepX/stepY per column/row.
func (c *PatternFeatures) AddRectangular(source []ID, countX, countY func() int, stepX, stepY math.Vector3) *RectangularPatternFeature {
	f := &RectangularPatternFeature{def: &RectangularPatternDefinition{
		SourceFeatures: source, CountX: countX, CountY: countY, StepX: stepX, StepY: stepY,
	}}
	c.engine.Add(f, source...)
	return f
}

// AddCircular adds a circular pattern of count occurrences over the total angle about
// the axis (axisPoint, axisDir).
func (c *PatternFeatures) AddCircular(source []ID, count func() int, angle func() float64, axisPoint math.Point3, axisDir math.Vector3) *CircularPatternFeature {
	f := &CircularPatternFeature{def: &CircularPatternDefinition{
		SourceFeatures: source, Count: count, Angle: angle, AxisPoint: axisPoint, AxisDir: axisDir,
	}}
	c.engine.Add(f, source...)
	return f
}

// AddSketchDriven adds a pattern placing an occurrence at each sketch point.
func (c *PatternFeatures) AddSketchDriven(source []ID, points func() []math.Point3) *SketchDrivenPatternFeature {
	f := &SketchDrivenPatternFeature{def: &SketchDrivenPatternDefinition{SourceFeatures: source, Points: points}}
	c.engine.Add(f, source...)
	return f
}

// AddMirror adds a mirror across the plane (origin, normal), identified by planeKey.
func (c *PatternFeatures) AddMirror(source []ID, planeKey []byte, origin math.Point3, normal math.Vector3) *MirrorFeature {
	f := &MirrorFeature{def: &MirrorDefinition{SourceFeatures: source, MirrorPlaneKey: planeKey, Origin: origin, Normal: normal}}
	c.engine.Add(f, source...)
	return f
}

// callOr1 evaluates a count closure, defaulting to 1 when nil.
func callOr1(f func() int) int {
	if f == nil {
		return 1
	}
	return f()
}

// callOrZero evaluates a float closure, defaulting to 0 when nil.
func callOrZero(f func() float64) float64 {
	if f == nil {
		return 0
	}
	return f()
}

// callPoints evaluates a points closure, defaulting to a single origin when nil.
func callPoints(f func() []math.Point3) []math.Point3 {
	if f == nil {
		return []math.Point3{math.P3(0, 0, 0)}
	}
	return f()
}
