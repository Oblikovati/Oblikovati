// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/sketch"
)

// Sketch inference over the wire (M06-F10, #625): get/set the session's
// inference preferences, and report what the engine auto-applied on
// addEntity (sketch_add_entity.go calls applyEntityInference).

// getInferenceOptions serves wire.MethodSketchGetInferenceOptions.
func getInferenceOptions(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(inferenceOptionsView(s.SketchInferenceOptions()))
}

// setInferenceOptions serves wire.MethodSketchSetInferenceOptions. An empty
// priority keeps the current value; unknown spellings are rejected.
func setInferenceOptions(s *app.Session, in wire.InferenceOptionsView) (wire.InferenceOptionsView, error) {
	opts := s.SketchInferenceOptions()
	opts.InferEnabled, opts.ConstrainEnabled = in.InferEnabled, in.ConstrainEnabled
	if in.Priority != "" {
		priority, ok := types.ParseConstraintInferencePriority(in.Priority)
		if !ok {
			return wire.InferenceOptionsView{}, fmt.Errorf("unknown inference priority %q (want parallelPerpendicular|horizontalVertical|none)", in.Priority)
		}
		opts.Priority = priority
	}
	s.SetSketchInferenceOptions(opts)
	return inferenceOptionsView(opts), nil
}

// inferenceOptionsView renders the model options as their wire view.
func inferenceOptionsView(opts sketch.InferenceOptions) wire.InferenceOptionsView {
	return wire.InferenceOptionsView{
		InferEnabled:     opts.InferEnabled,
		ConstrainEnabled: opts.ConstrainEnabled,
		Priority:         opts.Priority.String(),
	}
}

// applyEntityInference runs inference on a freshly added line entity and
// renders the applied records onto the addEntity result.
func applyEntityInference(s *app.Session, sk *sketch.Sketch, e sketch.Entity, out *wire.AddSketchEntityResult) {
	l, isLine := e.(*sketch.Line)
	if !isLine {
		return
	}
	constraints, points := sk.ApplyLineInference(l, s.SketchInferenceOptions())
	for _, c := range constraints {
		out.InferredConstraints = append(out.InferredConstraints, wire.AppliedConstraintInference{
			Kind: c.Kind.String(), ConstraintIndex: c.ConstraintIndex, Entities: entityRefIDs(c.Entities),
		})
	}
	for _, p := range points {
		out.InferredPoints = append(out.InferredPoints, wire.AppliedPointInference{
			Kind: p.Kind.String(), PointID: uint64(p.Point.EntityID()), Entities: entityRefIDs(p.Entities),
		})
	}
}

// entityRefIDs renders entity operands as session ids.
func entityRefIDs(ents []sketch.Entity) []uint64 {
	out := make([]uint64, len(ents))
	for i, e := range ents {
		out[i] = uint64(e.EntityID())
	}
	return out
}
