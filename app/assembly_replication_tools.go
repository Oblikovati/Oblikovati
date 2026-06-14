// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/occurrence"
)

// The assembly replication tools (M11, #765) replicate placed component occurrences picked in
// the browser (OccurrenceHandle) into real placed copies: a rectangular/circular pattern of a
// seed, a mirror across a plane, and an independent copy. They gather the source occurrence(s) +
// parameters and apply the occurrence-level op on Commit; the geometry lives in
// model/occurrence and is tested there, so these are interaction shells driven headlessly here.
// Parameters are edited through the generic tool-param dialog (Params), so no head dialog is
// needed — the same seam the part pattern tools use.

// occurrenceSelectTool collects the source occurrences a replication op acts on. It seeds itself
// from any pre-selected occurrences on Start (select-then-command), and accepts further picks
// while active.
type occurrenceSelectTool struct {
	sources []*occurrence.Occurrence
}

// Start seeds the sources from the current selection so "select a component, click Pattern" works
// without re-picking.
func (t *occurrenceSelectTool) Start(s *Session) {
	for _, it := range s.Selection().Items() {
		if h, ok := it.(OccurrenceHandle); ok && h.Occurrence != nil {
			t.sources = append(t.sources, h.Occurrence)
		}
	}
}

// Pick collects an occurrence pick (an OccurrenceHandle from the browser or, once #769 lands, the
// viewport).
func (t *occurrenceSelectTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(OccurrenceHandle); ok && h.Occurrence != nil {
		t.sources = append(t.sources, h.Occurrence)
	}
}

func (t *occurrenceSelectTool) Cancel(*Session) { t.sources = nil }

// assembly resolves the active assembly and asserts a source was selected, erroring with op's
// name otherwise.
func (t *occurrenceSelectTool) assembly(s *Session, op string) (*compdef.AssemblyComponentDefinition, error) {
	asm, err := activeAssembly(s)
	if err != nil {
		return nil, err
	}
	if len(t.sources) == 0 {
		return nil, errors.New(op + ": select a component first")
	}
	return asm, nil
}

// --- Rectangular pattern --------------------------------------------------

// AssemblyRectPatternTool patterns the seed occurrence on a grid: count1 columns along +X spaced
// spacing1, count2 rows along +Y spaced spacing2.
type AssemblyRectPatternTool struct {
	occurrenceSelectTool
	count1, count2     int
	spacing1, spacing2 float64
}

// NewAssemblyRectPatternTool returns a 2×1 grid at 2 cm spacing, ready to edit.
func NewAssemblyRectPatternTool() *AssemblyRectPatternTool {
	return &AssemblyRectPatternTool{count1: 2, count2: 1, spacing1: 2, spacing2: 2}
}
func (t *AssemblyRectPatternTool) Name() string { return "Rectangular Pattern" }
func (t *AssemblyRectPatternTool) Prompt(*Session) string {
	return "Select a component, set the counts and spacing, then OK."
}

// CanCommit needs a seed and a grid that produces at least one copy beyond the seed.
func (t *AssemblyRectPatternTool) CanCommit() bool {
	return len(t.sources) > 0 && t.count1 >= 1 && t.count2 >= 1 && t.count1*t.count2 > 1
}

func (t *AssemblyRectPatternTool) Commit(s *Session) error {
	asm, err := t.assembly(s, "rectangular pattern")
	if err != nil {
		return err
	}
	dir1, _ := math.NewUnitVector3(1, 0, 0) // axis literals are unit by construction; the error is unreachable
	dir2, _ := math.NewUnitVector3(0, 1, 0)
	seed := t.sources[0]
	arr := occurrence.RectangularArrangement{
		Dir1: dir1, Spacing1: math.Scalar(t.spacing1), Count1: t.count1,
		Dir2: dir2, Spacing2: math.Scalar(t.spacing2), Count2: t.count2,
	}
	occurrence.PatternComponents(asm.Occurrences(), seed, occurrence.NewOccurrencePattern(seed.Definition(), seed.Transform(), arr))
	s.recordEdit(asm, "Rectangular Pattern")
	return nil
}

func (t *AssemblyRectPatternTool) Params() ToolParams {
	return ToolParams{
		Ints: []IntParam{
			{"Count X", func() int { return t.count1 }, func(n int) { t.count1 = n }},
			{"Count Y", func() int { return t.count2 }, func(n int) { t.count2 = n }},
		},
		Floats: []FloatParam{
			{"Spacing X", func() float64 { return t.spacing1 }, func(v float64) { t.spacing1 = v }},
			{"Spacing Y", func() float64 { return t.spacing2 }, func(v float64) { t.spacing2 = v }},
		},
	}
}

// --- Circular pattern -----------------------------------------------------

// AssemblyCircPatternTool patterns the seed occurrence around the Z axis: count copies evenly
// spread over totalAngle (radians); a full ring uses 2π.
type AssemblyCircPatternTool struct {
	occurrenceSelectTool
	count      int
	totalAngle float64
}

// NewAssemblyCircPatternTool returns a 4-up full ring, ready to edit.
func NewAssemblyCircPatternTool() *AssemblyCircPatternTool {
	return &AssemblyCircPatternTool{count: 4, totalAngle: 2 * 3.141592653589793} // full ring (2π)
}
func (t *AssemblyCircPatternTool) Name() string { return "Circular Pattern" }
func (t *AssemblyCircPatternTool) Prompt(*Session) string {
	return "Select a component, set the count and angle, then OK."
}
func (t *AssemblyCircPatternTool) CanCommit() bool { return len(t.sources) > 0 && t.count >= 2 }

func (t *AssemblyCircPatternTool) Commit(s *Session) error {
	asm, err := t.assembly(s, "circular pattern")
	if err != nil {
		return err
	}
	axis, _ := math.NewUnitVector3(0, 0, 1)
	seed := t.sources[0]
	arr := occurrence.CircularArrangement{
		Origin: math.P3(0, 0, 0), Axis: axis,
		Step:  math.Scalar(t.totalAngle / float64(t.count)), // evenly distribute count over the sweep
		Count: t.count,
	}
	occurrence.PatternComponents(asm.Occurrences(), seed, occurrence.NewOccurrencePattern(seed.Definition(), seed.Transform(), arr))
	s.recordEdit(asm, "Circular Pattern")
	return nil
}

func (t *AssemblyCircPatternTool) Params() ToolParams {
	return ToolParams{
		Ints: []IntParam{{"Count", func() int { return t.count }, func(n int) { t.count = n }}},
		Floats: []FloatParam{{
			"Angle (deg)", func() float64 { return degFromRad(t.totalAngle) },
			func(d float64) { t.totalAngle = radFromDeg(d) },
		}},
	}
}

// --- Mirror ---------------------------------------------------------------

// AssemblyMirrorTool adds a mirror of each selected occurrence across the plane through the origin
// with the given normal (default the YZ plane, normal +X). Each mirror shares its source's
// component, handed by the reflection transform (an independent mirrored part is M11-F06).
type AssemblyMirrorTool struct {
	occurrenceSelectTool
	normalX, normalY, normalZ float64
}

// NewAssemblyMirrorTool mirrors across the YZ plane by default.
func NewAssemblyMirrorTool() *AssemblyMirrorTool {
	return &AssemblyMirrorTool{normalX: 1}
}
func (t *AssemblyMirrorTool) Name() string { return "Mirror" }
func (t *AssemblyMirrorTool) Prompt(*Session) string {
	return "Select components, set the mirror-plane normal, then OK."
}

func (t *AssemblyMirrorTool) CanCommit() bool {
	return len(t.sources) > 0 && (t.normalX != 0 || t.normalY != 0 || t.normalZ != 0)
}

func (t *AssemblyMirrorTool) Commit(s *Session) error {
	asm, err := t.assembly(s, "mirror")
	if err != nil {
		return err
	}
	normal, err := math.NewUnitVector3(t.normalX, t.normalY, t.normalZ)
	if err != nil {
		return fmt.Errorf("mirror: normal (%g, %g, %g) is not a direction: %w", t.normalX, t.normalY, t.normalZ, err)
	}
	occurrence.MirrorComponents(asm.Occurrences(), t.sources, math.P3(0, 0, 0), normal)
	s.recordEdit(asm, "Mirror Components")
	return nil
}

func (t *AssemblyMirrorTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{
		{"Normal X", func() float64 { return t.normalX }, func(v float64) { t.normalX = v }},
		{"Normal Y", func() float64 { return t.normalY }, func(v float64) { t.normalY = v }},
		{"Normal Z", func() float64 { return t.normalZ }, func(v float64) { t.normalZ = v }},
	}}
}

// --- Copy -----------------------------------------------------------------

// CopyComponents adds an independent copy of each currently-selected occurrence — same component
// and placement, a new unlinked instance. Unlike Pattern/Mirror it needs no parameters, so it
// runs immediately on the selection (Inventor's Copy), erroring when nothing is selected.
func (s *Session) CopyComponents() error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	sources := selectedOccurrences(s)
	if len(sources) == 0 {
		return errors.New("copy: select a component to copy first")
	}
	occurrence.CopyComponents(asm.Occurrences(), sources)
	s.recordEdit(asm, "Copy Components")
	return nil
}

// selectedOccurrences returns the occurrences in the current selection set, in selection order.
func selectedOccurrences(s *Session) []*occurrence.Occurrence {
	var out []*occurrence.Occurrence
	for _, it := range s.Selection().Items() {
		if h, ok := it.(OccurrenceHandle); ok && h.Occurrence != nil {
			out = append(out, h.Occurrence)
		}
	}
	return out
}
