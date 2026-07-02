// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// The feature-level Pattern and Mirror tools replicate one or more source features (picked
// in the browser or viewport as FeatureHandles) into real placed copies. They gather the
// source feature ids + parameters and add the pattern/mirror feature on OK; the geometry
// lives in model/feature and is tested there, so these are interaction shells driven
// headlessly here. Parameters are edited through the generic tool-param dialog.

// featureSelectTool collects the source features (by id) the pattern/mirror replicates.
type featureSelectTool struct {
	dialogTool
	sources []feature.ID
	added   *feature.PartFeature
}

// Pick collects a feature pick (a FeatureHandle from the browser or graphics).
func (t *featureSelectTool) Pick(_ *Session, sel Selectable) {
	if h, ok := sel.(FeatureHandle); ok && h.Feature != nil {
		t.sources = append(t.sources, h.Feature.ID())
	}
}

func (t *featureSelectTool) Cancel(*Session) { t.sources = nil }

// patternFeatures resolves the active part and its pattern-feature collection, erroring on
// no part or no selected source.
func (t *featureSelectTool) patternFeatures(s *Session, op string) (*feature.PatternFeatures, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	if len(t.sources) == 0 {
		return nil, errors.New(op + ": select a feature to pattern first")
	}
	return feature.NewPatternFeatures(part.Features()), nil
}

// draftPattern builds a pattern tool's feature into a scratch engine for the commit gate
// and preview (#1626). add runs the tool's exact commit-time construction; the Pattern
// Add* methods return the raw feature, so the wrapper is read back off the engine.
func draftPattern(add func(*feature.PatternFeatures)) (feature.Feature, bool) {
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		add(feature.NewPatternFeatures(fs))
		return lastEngineFeature(fs), nil
	})
}

// finishPattern recomputes the part, records the edit and reports a sick pattern.
func (t *featureSelectTool) finishPattern(s *Session, added *feature.PartFeature, label string) error {
	part, _ := activePart(s)
	t.added = added
	part.Recompute()
	s.recordEdit(part, label)
	if !added.Health().OK() {
		return errors.New(label + ": " + added.Health().Reason)
	}
	return nil
}

// --- Rectangular ----------------------------------------------------------

// FeatureRectPatternTool patterns the source features on a grid (countX along +X by
// spacingX, countY along +Y by spacingY).
type FeatureRectPatternTool struct {
	featureSelectTool
	countX, countY     int
	spacingX, spacingY float64
}

func NewFeatureRectPatternTool() *FeatureRectPatternTool {
	return &FeatureRectPatternTool{countX: 2, countY: 1, spacingX: 2, spacingY: 2}
}
func (t *FeatureRectPatternTool) Name() string { return "Rectangular Pattern" }
func (t *FeatureRectPatternTool) Prompt(*Session) string {
	return "Select features, set the counts and spacing, then OK."
}

func (t *FeatureRectPatternTool) CanCommit() bool {
	return len(t.sources) > 0 && t.countX >= 1 && t.countY >= 1 && t.countX*t.countY > 1
}

func (t *FeatureRectPatternTool) Commit(s *Session) error {
	pats, err := t.patternFeatures(s, "rectangular pattern")
	if err != nil {
		return err
	}
	t.addRectangular(pats)
	return t.finishPattern(s, lastPartFeature(s), "Rectangular Pattern")
}

// addRectangular builds the grid pattern into pats — the shared constructor used by both
// Commit (the part's engine) and DraftFeature (a scratch engine), so the two cannot drift.
func (t *FeatureRectPatternTool) addRectangular(pats *feature.PatternFeatures) {
	cx, cy := t.countX, t.countY
	pats.AddRectangular(t.sources, func() int { return cx }, func() int { return cy },
		math.V3(math.Scalar(t.spacingX), 0, 0), math.V3(0, math.Scalar(t.spacingY), 0))
}

// DraftFeature implements [PartFeatureTool] (#1626): the rectangular pattern it would
// commit, built into a scratch engine so the commit gate and preview can evaluate it
// without touching the part.
func (t *FeatureRectPatternTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftPattern(t.addRectangular)
}

func (t *FeatureRectPatternTool) Params() ToolParams {
	return ToolParams{
		Ints: []IntParam{
			{"Count X", func() int { return t.countX }, func(n int) { t.countX = n }},
			{"Count Y", func() int { return t.countY }, func(n int) { t.countY = n }},
		},
		Floats: []FloatParam{
			{"Spacing X", func() float64 { return t.spacingX }, func(v float64) { t.spacingX = v }},
			{"Spacing Y", func() float64 { return t.spacingY }, func(v float64) { t.spacingY = v }},
		},
	}
}

// --- Circular -------------------------------------------------------------

// FeatureCircPatternTool patterns the source features around the Z axis: count copies over
// totalAngle (radians).
type FeatureCircPatternTool struct {
	featureSelectTool
	count      int
	totalAngle float64
}

func NewFeatureCircPatternTool() *FeatureCircPatternTool {
	return &FeatureCircPatternTool{count: 4, totalAngle: FullTurn}
}
func (t *FeatureCircPatternTool) Name() string { return "Circular Pattern" }
func (t *FeatureCircPatternTool) Prompt(*Session) string {
	return "Select features, set the count and angle, then OK."
}
func (t *FeatureCircPatternTool) CanCommit() bool { return len(t.sources) > 0 && t.count >= 2 }

func (t *FeatureCircPatternTool) Commit(s *Session) error {
	pats, err := t.patternFeatures(s, "circular pattern")
	if err != nil {
		return err
	}
	t.addCircular(pats)
	return t.finishPattern(s, lastPartFeature(s), "Circular Pattern")
}

// addCircular builds the circular pattern into pats — the shared constructor used by both
// Commit (the part's engine) and DraftFeature (a scratch engine), so the two cannot drift.
func (t *FeatureCircPatternTool) addCircular(pats *feature.PatternFeatures) {
	n := t.count
	a := t.totalAngle
	pats.AddCircular(t.sources, func() int { return n }, func() float64 { return a },
		math.P3(0, 0, 0), math.V3(0, 0, 1))
}

// DraftFeature implements [PartFeatureTool] (#1626): the circular pattern it would
// commit, built into a scratch engine so the commit gate and preview can evaluate it
// without touching the part.
func (t *FeatureCircPatternTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftPattern(t.addCircular)
}

func (t *FeatureCircPatternTool) Params() ToolParams {
	return ToolParams{
		Ints: []IntParam{{"Count", func() int { return t.count }, func(n int) { t.count = n }}},
		Floats: []FloatParam{{
			"Angle (deg)", func() float64 { return degFromRad(t.totalAngle) },
			func(d float64) { t.totalAngle = radFromDeg(d) },
		}},
	}
}

// --- Mirror ---------------------------------------------------------------

// FeatureMirrorTool mirrors the source features across the plane through the origin with
// the given normal (default the YZ plane, normal +X).
type FeatureMirrorTool struct {
	featureSelectTool
	normal math.Vector3
}

func NewFeatureMirrorTool() *FeatureMirrorTool {
	return &FeatureMirrorTool{normal: math.V3(1, 0, 0)}
}
func (t *FeatureMirrorTool) Name() string { return "Mirror" }
func (t *FeatureMirrorTool) Prompt(*Session) string {
	return "Select features, set the mirror-plane normal, then OK."
}

func (t *FeatureMirrorTool) CanCommit() bool {
	return len(t.sources) > 0 && t.normal.Length() > 0
}

func (t *FeatureMirrorTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if len(t.sources) == 0 {
		return errors.New("mirror: select a feature to mirror first")
	}
	t.addMirror(feature.NewPatternFeatures(part.Features()))
	return t.finishPattern(s, lastPartFeature(s), "Mirror")
}

// addMirror builds the mirror into pats — the shared constructor used by both Commit
// (the part's engine) and DraftFeature (a scratch engine), so the two cannot drift.
func (t *FeatureMirrorTool) addMirror(pats *feature.PatternFeatures) {
	pats.AddMirror(t.sources, nil, math.P3(0, 0, 0), t.normal)
}

// DraftFeature implements [PartFeatureTool] (#1626): the mirror it would commit, built
// into a scratch engine so the commit gate and preview can evaluate it without touching
// the part.
func (t *FeatureMirrorTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	return draftPattern(t.addMirror)
}

func (t *FeatureMirrorTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{
		{"Normal X", func() float64 { return float64(t.normal.X) }, func(v float64) { t.normal.X = math.Scalar(v) }},
		{"Normal Y", func() float64 { return float64(t.normal.Y) }, func(v float64) { t.normal.Y = math.Scalar(v) }},
		{"Normal Z", func() float64 { return float64(t.normal.Z) }, func(v float64) { t.normal.Z = math.Scalar(v) }},
	}}
}

// lastPartFeature returns the most-recently-added feature — the pattern/mirror just
// created (the Pattern Add* methods return the raw feature, not its PartFeature wrapper).
func lastPartFeature(s *Session) *feature.PartFeature {
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	return lastEngineFeature(part.Features())
}

// lastEngineFeature returns the newest feature in fs — the one an Add* method just
// appended when it returns the raw feature instead of its PartFeature wrapper.
func lastEngineFeature(fs *feature.PartFeatures) *feature.PartFeature {
	if fs.Count() == 0 {
		return nil
	}
	return fs.Item(fs.Count() - 1)
}
