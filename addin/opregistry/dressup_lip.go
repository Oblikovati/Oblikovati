// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// --- lip / groove (M20-F10) ------------------------------------------------

const lipSchema = `{
  "type": "object",
  "properties": {
    "edgeRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the edges to run the bead along (from get_reference_keys)."},
    "width": {"type": "string", "description": "Bead width along the first adjacent face, e.g. \"2 mm\"."},
    "height": {"type": "string", "description": "Bead height along the second adjacent face, e.g. \"2 mm\"."},
    "groove": {"type": "boolean", "default": false, "description": "Cut a recessed groove instead of raising a lip."}
  },
  "required": ["edgeRefs", "width", "height"]
}`

func lipDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindLip, Summary: "Run a raised lip (or recessed groove) bead along picked edges.", Schema: json.RawMessage(lipSchema), Apply: applyLip}
}

func applyLip(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Lip](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.EdgeRefs) == 0 {
		return nil, errors.New("lip: edgeRefs is empty")
	}
	w, err := lengthClosure(part, in.Width, "lip: width")
	if err != nil {
		return nil, err
	}
	h, err := lengthClosure(part, in.Height, "lip: height")
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddLip(refKeys(in.EdgeRefs), w, h, in.Groove)
	return recomputeResult(part, pf)
}

// setChamferRun records which face the first setback is measured on and how much of each edge the
// bevel covers (#1888). Both are optional; absent, the chamfer runs the whole edge with the
// setbacks in the edge's own face order, exactly as before.
