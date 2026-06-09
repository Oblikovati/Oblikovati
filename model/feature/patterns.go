// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
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
// shared by every pattern kind.
type patternBase struct {
	suppressed map[int]bool
	elements   []PatternElement
}

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
		if p.suppressed[k] {
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

// replicate produces the patterned body state. When the single source feature added or
// removed material (a cut/join/intersect), it re-applies that source's tool at each active
// occurrence with the same boolean — so patterning a hole cuts N holes in one body, and
// patterning a boss unions N bosses into one body. A boolean source whose tool cannot be
// recovered (a deferred feature that built no geometry, or a degenerate delta) replicates
// nothing — copying the whole running body would wrongly multiply it. Only a new-body/base
// source falls back to placing whole-body copies as independent solids.
func (p *patternBase) replicate(in Input, sources []ID, transforms []math.Matrix4, feat string) (Output, error) {
	tool, op, ok := singleSourceTool(in, sources)
	if booleanReplicable(op) {
		if !ok {
			return Output{Bodies: in.Bodies}, nil
		}
		return p.replicateTool(in.Bodies, tool, op, transforms, feat)
	}
	copies, err := p.placeCopies(in.Bodies, transforms, feat)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: appendCopies(in.Bodies, copies)}, nil
}

// replicateTool re-applies the source tool, transformed by each active occurrence, against
// the running result with the source operation (the original sits at occurrence 0 already).
func (p *patternBase) replicateTool(bodies []*topo.Body, tool *topo.Body, op ops.PartFeatureOperation, transforms []math.Matrix4, feat string) (Output, error) {
	if len(bodies) == 0 {
		return Output{Bodies: bodies}, nil
	}
	// Re-facet a curved tool (an extruded-circle ANALYTIC cylinder, #129) into a planar B-rep
	// before the boolean — exactly as combine() does. The planar B-rep boolean hangs/explodes on
	// a full periodic cylinder face, so a circular pattern of a Ø-hole cut used to blow up into
	// tens of thousands of edges (Oblikovati/Oblikovati#129). A faceted tool is left unchanged.
	tool = planarized(tool, feat)
	running := append([]*topo.Body(nil), bodies...)
	last := len(running) - 1
	running[last] = planarized(running[last], feat) // ditto for a curved running target
	for k := 1; k < len(transforms); k++ {
		if p.suppressed[k] {
			continue
		}
		tk, err := ops.TransformBody(tool, transforms[k], copyLineage(feat, k, 0))
		if err != nil {
			return Output{}, err
		}
		res, err := ops.Boolean(op, running[last], tk)
		if err != nil {
			return Output{}, err
		}
		if res != nil && len(res.Faces()) > 0 {
			running[last] = res
		}
	}
	return Output{Bodies: running}, nil
}

// singleSourceTool returns the tool + operation of a lone source feature (the common case);
// it declines multi-source patterns and missing resolvers, leaving them to the copy path.
func singleSourceTool(in Input, sources []ID) (*topo.Body, ops.PartFeatureOperation, bool) {
	if in.SourceTool == nil || len(sources) != 1 {
		return nil, ops.NewBody, false
	}
	return in.SourceTool(sources[0])
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
	transforms := rectTransforms(nx, ny, r.def.StepX, r.def.StepY)
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
	transforms, err := circTransforms(count, callOrZero(c.def.Angle), c.def.AxisPoint, c.def.AxisDir)
	if err != nil {
		return Output{}, err
	}
	return c.replicate(in, c.def.SourceFeatures, transforms, "circ-pattern")
}

// circTransforms returns count occurrence rotations about the axis at angle/count
// increments; element 0 is the identity.
func circTransforms(count int, angle float64, axisPoint math.Point3, axisDir math.Vector3) ([]math.Matrix4, error) {
	dir, err := math.UnitVector3FromVector(axisDir)
	if err != nil {
		return nil, err
	}
	inc := angle / float64(count)
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
