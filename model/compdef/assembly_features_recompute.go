// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"errors"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/health"
	"oblikovati.org/model/occurrence"
)

// Recompute evaluates the assembly feature program against placed (the assembly's
// flattened, occurrence-attributed bodies). It machines each unsuppressed feature, up
// to the end-of-features marker, into the assembly-space copy of each participant
// occurrence's bodies — never the shared part definitions — and records the per-
// occurrence result. Each feature's health reflects how its participants evaluated.
//
// The shared definitions stay untouched because the running bodies are assembly-space
// COPIES (each placed body is transformed into assembly space first), so a boolean
// rewrites the copy, not the part's own B-rep.
func (fs *AssemblyFeatures) Recompute(placed []feature.PlacedBody) {
	groups := groupPlacedBodies(placed)
	end := fs.effectiveEnd()
	for i := 0; i < end; i++ {
		fs.evaluate(fs.items[i], groups)
	}
	fs.result = groups
}

// Result returns o's machined assembly-space bodies after the last recompute, or nil
// if o did not participate (or no recompute has run). The returned slice is the live
// result, so callers that mutate it must copy first.
func (fs *AssemblyFeatures) Result(o *occurrence.Occurrence) []*topo.Body {
	return fs.result[o]
}

// evaluate machines one feature into its participants' running bodies, aggregating the
// outcome into its health. A suppressed feature passes every participant through
// unchanged (the reference behavior: a suppressed assembly feature contributes no
// machining), mirroring the part engine's suppressed-passthrough.
func (fs *AssemblyFeatures) evaluate(af *AssemblyFeature, groups map[*occurrence.Occurrence][]*topo.Body) {
	if af.suppress {
		af.health = health.Health{Status: health.Suppressed}
		return
	}
	deferred, sick := false, false
	for _, o := range af.order {
		bodies, present := groups[o]
		if !present {
			continue // a participant with no current geometry (e.g. suppressed occurrence)
		}
		out, err := af.feature.Recompute(feature.Input{Bodies: bodies})
		switch {
		case err == nil:
			groups[o] = out.Bodies
		case isDeferred(err):
			deferred = true
		default:
			sick = true // a failed participant keeps its pre-feature geometry (passthrough)
		}
	}
	af.health = featureHealth(deferred, sick)
}

// featureHealth maps a feature's aggregated participant outcomes to a single health,
// mirroring the part engine's classify policy: any hard failure sickens the feature,
// otherwise a deferred participant warns, otherwise it is healthy.
func featureHealth(deferred, sick bool) health.Health {
	switch {
	case sick:
		return health.Sicken("assembly feature failed on a participant occurrence")
	case deferred:
		return health.Health{Status: health.Warning, Reason: feature.ErrDeferred.Error()}
	default:
		return health.Healthy
	}
}

// isDeferred reports whether err is (or wraps) the engine's deferred-geometry signal,
// so a participant whose feature only deferred warns rather than sickens — matching the
// part engine's errors.Is classification.
func isDeferred(err error) bool { return errors.Is(err, feature.ErrDeferred) }

// groupPlacedBodies transforms each placed body into its assembly-space copy and groups
// the copies by the occurrence that places them. A body whose transform is degenerate
// is skipped (it cannot be placed); such cases do not occur for rigid occurrence
// placements but are handled rather than panicking.
func groupPlacedBodies(placed []feature.PlacedBody) map[*occurrence.Occurrence][]*topo.Body {
	groups := map[*occurrence.Occurrence][]*topo.Body{}
	for i, pb := range placed {
		copyBody, err := ops.TransformBody(pb.Body, pb.Transform, asmFeatureLineage(i))
		if err != nil {
			continue
		}
		groups[pb.Source] = append(groups[pb.Source], copyBody)
	}
	return groups
}

// asmFeatureLineage gives each placed copy a distinct lineage prefix, so the same part
// placed at several occurrences yields independent reference keys in the machined
// result (mirrors the derived-assembly flatten's per-occurrence lineage).
func asmFeatureLineage(index int) func(topo.Lineage) topo.Lineage {
	prefix := topo.Tok("assemblyFeature", "occ", index)
	return func(l topo.Lineage) topo.Lineage {
		return topo.NewLineage(append([]topo.LineageToken{prefix}, l.Tokens()...)...)
	}
}

// distinctSources returns the occurrences that contribute geometry to placed, in
// first-seen order — the default participation an assembly feature snapshots (every
// component present participates). It does not transform bodies; it only lists the
// source occurrences.
func distinctSources(placed []feature.PlacedBody) []*occurrence.Occurrence {
	seen := map[*occurrence.Occurrence]bool{}
	var order []*occurrence.Occurrence
	for _, pb := range placed {
		if seen[pb.Source] {
			continue
		}
		seen[pb.Source] = true
		order = append(order, pb.Source)
	}
	return order
}
