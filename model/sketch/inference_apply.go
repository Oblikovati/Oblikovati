// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// Applied inference (M06-F10, Oblikovati/Oblikovati#625): the engine's
// suggestions (inference.go) reified as typed records and auto-applied as
// real constraints on commit, so interactive add-ins can drive and audit
// what sketching inferred. The user preferences — point/constraint inference
// on/off and which constraint family wins — live in [InferenceOptions].

// InferenceOptions is the sketch inference configuration.
type InferenceOptions struct {
	// InferEnabled runs point snapping (endpoint → existing point).
	InferEnabled bool
	// ConstrainEnabled auto-applies the inferred constraints on commit.
	ConstrainEnabled bool
	// Priority picks the constraint family when two could apply.
	Priority types.ConstraintInferencePriority
}

// DefaultInferenceOptions is the out-of-the-box configuration: everything on,
// horizontal/vertical preferred.
func DefaultInferenceOptions() InferenceOptions {
	return InferenceOptions{
		InferEnabled: true, ConstrainEnabled: true,
		Priority: types.PriorityHorizontalVertical,
	}
}

// AppliedInference is one geometric constraint the engine auto-applied.
type AppliedInference struct {
	Kind            types.ConstraintInferenceKind
	ConstraintIndex int
	Entities        []Entity
}

// AppliedPointInference is one defining point the engine snapped.
type AppliedPointInference struct {
	Kind     types.SketchPointInferenceKind
	Point    *Point
	Entities []Entity
}

// ApplyLineInference runs the engine against a freshly committed line and —
// when constraining is enabled — applies the winning suggestions: endpoint
// coincidence onto nearby points, then horizontal/vertical or parallel/
// perpendicular per the priority preference. Returns what was applied.
func (s *Sketch) ApplyLineInference(l *Line, opts InferenceOptions) ([]AppliedInference, []AppliedPointInference) {
	if !opts.InferEnabled {
		return nil, nil
	}
	engine := NewInference()
	points := s.inferSnapPoints(l, engine, opts)
	constraints := s.inferDirection(l, engine, opts)
	return constraints, points
}

// inferSnapPoints snaps the line's endpoints onto nearby existing points.
func (s *Sketch) inferSnapPoints(l *Line, engine *Inference, opts InferenceOptions) []AppliedPointInference {
	var out []AppliedPointInference
	candidates := s.snapCandidates(l)
	for _, end := range []*Point{l.A, l.B} {
		suggestions := engine.InferSnap(end.Position(), candidates)
		if len(suggestions) == 0 || suggestions[0].Target == nil {
			continue
		}
		target := suggestions[0].Target
		if opts.ConstrainEnabled {
			end.SetPosition(target.Position())
			s.geomCons.AddCoincident(end, target)
		}
		out = append(out, AppliedPointInference{
			Kind: types.SketchInferenceOnPoint, Point: end, Entities: []Entity{target},
		})
	}
	return out
}

// snapCandidates are every sketch point except the line's own endpoints.
func (s *Sketch) snapCandidates(l *Line) []*Point {
	out := make([]*Point, 0, len(s.pts))
	for _, p := range s.pts {
		if p != l.A && p != l.B {
			out = append(out, p)
		}
	}
	return out
}

// inferDirection applies the winning direction suggestion: horizontal/
// vertical, or parallel/perpendicular against an existing line, ranked by the
// priority preference.
func (s *Sketch) inferDirection(l *Line, engine *Inference, opts InferenceOptions) []AppliedInference {
	best, target, ok := engine.bestDirection(l, s.otherLines(l), opts.Priority)
	if !ok {
		return nil
	}
	if opts.ConstrainEnabled {
		s.applyDirectionConstraint(l, best, target)
	}
	record := AppliedInference{
		Kind:            inferenceKindOf(best),
		ConstraintIndex: s.geomCons.Count() - 1,
		Entities:        directionEntities(l, target),
	}
	return []AppliedInference{record}
}

// otherLines are the sketch's lines excluding l.
func (s *Sketch) otherLines(l *Line) []*Line {
	out := make([]*Line, 0, s.lines.Count())
	for i := 0; i < s.lines.Count(); i++ {
		if x := s.lines.Item(i); x != l {
			out = append(out, x)
		}
	}
	return out
}

// applyDirectionConstraint adds the model constraint for the suggestion.
func (s *Sketch) applyDirectionConstraint(l *Line, kind SuggestionKind, target *Line) {
	switch kind {
	case SuggestHorizontal:
		// The inference is "this LINE is horizontal", so it applies the single-line form
		// (enumerates "horizontal"), not the two-point align form (#1871).
		s.geomCons.AddLineHorizontal(l)
	case SuggestVertical:
		s.geomCons.AddLineVertical(l)
	case SuggestParallel:
		s.geomCons.AddParallel(l, target)
	case SuggestPerpendicular:
		s.geomCons.AddPerpendicular(l, target)
	}
}

// inferenceKindOf maps a suggestion onto its public inference kind.
func inferenceKindOf(kind SuggestionKind) types.ConstraintInferenceKind {
	switch kind {
	case SuggestHorizontal:
		return types.InferHorizontal
	case SuggestVertical:
		return types.InferVertical
	case SuggestParallel:
		return types.InferParallel
	case SuggestPerpendicular:
		return types.InferPerpendicular
	default:
		return types.InferCoincident
	}
}

// directionEntities are the constraint's operands for the record.
func directionEntities(l *Line, target *Line) []Entity {
	if target != nil {
		return []Entity{l, target}
	}
	return []Entity{l}
}

// bestDirection ranks the H/V suggestions against parallel/perpendicular
// candidates per the priority preference and returns the winner.
func (in *Inference) bestDirection(l *Line, others []*Line, priority types.ConstraintInferencePriority) (SuggestionKind, *Line, bool) {
	axial := in.InferSegment(l.A.Position(), l.B.Position())
	relKind, relTarget, relScore := in.bestRelative(l, others)
	axKind, axScore := bestAxial(axial)
	switch {
	case axScore < 0 && relScore < 0:
		return 0, nil, false
	case axScore < 0:
		return relKind, relTarget, true
	case relScore < 0:
		return axKind, nil, true
	case priority == types.PriorityParallelPerpendicular:
		return relKind, relTarget, true
	case priority == types.PriorityNone && relScore > axScore:
		return relKind, relTarget, true
	default: // horizontal/vertical preferred (the default)
		return axKind, nil, true
	}
}

// bestAxial picks the strongest horizontal/vertical suggestion.
func bestAxial(suggestions []Suggestion) (SuggestionKind, int) {
	kind, score := SuggestionKind(0), -1
	for _, sg := range suggestions {
		if sg.Priority > score {
			kind, score = sg.Kind, sg.Priority
		}
	}
	return kind, score
}

// bestRelative finds the existing line this segment is most nearly parallel
// or perpendicular to (within the angle tolerance).
func (in *Inference) bestRelative(l *Line, others []*Line) (SuggestionKind, *Line, int) {
	kind, score := SuggestionKind(0), -1
	var target *Line
	for _, o := range others {
		k, p, ok := in.relativeAngle(l, o)
		if ok && p > score {
			kind, target, score = k, o, p
		}
	}
	return kind, target, score
}

// relativeAngle classifies the angle between two lines as parallel or
// perpendicular within the tolerance.
func (in *Inference) relativeAngle(l, o *Line) (SuggestionKind, int, bool) {
	a := l.Direction().AngleTo(o.Direction())
	folded := stdmath.Min(float64(a), stdmath.Pi-float64(a)) // direction-insensitive
	if folded <= in.AngleTolerance {
		return SuggestParallel, priorityFor(folded, 0, in.AngleTolerance), true
	}
	if stdmath.Abs(folded-stdmath.Pi/2) <= in.AngleTolerance {
		return SuggestPerpendicular, priorityFor(folded, stdmath.Pi/2, in.AngleTolerance), true
	}
	return 0, 0, false
}

// GlyphSuggestions returns the inference glyphs to draw at the cursor for an
// in-progress segment a→b — the headless feed of the UI overlay (#599 row
// "inference glyph overlay").
func (s *Sketch) GlyphSuggestions(a, b math.Point2, opts InferenceOptions) []Suggestion {
	if !opts.InferEnabled {
		return nil
	}
	engine := NewInference()
	out := engine.InferSegment(a, b)
	for i := 0; i < s.lines.Count(); i++ {
		probe := &Line{A: &Point{X: a.X, Y: a.Y}, B: &Point{X: b.X, Y: b.Y}}
		if kind, p, ok := engine.relativeAngle(probe, s.lines.Item(i)); ok {
			out = append(out, Suggestion{Kind: kind, Priority: p})
		}
	}
	if snap := engine.InferSnap(b, s.AllPoints()); len(snap) > 0 {
		out = append(out, snap...)
	}
	return out
}
