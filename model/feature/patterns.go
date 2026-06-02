// SPDX-License-Identifier: GPL-2.0-only

package feature

// Pattern and mirror features replicate source features/bodies. The per-element
// bookkeeping — element count driven by parameters, with per-element suppression —
// is real here; duplicating the geometry needs the body-transform op that arrives
// with assembly occurrences (M11), so the geometry defers (ErrDeferred → Warning)
// while the element model is fully queryable.

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

// RectangularPatternDefinition replicates source features in a 2D grid.
type RectangularPatternDefinition struct {
	SourceFeatures []ID
	CountX         func() int
	CountY         func() int
}

// RectangularPatternFeature is a 2D grid pattern.
type RectangularPatternFeature struct {
	patternBase
	def *RectangularPatternDefinition
}

func (r *RectangularPatternFeature) Definition() *RectangularPatternDefinition { return r.def }
func (r *RectangularPatternFeature) Kind() string                              { return "rectangular-pattern" }

func (r *RectangularPatternFeature) Recompute(in Input) (Output, error) {
	r.rebuild(callOr1(r.def.CountX) * callOr1(r.def.CountY))
	return Output{Bodies: in.Bodies}, ErrDeferred
}

// CircularPatternDefinition replicates source features around an axis.
type CircularPatternDefinition struct {
	SourceFeatures []ID
	Count          func() int
	Angle          func() float64
}

// CircularPatternFeature is a circular pattern.
type CircularPatternFeature struct {
	patternBase
	def *CircularPatternDefinition
}

func (c *CircularPatternFeature) Definition() *CircularPatternDefinition { return c.def }
func (c *CircularPatternFeature) Kind() string                           { return "circular-pattern" }

func (c *CircularPatternFeature) Recompute(in Input) (Output, error) {
	c.rebuild(callOr1(c.def.Count))
	return Output{Bodies: in.Bodies}, ErrDeferred
}

// SketchDrivenPatternDefinition places one occurrence per sketch point.
type SketchDrivenPatternDefinition struct {
	SourceFeatures []ID
	PointCount     func() int
}

// SketchDrivenPatternFeature places occurrences at sketch points.
type SketchDrivenPatternFeature struct {
	patternBase
	def *SketchDrivenPatternDefinition
}

func (s *SketchDrivenPatternFeature) Definition() *SketchDrivenPatternDefinition { return s.def }
func (s *SketchDrivenPatternFeature) Kind() string                               { return "sketch-driven-pattern" }

func (s *SketchDrivenPatternFeature) Recompute(in Input) (Output, error) {
	s.rebuild(callOr1(s.def.PointCount))
	return Output{Bodies: in.Bodies}, ErrDeferred
}

// MirrorDefinition reflects source features across a plane (one mirrored element).
type MirrorDefinition struct {
	SourceFeatures []ID
	MirrorPlaneKey []byte
}

// MirrorFeature mirrors features across a plane.
type MirrorFeature struct {
	patternBase
	def *MirrorDefinition
}

func (m *MirrorFeature) Definition() *MirrorDefinition { return m.def }
func (m *MirrorFeature) Kind() string                  { return "mirror" }
func (m *MirrorFeature) Recompute(in Input) (Output, error) {
	m.rebuild(1) // a mirror produces a single reflected occurrence
	return Output{Bodies: in.Bodies}, ErrDeferred
}

// PatternFeatures adds pattern/mirror features into the engine.
type PatternFeatures struct{ engine *PartFeatures }

// NewPatternFeatures binds the collection to an engine.
func NewPatternFeatures(engine *PartFeatures) *PatternFeatures { return &PatternFeatures{engine} }

// AddRectangular adds a grid pattern of the source features.
func (c *PatternFeatures) AddRectangular(source []ID, countX, countY func() int) *RectangularPatternFeature {
	f := &RectangularPatternFeature{def: &RectangularPatternDefinition{SourceFeatures: source, CountX: countX, CountY: countY}}
	c.engine.Add(f, source...)
	return f
}

// AddCircular adds a circular pattern.
func (c *PatternFeatures) AddCircular(source []ID, count func() int, angle func() float64) *CircularPatternFeature {
	f := &CircularPatternFeature{def: &CircularPatternDefinition{SourceFeatures: source, Count: count, Angle: angle}}
	c.engine.Add(f, source...)
	return f
}

// AddSketchDriven adds a sketch-driven pattern.
func (c *PatternFeatures) AddSketchDriven(source []ID, pointCount func() int) *SketchDrivenPatternFeature {
	f := &SketchDrivenPatternFeature{def: &SketchDrivenPatternDefinition{SourceFeatures: source, PointCount: pointCount}}
	c.engine.Add(f, source...)
	return f
}

// AddMirror adds a mirror across a plane (by reference key).
func (c *PatternFeatures) AddMirror(source []ID, planeKey []byte) *MirrorFeature {
	f := &MirrorFeature{def: &MirrorDefinition{SourceFeatures: source, MirrorPlaneKey: planeKey}}
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
