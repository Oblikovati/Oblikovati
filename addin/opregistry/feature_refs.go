// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/param"
)

// Shared plumbing for the topology-referencing feature operations (fillet, chamfer, shell,
// draft, hole). Over the JSON API there is no viewport pick, so these operations take
// persistent reference-key strings (from model.referenceKeys / get_reference_keys) and the
// model resolves them lazily against the current topology — the same keys edges/faces report.

// refKeys converts reference-key strings to the model's [][]byte key form. The keys are the
// exact strings model.referenceKeys emits (string(topo.ReferenceKey())), so the byte round
// trip is lossless and resolves against the body they came from.
func refKeys(refs []string) [][]byte {
	keys := make([][]byte, len(refs))
	for i, r := range refs {
		keys[i] = []byte(r)
	}
	return keys
}

// lengthValue parses a unit-bearing length ("5 mm") in database units, naming field on error.
func lengthValue(part *compdef.PartComponentDefinition, expr, field string) (float64, error) {
	v, err := part.Units().Parse(expr, param.Length)
	if err != nil {
		return 0, fmt.Errorf("%s %q: %w", field, expr, err)
	}
	return v.Value, nil
}

// angleValue parses a unit-bearing angle ("3 deg") in database units (radians), naming field.
func angleValue(part *compdef.PartComponentDefinition, expr, field string) (float64, error) {
	v, err := part.Units().Parse(expr, param.Angle)
	if err != nil {
		return 0, fmt.Errorf("%s %q: %w", field, expr, err)
	}
	return v.Value, nil
}

// featureResult is the common reply for a feature operation: the created feature's name and
// kind, the resulting body count, and its health (so a caller learns whether the kernel could
// build the geometry — an unhealthy result is reported, not hidden).
type featureResult struct {
	Feature string `json:"feature"`
	Kind    string `json:"kind"`
	Bodies  int    `json:"bodies"`
	Healthy bool   `json:"healthy"`
	Reason  string `json:"reason,omitempty"`
}

// lastFeatureResult recomputes and reports the most recently added feature — used by the
// builders (pattern/mirror) that register through the engine and return a definition object
// rather than the *PartFeature the engine wraps it in.
func lastFeatureResult(part *compdef.PartComponentDefinition) (json.RawMessage, error) {
	feats := part.Features()
	if feats.Count() == 0 {
		return nil, fmt.Errorf("no feature was added")
	}
	return recomputeResult(part, feats.Item(feats.Count()-1))
}

// recomputeResult recomputes the part and reports the new feature's outcome. The feature stays
// in the tree even when unhealthy, so model.tree / referenceKeys can inspect it.
func recomputeResult(part *compdef.PartComponentDefinition, pf *feature.PartFeature) (json.RawMessage, error) {
	part.Recompute()
	h := pf.Health()
	return json.Marshal(featureResult{
		Feature: pf.Name(), Kind: pf.Kind(), Bodies: len(part.SurfaceBodies().All()),
		Healthy: h.OK(), Reason: h.Reason,
	})
}

// constFn wraps a constant as the func() float64 the model builders take for live parameters.
func constFn(v float64) func() float64 { return func() float64 { return v } }

// constIntFn wraps a constant as the func() int the pattern builders take for live counts.
func constIntFn(v int) func() int { return func() int { return v } }

// axisFromRef resolves a work-axis reference (an origin axis like "origin/axis/y", or a user
// axis) to the part's work axis. An empty ref defaults to the Y origin axis (a common revolve
// axis). It errors when the ref names no axis.
func axisFromRef(part *compdef.PartComponentDefinition, ref string) (*feature.WorkAxis, error) {
	if ref == "" {
		ref = string(feature.OriginYAxis)
	}
	axis, ok := part.WorkGeometry().AxisByRef(feature.WorkRef(ref))
	if !ok {
		return nil, fmt.Errorf("axis ref %q not found (try origin/axis/x|y|z)", ref)
	}
	return axis, nil
}

// featureIDByName resolves an existing feature's name (as shown in model.tree, e.g.
// "Extrusion1") to its id — the handle pattern/mirror replicate. Errors when unknown.
func featureIDByName(part *compdef.PartComponentDefinition, name string) (feature.ID, error) {
	f, ok := part.Features().ByName(name)
	if !ok {
		return 0, fmt.Errorf("feature %q not found (see model.tree)", name)
	}
	return f.ID(), nil
}

// featureIDsByName resolves several feature names to their ids.
func featureIDsByName(part *compdef.PartComponentDefinition, names []string) ([]feature.ID, error) {
	ids := make([]feature.ID, len(names))
	for i, n := range names {
		id, err := featureIDByName(part, n)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}

// vec3 / point3 convert a JSON [x,y,z] triple to model vectors/points, erroring on bad arity.
func vec3(a []float64, field string) (math.Vector3, error) {
	if len(a) != 3 {
		return math.Vector3{}, fmt.Errorf("%s needs 3 components [x,y,z], got %d", field, len(a))
	}
	return math.V3(math.Scalar(a[0]), math.Scalar(a[1]), math.Scalar(a[2])), nil
}

func point3(a []float64, field string) (math.Point3, error) {
	v, err := vec3(a, field)
	if err != nil {
		return math.Point3{}, err
	}
	return v.AsPoint(), nil
}
