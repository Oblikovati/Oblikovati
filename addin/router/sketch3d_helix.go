// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// Helical curve definitions over the wire (M06-F09, #624):
// sketch3d.editHelix redefines an existing helix — constant or variable
// shape plus end conditions — and the helical addEntity payload accepts the
// same rows/ends at creation.

// sketch3DEditHelix serves wire.MethodSketch3DEditHelix.
func sketch3DEditHelix(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.EditHelixArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	part, sk, err := activePartSketch3D(s, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	h, err := helixByID(sk, in.Entity)
	if err != nil {
		return nil, err
	}
	if err := applyHelixShapeEdit(part, h, in); err != nil {
		return nil, err
	}
	if err := applyHelixEndEdit(part, h, in.Start, in.End); err != nil {
		return nil, err
	}
	if _, err := h.Curve(); err != nil {
		return nil, fmt.Errorf("sketch3d.editHelix: the edited definition does not regenerate: %w", err)
	}
	return json.Marshal(helixDefinitionView(h))
}

// applyHelixShapeEdit applies the constant fields or the variable rows; an
// edit carrying neither keeps the current shape.
func applyHelixShapeEdit(part *compdef.PartComponentDefinition, h *sketch.HelicalCurve3D, in wire.EditHelixArgs) error {
	kind, err := helixShapeKind(in.Mode)
	if err != nil {
		return err
	}
	if len(in.Rows) > 0 {
		rows, err := helixRowsFromWire(part, in.Rows)
		if err != nil {
			return err
		}
		return h.SetVariableShape(kind, rows)
	}
	if in.Mode == "" && in.Pitch == "" && in.Height == "" && in.Revolutions == 0 && in.Taper == "" {
		return nil
	}
	pitch, turns, radial, err := helixShape(part, wire.AddSketch3DEntityArgs{
		Mode: in.Mode, Pitch: in.Pitch, Height: in.Height,
		Revolutions: in.Revolutions, Taper: in.Taper,
	})
	if err != nil {
		return err
	}
	h.Clockwise = in.Clockwise
	return h.SetConstantShape(kind, pitch, radial, turns)
}

// applyHelixEndEdit parses and stores the optional end conditions.
func applyHelixEndEdit(part *compdef.PartComponentDefinition, h *sketch.HelicalCurve3D, start, end *wire.HelixEndCondition) error {
	s, err := helixEndFromWire(part, start)
	if err != nil {
		return err
	}
	e, err := helixEndFromWire(part, end)
	if err != nil {
		return err
	}
	h.SetEndConditions(s, e)
	return nil
}

// helixShapeKind parses the mode spelling (empty ⇒ pitchRevolution).
func helixShapeKind(mode string) (types.HelicalShapeDefinitionKind, error) {
	if mode == "" {
		return types.HelixShapePitchRevolution, nil
	}
	kind, ok := types.ParseHelicalShapeDefinitionKind(mode)
	if !ok {
		return 0, fmt.Errorf("unknown helix mode %q (want pitchRevolution|pitchHeight|revolutionHeight|spiral)", mode)
	}
	return kind, nil
}

// helixRowsFromWire parses the unit-bearing row table.
func helixRowsFromWire(part *compdef.PartComponentDefinition, rows []wire.HelixShapeRow) ([]sketch.HelixRow, error) {
	out := make([]sketch.HelixRow, len(rows))
	for i, r := range rows {
		diameter, err := lengthArg(part, "row diameter", r.Diameter)
		if err != nil {
			return nil, err
		}
		pitch, err := lengthArg(part, "row pitch", r.Pitch)
		if err != nil {
			return nil, err
		}
		height, err := lengthArg(part, "row height", r.Height)
		if err != nil {
			return nil, err
		}
		out[i] = sketch.HelixRow{Diameter: diameter, Pitch: pitch, Height: height, Revolution: r.Revolution}
	}
	return out, nil
}

// helixEndFromWire parses one optional end condition.
func helixEndFromWire(part *compdef.PartComponentDefinition, c *wire.HelixEndCondition) (*sketch.HelixEndCondition, error) {
	if c == nil {
		return nil, nil
	}
	kind := types.HelixEndNatural
	if c.Kind != "" {
		parsed, ok := types.ParseHelixEndKind(c.Kind)
		if !ok {
			return nil, fmt.Errorf("unknown helix end kind %q (want natural|flat)", c.Kind)
		}
		kind = parsed
	}
	transition, err := angleArg(part, "transitionAngle", c.TransitionAngle)
	if err != nil {
		return nil, err
	}
	flat, err := angleArg(part, "flatAngle", c.FlatAngle)
	if err != nil {
		return nil, err
	}
	return &sketch.HelixEndCondition{Kind: kind, TransitionAngle: transition, FlatAngle: flat}, nil
}

// helixDefinitionView renders the stored definition as its wire DTO.
func helixDefinitionView(h *sketch.HelicalCurve3D) wire.HelixDefinitionView {
	def := h.Definition()
	view := wire.HelixDefinitionView{
		ShapeKind: def.ShapeKind.String(), Variable: def.Variable(),
		Pitch: h.AxialPerTurn, Height: h.Height(), Revolutions: h.Turns,
		Taper: h.RadialPerTurn, Clockwise: h.Clockwise,
		Start: helixEndToWire(def.Start), End: helixEndToWire(def.End),
	}
	for _, r := range def.Rows {
		view.Rows = append(view.Rows, wire.HelixShapeRow{
			Diameter: formatLength(r.Diameter), Pitch: formatLength(r.Pitch),
			Height: formatLength(r.Height), Revolution: r.Revolution,
		})
	}
	return view
}

// formatLength renders a resolved cm value for the view's rows.
func formatLength(v float64) string {
	if v == 0 {
		return ""
	}
	return fmt.Sprintf("%g cm", v)
}

// helixEndToWire renders a non-natural end condition (nil for natural).
func helixEndToWire(c sketch.HelixEndCondition) *wire.HelixEndCondition {
	if c.Kind != types.HelixEndFlat {
		return nil
	}
	return &wire.HelixEndCondition{
		Kind:            c.Kind.String(),
		TransitionAngle: fmt.Sprintf("%g rad", c.TransitionAngle),
		FlatAngle:       fmt.Sprintf("%g rad", c.FlatAngle),
	}
}

// helixByID resolves a helical entity by session id.
func helixByID(sk *sketch.Sketch3D, id uint64) (*sketch.HelicalCurve3D, error) {
	e, ok := sk.EntityByID(sketch.ID(id))
	if !ok {
		return nil, fmt.Errorf("entity %d not found", id)
	}
	h, isHelix := e.(*sketch.HelicalCurve3D)
	if !isHelix {
		return nil, fmt.Errorf("entity %d is %T, want a helical curve", id, e)
	}
	return h, nil
}

// activePartSketch3D resolves the active part and its 3D sketch at index.
func activePartSketch3D(s *app.Session, index int) (*compdef.PartComponentDefinition, *sketch.Sketch3D, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, nil, err
	}
	sk, err := sketch3DAtIndex(part, index)
	if err != nil {
		return nil, nil, err
	}
	return part, sk, nil
}
