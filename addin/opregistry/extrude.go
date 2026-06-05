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
	SketchIndex    int    `json:"sketchIndex"`
	ProfileIndex   int    `json:"profileIndex"`
	Distance       string `json:"distance"`                 // e.g. "50 mm", "5 cm"
	Operation      string `json:"operation"`                // join|cut|intersect|new (default new)
	Extent         string `json:"extent,omitempty"`         // distance|through-all|to-next (default distance)
	Direction      string `json:"direction,omitempty"`      // positive|negative|symmetric (default positive)
	SecondDistance string `json:"secondDistance,omitempty"` // asymmetric two-direction depth
	Taper          string `json:"taper,omitempty"`          // draft angle, e.g. "3 deg"
}

const extrudeSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Index of the sketch to extrude (see model.tree)."},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0, "description": "Which closed profile of the sketch to extrude."},
    "distance": {"type": "string", "description": "Extrude depth with units, e.g. \"50 mm\" or \"5 cm\". Required for distance and distance-from-face extents."},
    "operation": {"type": "string", "enum": ["new", "join", "cut", "intersect"], "default": "new", "description": "Boolean against existing bodies."},
    "extent": {"type": "string", "enum": ["distance", "through-all", "to-next"], "default": "distance", "description": "How the extrude terminates."},
    "direction": {"type": "string", "enum": ["positive", "negative", "symmetric"], "default": "positive", "description": "Which side(s) of the sketch plane to grow."},
    "secondDistance": {"type": "string", "description": "Asymmetric two-direction depth on the negative side, e.g. \"10 mm\"."},
    "taper": {"type": "string", "description": "Draft angle, e.g. \"3 deg\" (positive widens away from the sketch)."}
  },
  "required": ["sketchIndex"]
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
	in, sk, extent, op, taper, err := resolveExtrude(part, raw)
	if err != nil {
		return nil, err
	}
	pf := feature.NewExtrudeFeatures(part.Features()).
		AddExtrude(sk, []int{in.ProfileIndex}, op, extent, taper)
	// Uniform feature result (feature/kind/bodies/healthy/reason), shared with every other
	// operation so callers read one shape — and so an unhealthy extrude is reported, not hidden.
	return recomputeResult(part, pf)
}

// resolveExtrude decodes and validates the extrude args against the part: the target
// sketch, the boolean operation, the full extent (type/direction/distances), and the
// taper, in database units.
func resolveExtrude(part *compdef.PartComponentDefinition, raw json.RawMessage) (extrudeArgs, *sketch.Sketch, feature.Extent, ops.PartFeatureOperation, float64, error) {
	var in extrudeArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return in, nil, feature.Extent{}, 0, 0, fmt.Errorf("extrude: invalid args: %w", err)
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return in, nil, feature.Extent{}, 0, 0, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return in, nil, feature.Extent{}, 0, 0, err
	}
	extent, err := buildExtent(part, in)
	if err != nil {
		return in, nil, feature.Extent{}, 0, 0, err
	}
	taper, err := parseTaperAngle(part, in.Taper)
	if err != nil {
		return in, nil, feature.Extent{}, 0, 0, err
	}
	return in, sk, extent, op, taper, nil
}

// buildExtent assembles the model extent from the request: the extent type and direction,
// plus the distance(s) for the distance extent.
func buildExtent(part *compdef.PartComponentDefinition, in extrudeArgs) (feature.Extent, error) {
	etype, err := parseExtentType(in.Extent)
	if err != nil {
		return feature.Extent{}, err
	}
	ext := feature.Extent{Type: etype, Direction: parseExtentDirection(in.Direction)}
	if etype != feature.DistanceExtent {
		return ext, nil // through-all / to-next are gauged from the model, not a distance
	}
	d, err := part.Units().Parse(in.Distance, param.Length)
	if err != nil {
		return feature.Extent{}, fmt.Errorf("extrude: distance %q: %w", in.Distance, err)
	}
	if d.Value == 0 {
		return feature.Extent{}, errors.New("extrude: distance is zero")
	}
	ext.Distance = func() float64 { return d.Value }
	if in.SecondDistance != "" {
		d2, err := part.Units().Parse(in.SecondDistance, param.Length)
		if err != nil {
			return feature.Extent{}, fmt.Errorf("extrude: secondDistance %q: %w", in.SecondDistance, err)
		}
		ext.Distance2 = func() float64 { return d2.Value }
	}
	return ext, nil
}

// parseTaperAngle parses an optional draft angle ("" ⇒ no taper).
func parseTaperAngle(part *compdef.PartComponentDefinition, expr string) (float64, error) {
	if strings.TrimSpace(expr) == "" {
		return 0, nil
	}
	a, err := part.Units().Parse(expr, param.Angle)
	if err != nil {
		return 0, fmt.Errorf("extrude: taper %q: %w", expr, err)
	}
	return a.Value, nil
}

// parseExtentType maps an extent name to its model type (empty ⇒ distance). Reference
// extents (to-face / from-to / distance-from-face) need a target plane and are not yet
// exposed over the JSON API.
func parseExtentType(name string) (feature.ExtentType, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "distance":
		return feature.DistanceExtent, nil
	case "through-all", "throughall":
		return feature.ThroughAllExtent, nil
	case "to-next", "tonext":
		return feature.ToNextExtent, nil
	default:
		return 0, fmt.Errorf("extrude: unknown extent %q (want distance|through-all|to-next)", name)
	}
}

// parseExtentDirection maps a direction name to its model value (empty/unknown ⇒ positive).
func parseExtentDirection(name string) feature.ExtentDirection {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "negative":
		return feature.NegativeDir
	case "symmetric":
		return feature.SymmetricDir
	default:
		return feature.PositiveDir
	}
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
