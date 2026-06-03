// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// listSketches enumerates the active part's sketches with their identity and solve state.
func listSketches(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	sketches := part.Sketches()
	out := make([]wire.SketchInfo, sketches.Count())
	for i := 0; i < sketches.Count(); i++ {
		out[i] = sketchInfo(i, sketches.Item(i))
	}
	return json.Marshal(wire.ListSketchesResult{Sketches: out})
}

// getSketch returns one sketch's info by index.
func getSketch(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, idx, err := resolveSketch(s, raw)
	if err != nil {
		return nil, err
	}
	return json.Marshal(sketchInfo(idx, sk))
}

// editSketch enters edit mode; exitEditSketch leaves it.
func editSketch(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	return setSketchEdit(s, raw, true)
}

func exitEditSketch(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	return setSketchEdit(s, raw, false)
}

// setSketchEdit toggles a sketch's edit mode and echoes the resulting state.
func setSketchEdit(s *app.Session, raw json.RawMessage, edit bool) (json.RawMessage, error) {
	sk, idx, err := resolveSketch(s, raw)
	if err != nil {
		return nil, err
	}
	if edit {
		sk.Edit()
	} else {
		sk.ExitEdit()
	}
	return json.Marshal(wire.EditSketchResult{SketchIndex: idx, Editing: sk.IsEditing()})
}

// solveSketch resolves the sketch and reports DOF/status/convergence/health.
func solveSketch(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, idx, err := resolveSketch(s, raw)
	if err != nil {
		return nil, err
	}
	res := sk.Solve()
	return json.Marshal(wire.SolveSketchResult{
		SketchIndex: idx,
		DOF:         res.DOF,
		Status:      solveStatus(res.Status),
		Converged:   res.Converged,
		Healthy:     sk.Health().OK(),
	})
}

// constraintStatus reports the sketch's DOF analysis without moving geometry.
func constraintStatus(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch(s, raw)
	if err != nil {
		return nil, err
	}
	a := sk.AnalyzeConstraints()
	return json.Marshal(wire.ConstraintStatusResult{
		Status:    dofStatus(a.DOF, a.Redundant),
		DOF:       a.DOF,
		Variables: a.Variables,
		Equations: a.Equations,
		Redundant: a.Redundant,
	})
}

// dofStatus maps a DOF count + redundancy to the wire constraint-status string.
func dofStatus(dof, redundant int) string {
	switch {
	case redundant > 0:
		return "over"
	case dof > 0:
		return "under"
	default:
		return "well"
	}
}

// deleteSketch removes a sketch (only valid when no feature consumes it).
func deleteSketch(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch(s, raw)
	if err != nil {
		return nil, err
	}
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	if !part.Sketches().Remove(sk.ID()) {
		return nil, fmt.Errorf("sketch.delete: sketch %d could not be removed", sk.ID())
	}
	return json.Marshal(wire.OKResult{OK: true})
}

// sketchInfo renders a sketch as its wire summary (DOF computed without moving geometry).
func sketchInfo(index int, sk *sketch.Sketch) wire.SketchInfo {
	return wire.SketchInfo{
		Index:        index,
		Name:         sk.Name(),
		Plane:        planeLabel(sk.Plane()),
		Visible:      sk.Visible(),
		EntityCount:  sk.EntityCount(),
		DOF:          sk.DegreesOfFreedom(),
		Editing:      sk.IsEditing(),
		Healthy:      sk.Health().OK(),
		Color:        sk.Color(),
		LineType:     sk.LineType(),
		LineWeight:   sk.LineWeight(),
		DeferUpdates: sk.DeferUpdates(),
	}
}

// activeSketchAt returns the active part's sketch at the given index.
func activeSketchAt(s *app.Session, index int) (*sketch.Sketch, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	return sketchAtIndex(part, index)
}

// resolveSketch decodes a SketchArgs and returns the active part's sketch at that index.
func resolveSketch(s *app.Session, raw json.RawMessage) (*sketch.Sketch, int, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, 0, err
	}
	var in wire.SketchArgs
	if err := decode(raw, &in); err != nil {
		return nil, 0, err
	}
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, 0, err
	}
	return sk, in.SketchIndex, nil
}

// solveStatus maps the solver status to the wire string.
func solveStatus(st sketch.SolveStatus) string {
	switch st {
	case sketch.OverConstrained:
		return "over"
	case sketch.UnderConstrained:
		return "under"
	default:
		return "well"
	}
}

// planeLabel names an origin plane ("XY"|"XZ"|"YZ") by the world axis its normal aligns
// with, or "custom" for a non-origin plane. The sketch plane stores only a frame, not a
// label, so we recover it from the normal (XY⟂Z, XZ⟂Y, YZ⟂X).
func planeLabel(p sketch.Plane) string {
	n := p.Normal().AsVector()
	switch {
	case axisAligned(float64(n.Z), float64(n.X), float64(n.Y)):
		return "XY"
	case axisAligned(float64(n.Y), float64(n.X), float64(n.Z)):
		return "XZ"
	case axisAligned(float64(n.X), float64(n.Y), float64(n.Z)):
		return "YZ"
	default:
		return "custom"
	}
}

// axisAligned reports whether dominant is ±1 while the two others are ~0.
func axisAligned(dominant, a, b float64) bool {
	const eps = 1e-9
	return math.Abs(math.Abs(dominant)-1) < eps && math.Abs(a) < eps && math.Abs(b) < eps
}
