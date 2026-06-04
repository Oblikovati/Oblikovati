// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// createSketch3D adds an empty 3D sketch to the active part and returns its index.
func createSketch3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.CreateSketch3DArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if in.Name != "" {
		part.Sketches3D().AddNamed(in.Name)
	} else {
		part.Sketches3D().Add()
	}
	return json.Marshal(wire.CreateSketch3DResult{SketchIndex: part.Sketches3D().Count() - 1})
}

// listSketches3D enumerates the active part's 3D sketches with identity and solve state.
func listSketches3D(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	sketches := part.Sketches3D()
	out := make([]wire.Sketch3DInfo, sketches.Count())
	for i := 0; i < sketches.Count(); i++ {
		out[i] = sketch3DInfo(i, sketches.Item(i))
	}
	return json.Marshal(wire.ListSketches3DResult{Sketches: out})
}

// getSketch3D returns one 3D sketch's info by index.
func getSketch3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, idx, err := resolveSketch3D(s, raw)
	if err != nil {
		return nil, err
	}
	return json.Marshal(sketch3DInfo(idx, sk))
}

// editSketch3D enters edit mode; exitEditSketch3D leaves it.
func editSketch3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	return setSketch3DEdit(s, raw, true)
}

func exitEditSketch3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	return setSketch3DEdit(s, raw, false)
}

func setSketch3DEdit(s *app.Session, raw json.RawMessage, edit bool) (json.RawMessage, error) {
	sk, idx, err := resolveSketch3D(s, raw)
	if err != nil {
		return nil, err
	}
	if edit {
		sk.Edit()
	} else {
		sk.ExitEdit()
	}
	return json.Marshal(wire.EditSketch3DResult{SketchIndex: idx, Editing: sk.IsEditing()})
}

// solveSketch3D resolves the sketch and reports DOF/status/convergence/health.
func solveSketch3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, idx, err := resolveSketch3D(s, raw)
	if err != nil {
		return nil, err
	}
	res := sk.Solve()
	return json.Marshal(wire.SolveSketch3DResult{
		SketchIndex: idx,
		DOF:         res.DOF,
		Status:      solveStatus(res.Status),
		Converged:   res.Converged,
		Healthy:     sk.Health().OK(),
	})
}

// constraintStatus3D reports the sketch's DOF analysis without moving geometry.
func constraintStatus3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch3D(s, raw)
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

// deleteSketch3D removes a 3D sketch (only valid when no feature consumes it).
func deleteSketch3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch3D(s, raw)
	if err != nil {
		return nil, err
	}
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	if !part.Sketches3D().Remove(sk.ID()) {
		return nil, fmt.Errorf("sketch3d.delete: sketch %d could not be removed", sk.ID())
	}
	return json.Marshal(wire.OKResult{OK: true})
}

// setSketch3DProperty edits one display/solve property and echoes the updated info.
func setSketch3DProperty(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetSketch3DPropertyArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	if err := applySketch3DProperty(sk, in.Property, in.Value); err != nil {
		return nil, err
	}
	return json.Marshal(sketch3DInfo(in.SketchIndex, sk))
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

// resolveSketch3D decodes a Sketch3DArgs and returns the active part's 3D sketch at that index.
func resolveSketch3D(s *app.Session, raw json.RawMessage) (*sketch.Sketch3D, int, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, 0, err
	}
	var in wire.Sketch3DArgs
	if err := decode(raw, &in); err != nil {
		return nil, 0, err
	}
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, 0, err
	}
	return sk, in.SketchIndex, nil
}
