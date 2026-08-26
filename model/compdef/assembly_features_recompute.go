// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"errors"
	"strings"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/health"
	"oblikovati.org/model/occurrence"
)

// Recompute evaluates the assembly feature program against placed (the assembly's
// flattened, occurrence-attributed bodies). It machines each unsuppressed feature, up
// to the end-of-features marker, into the assembly-space copy of each participating
// contribution — never the shared part definitions — and records the per-contribution
// result keyed by occurrence path. Each feature's health reflects how its participants
// evaluated.
//
// Contributions are keyed by occurrence PATH, not by leaf occurrence, so a sub-assembly
// placed more than once is machined independently per placement — a feature can target
// one placement via [AssemblyFeature.SetParticipantPaths] without affecting the other.
func (fs *AssemblyFeatures) Recompute(placed []feature.PlacedBody) {
	groups, leaves := groupByPath(placed)
	end := fs.effectiveEnd()
	for i := range end {
		fs.evaluate(fs.items[i], groups, leaves)
	}
	fs.result, fs.resultLeaf = groups, leaves
	fs.raiseRecomputed()
}

// Result returns o's machined assembly-space bodies after the last recompute,
// aggregating every path that ends at o (one path for a flat placement, several when o
// is a leaf of a sub-assembly placed more than once). Nil if o did not participate. The
// returned slice is freshly built but holds the live bodies, so copy before mutating.
func (fs *AssemblyFeatures) Result(o *occurrence.Occurrence) []*topo.Body {
	var out []*topo.Body
	for key, leaf := range fs.resultLeaf {
		if leaf == o {
			out = append(out, fs.result[key]...)
		}
	}
	return out
}

// ResultPath returns the machined bodies of one specific placement (occurrence path),
// disambiguating a shared flyweight reached through several placements.
func (fs *AssemblyFeatures) ResultPath(path occurrence.OccurrencePath) []*topo.Body {
	return fs.result[pathKey(path)]
}

// evaluate machines one feature into the contributions it participates on, aggregating
// the outcome into its health. A suppressed feature passes every contribution through
// unchanged (the reference behavior), mirroring the part engine's suppressed-passthrough.
func (fs *AssemblyFeatures) evaluate(af *AssemblyFeature, groups map[string][]*topo.Body, leaves map[string]*occurrence.Occurrence) {
	if af.suppress {
		af.health = health.Health{Status: health.Suppressed}
		return
	}
	deferred, sick := false, false
	for key, bodies := range groups {
		if !af.participatesContribution(leaves[key], key) {
			continue
		}
		out, err := af.feature.Recompute(feature.Input{Bodies: bodies})
		switch {
		case err == nil:
			groups[key] = out.Bodies
		case isDeferred(err):
			deferred = true
		default:
			sick = true // a failed contribution keeps its pre-feature geometry (passthrough)
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

// groupByPath transforms each placed body into its assembly-space copy and groups the
// copies by occurrence-path key, recording the leaf occurrence each path ends at. A
// body whose transform is degenerate is skipped (it cannot be placed); such cases do
// not occur for rigid occurrence placements but are handled rather than panicking.
func groupByPath(placed []feature.PlacedBody) (map[string][]*topo.Body, map[string]*occurrence.Occurrence) {
	groups := map[string][]*topo.Body{}
	leaves := map[string]*occurrence.Occurrence{}
	for i, pb := range placed {
		copyBody, err := ops.TransformBody(pb.Body, pb.Transform, asmFeatureLineage(i))
		if err != nil {
			continue
		}
		key := pathKey(pb.Path)
		groups[key] = append(groups[key], copyBody)
		leaves[key] = pb.Source
	}
	return groups, leaves
}

// pathKey is the map key for an occurrence path. Instance names cannot contain the NUL
// separator, so the join is unambiguous.
func pathKey(p occurrence.OccurrencePath) string { return strings.Join(p, "\x00") }

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
