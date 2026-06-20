// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// listFeatureKinds returns every feature operation the bridge can create, from the
// operation registry (so it grows as descriptors are registered, no code here).
func (r *Router) listFeatureKinds(_ *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	all := r.ops.All()
	out := make([]wire.FeatureKindInfo, len(all))
	for i, d := range all {
		out[i] = wire.FeatureKindInfo{Kind: d.Name, Summary: d.Summary, Schema: d.Schema}
	}
	return json.Marshal(wire.ListFeatureKindsResult{Kinds: out})
}

// addFeature applies the named operation's descriptor to the session.
func (r *Router) addFeature(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.AddFeatureArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	d, ok := r.ops.ByName(in.Kind)
	if !ok {
		return nil, fmt.Errorf("features.add: unknown kind %q (see features.list)", in.Kind)
	}
	args := in.Args
	if len(args) == 0 {
		args = json.RawMessage("{}")
	}
	out, err := d.Apply(s, args)
	if err != nil {
		return nil, err
	}
	s.EmitFeatureLifecycle(app.FeatureAdded, lastPartFeature(s)) // feature.added (#1085)
	return out, nil
}
