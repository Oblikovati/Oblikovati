// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/param"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// extrudeArgs is the argument shape for the "extrude" operation (mirrors the schema).
type extrudeArgs struct {
	SketchIndex  int    `json:"sketchIndex"`
	ProfileIndex int    `json:"profileIndex"`
	Distance     string `json:"distance"`  // e.g. "50 mm", "5 cm"
	Operation    string `json:"operation"` // join|cut|intersect|new (default new)
}

// extrudeResult reports the created feature and the resulting body count.
type extrudeResult struct {
	Feature string `json:"feature"`
	Kind    string `json:"kind"`
	Bodies  int    `json:"bodies"`
}

const extrudeSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Index of the sketch to extrude (see model.tree)."},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0, "description": "Which closed profile of the sketch to extrude."},
    "distance": {"type": "string", "description": "Extrude depth with units, e.g. \"50 mm\" or \"5 cm\"."},
    "operation": {"type": "string", "enum": ["new", "join", "cut", "intersect"], "default": "new", "description": "Boolean against existing bodies."}
  },
  "required": ["sketchIndex", "distance"]
}`

// extrudeDescriptor is the self-describing "extrude" operation: turn a closed sketch
// profile into a prism. It is the reference operation; further feature kinds follow
// the same shape (PBI: revolve, hole, fillet…).
func extrudeDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    "extrude",
		Summary: "Extrude a closed sketch profile into a solid prism.",
		Schema:  json.RawMessage(extrudeSchema),
		Apply:   applyExtrude,
	}
}

func applyExtrude(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	in, sk, op, depth, err := resolveExtrude(part, raw)
	if err != nil {
		return nil, err
	}
	pf := feature.NewExtrudeFeatures(part.Features()).
		AddByDistanceExtent(sk, in.ProfileIndex, op, func() float64 { return depth })
	part.Recompute()
	return json.Marshal(extrudeResult{Feature: pf.Name(), Kind: pf.Kind(), Bodies: len(part.SurfaceBodies().All())})
}

// resolveExtrude decodes and validates the extrude args against the part: the target
// sketch, the boolean operation, and the extent distance in database units.
func resolveExtrude(part *compdef.PartComponentDefinition, raw json.RawMessage) (extrudeArgs, *sketch.Sketch, ops.PartFeatureOperation, float64, error) {
	var in extrudeArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return in, nil, 0, 0, fmt.Errorf("extrude: invalid args: %w", err)
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return in, nil, 0, 0, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return in, nil, 0, 0, err
	}
	dist, err := part.Units().Parse(in.Distance, param.Length)
	if err != nil {
		return in, nil, 0, 0, fmt.Errorf("extrude: distance %q: %w", in.Distance, err)
	}
	if dist.Value == 0 {
		return in, nil, 0, 0, errors.New("extrude: distance is zero")
	}
	return in, sk, op, dist.Value, nil
}

// sketchAt returns the part's sketch at index i, bounds-checked.
func sketchAt(part *compdef.PartComponentDefinition, i int) (*sketch.Sketch, error) {
	sks := part.Sketches()
	if i < 0 || i >= sks.Count() {
		return nil, fmt.Errorf("extrude: sketch index %d out of range (part has %d sketches)", i, sks.Count())
	}
	return sks.Item(i), nil
}

// parseOperation maps an operation name to the boolean op (default new body).
func parseOperation(name string) (ops.PartFeatureOperation, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "new", "newbody", "new-body":
		return ops.NewBody, nil
	case "join":
		return ops.Join, nil
	case "cut":
		return ops.Cut, nil
	case "intersect":
		return ops.Intersect, nil
	default:
		return 0, fmt.Errorf("extrude: unknown operation %q (want new|join|cut|intersect)", name)
	}
}
