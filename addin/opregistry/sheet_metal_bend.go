// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// sheetMetalBendArgs is the argument shape for the "sheetMetalBend" operation: the sketch bend
// line (sketch + line index), the optional bend angle/radius (radius defaults to the rule's
// bend radius, angle to 90°), and a flip. Thickness comes from the rule.
type sheetMetalBendArgs struct {
	SketchIndex int    `json:"sketchIndex"`
	LineIndex   int    `json:"lineIndex"`
	Angle       string `json:"angle,omitempty"`
	Radius      string `json:"radius,omitempty"`
	Flip        bool   `json:"flip,omitempty"`
}

const sheetMetalBendSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Index of the sketch holding the bend line (see model.tree)."},
    "lineIndex": {"type": "integer", "minimum": 0, "default": 0, "description": "Which line of the sketch is the bend axis. It must cross the sheet."},
    "angle": {"type": "string", "description": "Bend angle, e.g. \"90 deg\" (default)."},
    "radius": {"type": "string", "description": "Inside bend radius (default: the rule's bend radius)."},
    "flip": {"type": "boolean", "default": false, "description": "Fold toward the opposite side of the sketch plane."}
  },
  "required": ["sketchIndex"]
}`

// sheetMetalBendDescriptor is the self-describing "sheetMetalBend" operation: fold a flat
// sheet along a sketch line at the active rule's bend radius.
func sheetMetalBendDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    "sheetMetalBend",
		Summary: "Fold a flat sheet-metal wall along a sketch line over a bend, at the active rule's thickness and bend radius.",
		Schema:  json.RawMessage(sheetMetalBendSchema),
		Apply:   applySheetMetalBend,
	}
}

func applySheetMetalBend(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalBend")
	if err != nil {
		return nil, err
	}
	var in sheetMetalBendArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalBend: invalid args: %w", err)
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	def, err := bendDef(part, sk, in)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, feature.NewSheetMetalBendFeatures(part.Features()).Add(def))
}

// bendDef resolves the bend args into a definition: the sketch + line, and the optional
// angle/radius closures (omitted ⇒ nil, so the feature uses its defaults).
func bendDef(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in sheetMetalBendArgs) (*feature.SheetMetalBendDefinition, error) {
	def := &feature.SheetMetalBendDefinition{Sketch: sk, LineIndex: in.LineIndex, Flip: in.Flip}
	if in.Angle != "" {
		angle, err := angleClosure(part, in.Angle, "sheetMetalBend: angle")
		if err != nil {
			return nil, err
		}
		def.Angle = angle
	}
	if in.Radius != "" {
		radius, err := lengthClosure(part, in.Radius, "sheetMetalBend: radius")
		if err != nil {
			return nil, err
		}
		def.Radius = radius
	}
	return def, nil
}
