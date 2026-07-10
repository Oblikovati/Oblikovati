// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"math"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// listSketches enumerates the active part's sketches with their identity and solve state.
func listSketches(_ *app.Session, part *compdef.PartComponentDefinition) (wire.ListSketchesResult, error) {
	out := projectAll(part.Sketches(), func(i int, sk *sketch.Sketch) wire.SketchInfo {
		return sketchInfo(part, i, sk)
	})
	return wire.ListSketchesResult{Sketches: out}, nil
}

// getSketch returns one sketch's info by index.
func getSketch(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SketchArgs) (wire.SketchInfo, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.SketchInfo{}, err
	}
	return sketchInfo(part, in.SketchIndex, sk), nil
}

// editSketch enters edit mode; exitEditSketch leaves it.
func editSketch(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SketchArgs) (wire.EditSketchResult, error) {
	return setSketchEdit(part, in, true)
}

func exitEditSketch(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SketchArgs) (wire.EditSketchResult, error) {
	return setSketchEdit(part, in, false)
}

// setSketchEdit toggles a sketch's edit mode and echoes the resulting state.
func setSketchEdit(part *compdef.PartComponentDefinition, in wire.SketchArgs, edit bool) (wire.EditSketchResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.EditSketchResult{}, err
	}
	if edit {
		sk.Edit()
	} else {
		sk.ExitEdit()
	}
	return wire.EditSketchResult{SketchIndex: in.SketchIndex, Editing: sk.IsEditing()}, nil
}

// solveSketch resolves the sketch and reports DOF/status/convergence/health.
func solveSketch(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SketchArgs) (wire.SolveSketchResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.SolveSketchResult{}, err
	}
	res := sk.Solve()
	return wire.SolveSketchResult{
		SketchIndex: in.SketchIndex,
		DOF:         res.DOF,
		Status:      solveStatus(res.Status),
		Converged:   res.Converged,
		Healthy:     sk.Health().OK(),
	}, nil
}

// constraintStatus reports the sketch's DOF analysis without moving geometry.
func constraintStatus(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SketchArgs) (wire.ConstraintStatusResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.ConstraintStatusResult{}, err
	}
	a := sk.AnalyzeConstraints()
	return wire.ConstraintStatusResult{
		Status:    dofStatus(a.DOF, a.Redundant),
		DOF:       a.DOF,
		Variables: a.Variables,
		Equations: a.Equations,
		Redundant: a.Redundant,
	}, nil
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
func deleteSketch(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SketchArgs) (wire.OKResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.OKResult{}, err
	}
	if err := rejectIfConsumed(part, sk); err != nil {
		return wire.OKResult{}, err
	}
	snapshot := snapshotConstructionConsumers(part) // before delete: construction datums with a consumer (#1849)
	if !part.Sketches().Remove(sk.ID()) {
		return wire.OKResult{}, fmt.Errorf("sketch.delete: sketch %d could not be removed", sk.ID())
	}
	pruneConstructionAfterDelete(part, snapshot) // auto-delete the host work plane if this sketch was its last consumer
	return wire.OKResult{OK: true}, nil
}

// sketchInfo renders a sketch as its wire summary (DOF computed without moving geometry),
// including its consumed/owned-by/shared state derived from the part's features (#154).
func sketchInfo(part *compdef.PartComponentDefinition, index int, sk *sketch.Sketch) wire.SketchInfo {
	info := wire.SketchInfo{
		Index:        index,
		Name:         sk.Name(),
		Plane:        planeLabel(sk.Plane()),
		Visible:      sk.Visible(),
		EntityCount:  sk.EntityCount(),
		DOF:          sk.DegreesOfFreedom(),
		Editing:      sk.IsEditing(),
		Healthy:      sk.Health().OK(),
		Shared:       sk.Shared(),
		Color:        sk.Color(),
		LineType:     sk.LineType(),
		LineWeight:   sk.LineWeight(),
		DeferUpdates: sk.DeferUpdates(),
	}
	if cons := sketchConsumers(part, sk); len(cons) > 0 {
		info.Consumed, info.OwnedBy = true, cons[0].Name()
	}
	return info
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
