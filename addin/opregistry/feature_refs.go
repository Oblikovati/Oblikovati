// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"
	"strings"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
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

// lengthClosure turns a length argument into a live value closure. A plain literal
// ("10 mm") becomes a constant; anything else (a parameter reference like "h" or an
// expression like "h+2 mm") is backed by an auto-named model parameter so the
// feature argument joins the parameter DAG and tracks edits — the same mechanism
// that makes sketch dimensions parametric. The closure re-reads the parameter, so a
// recompute after a parameter change rebuilds the feature at the new size.
func lengthClosure(part *compdef.PartComponentDefinition, expr, field string) (func() float64, error) {
	return valueClosure(part, expr, field, param.Length)
}

// angleClosure is lengthClosure for an angle argument (revolve angle, taper).
func angleClosure(part *compdef.PartComponentDefinition, expr, field string) (func() float64, error) {
	return valueClosure(part, expr, field, param.Angle)
}

// numberClosure turns a unitless argument (a count like coil revolutions or pattern
// occurrences) into a live value closure, parameter-aware like lengthClosure.
func numberClosure(part *compdef.PartComponentDefinition, expr, field string) (func() float64, error) {
	return valueClosure(part, expr, field, param.Unitless)
}

// optionalAngleClosure is angleClosure that yields a constant 0 for a blank argument.
func optionalAngleClosure(part *compdef.PartComponentDefinition, expr, field string) (func() float64, error) {
	if strings.TrimSpace(expr) == "" {
		return func() float64 { return 0 }, nil
	}
	return angleClosure(part, expr, field)
}

// valueClosure is the shared core of the *Closure helpers: a plain unit literal
// ("10 mm") becomes a constant closure; any expression (a parameter reference like
// "h", or "h+2 mm") is backed by an auto-named model parameter so the argument joins
// the parameter DAG and tracks edits — set_parameter marks features dirty and the
// next recompute re-reads the closure at the new value. EVERY feature/tool numeric
// argument must flow through one of these so it is both parameter-aware and
// dirty-recompute-correct (see addin/opregistry/doc.go).
func valueClosure(part *compdef.PartComponentDefinition, expr, field string, unit param.Unit) (func() float64, error) {
	if v, err := part.Units().Parse(expr, unit); err == nil {
		val := v.Value
		return func() float64 { return val }, nil
	}
	p, err := part.Parameters().AddAutoModelParameter(expr)
	if err != nil {
		return nil, fmt.Errorf("%s %q: %w", field, expr, err)
	}
	if h := p.Health(); !h.OK() {
		return nil, fmt.Errorf("%s %q: %s", field, expr, h.Reason)
	}
	return func() float64 { return p.ModelValue() }, nil
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
