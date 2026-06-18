// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"strconv"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// setSketchProperty updates one scalar sketch property (name/visible/color/lineType/
// lineWeight/deferUpdates) and returns the updated sketch info.
func setSketchProperty(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetSketchPropertyArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	if err := applySketchProperty(part, sk, in.Property, in.Value); err != nil {
		return nil, err
	}
	return json.Marshal(sketchInfo(part, in.SketchIndex, sk))
}

// applySketchProperty applies one property=value to the sketch, parsing typed values.
func applySketchProperty(part *compdef.PartComponentDefinition, sk *sketch.Sketch, property, value string) error {
	switch property {
	case "name":
		sk.SetName(value)
	case "color":
		sk.SetColor(value)
	case "lineType":
		sk.SetLineType(value)
	case "visible":
		return parseBoolInto(value, property, sk.SetVisible)
	case "deferUpdates":
		return parseBoolInto(value, property, sk.SetDeferUpdates)
	case "lineWeight":
		return setSketchLineWeight(part, sk, value)
	default:
		return fmt.Errorf("sketch.setProperty: unknown property %q (want name|visible|color|lineType|lineWeight|deferUpdates)", property)
	}
	return nil
}

// parseBoolInto parses a "true"/"false" value and applies it, naming the property on error.
func parseBoolInto(value, property string, set func(bool)) error {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("sketch.setProperty: %s wants a bool, got %q: %w", property, value, err)
	}
	set(b)
	return nil
}

// setSketchLineWeight parses a unit-bearing length ("0.5 mm") into model cm and applies it.
func setSketchLineWeight(part *compdef.PartComponentDefinition, sk *sketch.Sketch, value string) error {
	q, err := part.Units().Parse(value, param.Length)
	if err != nil {
		return fmt.Errorf("sketch.setProperty: lineWeight %q: %w", value, err)
	}
	sk.SetLineWeight(q.Value)
	return nil
}
