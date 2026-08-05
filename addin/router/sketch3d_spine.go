// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"strconv"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// createSketch3D adds an empty 3D sketch to the active part and returns its index.
func createSketch3D(_ *app.Session, part *compdef.PartComponentDefinition, in wire.CreateSketch3DArgs) (wire.CreateSketch3DResult, error) {
	if in.Name != "" {
		part.Sketches3D().AddNamed(in.Name)
	} else {
		part.Sketches3D().Add()
	}
	return wire.CreateSketch3DResult{SketchIndex: part.Sketches3D().Count() - 1}, nil
}

// listSketches3D enumerates the active part's 3D sketches with identity and solve state.
func listSketches3D(_ *app.Session, part *compdef.PartComponentDefinition) (wire.ListSketches3DResult, error) {
	return wire.ListSketches3DResult{Sketches: projectAll(part.Sketches3D(), sketch3DInfo)}, nil
}

// getSketch3D returns one 3D sketch's info by index.
func getSketch3D(_ *app.Session, part *compdef.PartComponentDefinition, in wire.Sketch3DArgs) (wire.Sketch3DInfo, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.Sketch3DInfo{}, err
	}
	return sketch3DInfo(in.SketchIndex, sk), nil
}

// editSketch3D enters edit mode; exitEditSketch3D leaves it.
func editSketch3D(s *app.Session, part *compdef.PartComponentDefinition, in wire.Sketch3DArgs) (wire.EditSketch3DResult, error) {
	return setSketch3DEdit(s, part, in, true)
}

func exitEditSketch3D(s *app.Session, part *compdef.PartComponentDefinition, in wire.Sketch3DArgs) (wire.EditSketch3DResult, error) {
	return setSketch3DEdit(s, part, in, false)
}

// setSketch3DEdit opens or closes a 3D sketch for editing and echoes the resulting state.
//
// It goes through the SESSION, mirroring the planar setSketchEdit. Marking the sketch object
// edited without entering the environment answered {"editing": true} while InSketch3D stayed
// false — so the contextual 3D Sketch tab never appeared and every command gated on it stayed
// disabled, which is what made the 3D sketch undriveable over the API and made a live sweep read
// its whole ribbon as inert.
func setSketch3DEdit(s *app.Session, part *compdef.PartComponentDefinition, in wire.Sketch3DArgs, edit bool) (wire.EditSketch3DResult, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.EditSketch3DResult{}, err
	}
	switch {
	case edit:
		s.EnterSketch3D(sk)
	case s.ActiveSketch3D() == sk:
		s.ExitSketch3D()
	default:
		sk.ExitEdit() // a sketch marked edited without being the session's active one
	}
	return wire.EditSketch3DResult{SketchIndex: in.SketchIndex, Editing: sk.IsEditing()}, nil
}

// solveSketch3D resolves the sketch and reports DOF/status/convergence/health.
func solveSketch3D(_ *app.Session, part *compdef.PartComponentDefinition, in wire.Sketch3DArgs) (wire.SolveSketch3DResult, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.SolveSketch3DResult{}, err
	}
	res := sk.Solve()
	return wire.SolveSketch3DResult{
		SketchIndex: in.SketchIndex,
		DOF:         res.DOF,
		Status:      solveStatus(res.Status),
		Converged:   res.Converged,
		Healthy:     sk.Health().OK(),
	}, nil
}

// constraintStatus3D reports the sketch's DOF analysis without moving geometry.
func constraintStatus3D(_ *app.Session, part *compdef.PartComponentDefinition, in wire.Sketch3DArgs) (wire.ConstraintStatusResult, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
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

// deleteSketch3D removes a 3D sketch (only valid when no feature consumes it).
func deleteSketch3D(_ *app.Session, part *compdef.PartComponentDefinition, in wire.Sketch3DArgs) (wire.OKResult, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.OKResult{}, err
	}
	if !part.Sketches3D().Remove(sk.ID()) {
		return wire.OKResult{}, fmt.Errorf("sketch3d.delete: sketch %d could not be removed", sk.ID())
	}
	return wire.OKResult{OK: true}, nil
}

// setSketch3DProperty edits one display/solve property and echoes the updated info.
func setSketch3DProperty(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SetSketch3DPropertyArgs) (wire.Sketch3DInfo, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.Sketch3DInfo{}, err
	}
	if err := applySketch3DProperty(sk, in.Property, in.Value); err != nil {
		return wire.Sketch3DInfo{}, err
	}
	return sketch3DInfo(in.SketchIndex, sk), nil
}

// applySketch3DProperty sets one named property from its string value.
func applySketch3DProperty(sk *sketch.Sketch3D, property, value string) error {
	switch property {
	case "name":
		sk.SetName(value)
	case "color":
		sk.SetColor(value)
	case "visible", "dimensionsVisible", "deferUpdates":
		return applySketch3DBoolProperty(sk, property, value)
	default:
		return fmt.Errorf("sketch3d.setProperty: unknown property %q (want name|visible|dimensionsVisible|color|deferUpdates)", property)
	}
	return nil
}

// applySketch3DBoolProperty parses and sets one of the boolean display/solve properties.
func applySketch3DBoolProperty(sk *sketch.Sketch3D, property, value string) error {
	b, err := parseBoolProp(property, value)
	if err != nil {
		return err
	}
	switch property {
	case "visible":
		sk.SetVisible(b)
	case "dimensionsVisible":
		sk.SetDimensionsVisible(b)
	case "deferUpdates":
		sk.SetDeferUpdates(b)
	}
	return nil
}

// parseBoolProp parses a "true"/"false" property value, reporting the offending input.
func parseBoolProp(property, value string) (bool, error) {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("sketch3d.setProperty: %s value %q is not a bool", property, value)
	}
	return b, nil
}

// sketch3DInfo renders a 3D sketch as its wire summary (DOF computed without moving geometry).
func sketch3DInfo(index int, sk *sketch.Sketch3D) wire.Sketch3DInfo {
	return wire.Sketch3DInfo{
		Index:             index,
		Name:              sk.Name(),
		Visible:           sk.Visible(),
		DimensionsVisible: sk.DimensionsVisible(),
		EntityCount:       sk.EntityCount(),
		DOF:               sk.DegreesOfFreedom(),
		Editing:           sk.IsEditing(),
		Healthy:           sk.Health().OK(),
		Shared:            sk.Shared(),
		Color:             sk.Color(),
		DeferUpdates:      sk.DeferUpdates(),
	}
}

// sketch3DAtIndex returns the active part's 3D sketch at i, bounds-checked.
func sketch3DAtIndex(part *compdef.PartComponentDefinition, i int) (*sketch.Sketch3D, error) {
	if i < 0 || i >= part.Sketches3D().Count() {
		return nil, fmt.Errorf("3D sketch index %d out of range (part has %d 3D sketches)", i, part.Sketches3D().Count())
	}
	return part.Sketches3D().Item(i), nil
}
