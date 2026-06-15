// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// Model GD&T tolerance carrier (M20-F13 #866): a metadata feature that annotates model
// geometry with feature-control frames and datum labels. It changes no geometry — it carries
// records that survive recompute and the .obk round trip.

type toleranceFrameArgs struct {
	Geometry       string   `json:"geometry"`
	Characteristic string   `json:"characteristic"`
	Value          string   `json:"value"`
	Datums         []string `json:"datums"`
}

type datumLabelArgs struct {
	Geometry string `json:"geometry"`
	Label    string `json:"label"`
}

type modelToleranceArgs struct {
	Frames []toleranceFrameArgs `json:"frames"`
	Datums []datumLabelArgs     `json:"datums"`
}

const modelToleranceSchema = `{
  "type": "object",
  "properties": {
    "frames": {
      "type": "array",
      "description": "Feature-control frames annotating model geometry.",
      "items": {
        "type": "object",
        "properties": {
          "geometry": {"type": "string", "description": "Reference key of the face/edge the frame applies to (from get_reference_keys)."},
          "characteristic": {"type": "string", "description": "Geometric characteristic symbol, e.g. flatness, position, perpendicularity, circularRunout."},
          "value": {"type": "string", "description": "Tolerance zone size, e.g. \"0.05 mm\"."},
          "datums": {"type": "array", "items": {"type": "string"}, "description": "Referenced datum labels, e.g. [\"A\",\"B\"]."}
        },
        "required": ["geometry", "characteristic", "value"]
      }
    },
    "datums": {
      "type": "array",
      "description": "Datum features: model geometry tagged with a datum label.",
      "items": {
        "type": "object",
        "properties": {
          "geometry": {"type": "string", "description": "Reference key of the datum face/edge (from get_reference_keys)."},
          "label": {"type": "string", "description": "Datum label, e.g. \"A\"."}
        },
        "required": ["geometry", "label"]
      }
    }
  }
}`

func modelToleranceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "modelTolerance", Summary: "Annotate model geometry with GD&T feature-control frames and datums (no geometry change).", Schema: json.RawMessage(modelToleranceSchema), Apply: applyModelTolerance}
}

func applyModelTolerance(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in modelToleranceArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	if len(in.Frames) == 0 && len(in.Datums) == 0 {
		return nil, fmt.Errorf("modelTolerance: nothing to annotate (no frames, no datums)")
	}
	def, err := toleranceDefFromArgs(part, in)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, feature.NewToleranceFeatures(part.Features()).AddModelTolerance(def))
}

// toleranceDefFromArgs decodes the GD&T records, resolving characteristic spellings and value
// expressions; an unknown characteristic is a precise error.
func toleranceDefFromArgs(part *compdef.PartComponentDefinition, in modelToleranceArgs) (*feature.ModelToleranceDefinition, error) {
	def := &feature.ModelToleranceDefinition{}
	for i, fr := range in.Frames {
		ch, ok := types.ParseGeometricCharacteristic(fr.Characteristic)
		if !ok {
			return nil, fmt.Errorf("modelTolerance: frames[%d] unknown characteristic %q", i, fr.Characteristic)
		}
		value, err := lengthClosure(part, fr.Value, fmt.Sprintf("modelTolerance: frames[%d].value", i))
		if err != nil {
			return nil, err
		}
		def.Frames = append(def.Frames, feature.ToleranceFrame{
			GeometryKey:    []byte(fr.Geometry),
			Characteristic: ch,
			Value:          value(),
			Datums:         append([]string(nil), fr.Datums...),
		})
	}
	for _, dl := range in.Datums {
		def.Datums = append(def.Datums, feature.DatumLabel{GeometryKey: []byte(dl.Geometry), Label: dl.Label})
	}
	return def, nil
}
