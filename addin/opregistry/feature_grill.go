// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// Grill ventilation feature (M20-F10 #863): cut a vent through a thin wall, leaving the
// boundary profile's inner-loop structure (ribs/spars/islands) bridging it. The structure is
// drawn as holes of the boundary profile in the sketch.

const grillSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Index of the sketch holding the grill profiles (see model.tree)."},
    "boundaries": {"type": "array", "items": {"type": "integer", "minimum": 0}, "minItems": 1, "description": "Profile indices of the vent boundaries; each profile's inner loops are the kept ribs/spars/islands that bridge the vent."},
    "draft": {"type": "number", "description": "Draft angle (radians) tapering the vent walls. Optional."}
  },
  "required": ["sketchIndex", "boundaries"]
}`

func grillDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindGrill, Summary: "Cut a ventilation grill: a vent bridged by the boundary profile's rib/spar/island structure.", Schema: json.RawMessage(grillSchema), Apply: applyGrill}
}

func applyGrill(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Grill](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.Boundaries) == 0 {
		return nil, fmt.Errorf("grill: boundaries is empty")
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	def := &feature.GrillDefinition{Sketch: sk, Boundaries: append([]int(nil), in.Boundaries...), Draft: in.Draft}
	return recomputeResult(part, feature.NewGrillFeatures(part.Features()).Add(def))
}
