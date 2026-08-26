// SPDX-License-Identifier: GPL-2.0-only

package depend

import "slices"

// Footprint is the set of keys one consumer (a sketch solve, a feature evaluation, a work
// plane's recompute) read during the last recompute. The engine's whole correctness rests
// on one rule: a consumer is dirtied by a change exactly when the change touches a key in
// its footprint, and any read NOT captured in some footprint is treated as unattributable
// and forces a wholesale rebuild — never silently skipped (ADR-0044, Oblikovati#1414/#1403).
type Footprint = []Key

// Set is a membership view of a change-set, built once per edit so attribution is O(1) per
// footprint key instead of O(changed).
type Set map[Key]struct{}

// NewSet builds a membership set from a change-set slice.
func NewSet(keys []Key) Set {
	s := make(Set, len(keys))
	for _, k := range keys {
		s[k] = struct{}{}
	}
	return s
}

// Has reports membership.
func (s Set) Has(k Key) bool { _, ok := s[k]; return ok }

// Empty reports whether the set has no keys (an edit that changed nothing the graph
// recorded — the caller then rebuilds conservatively).
func (s Set) Empty() bool { return len(s) == 0 }

// Intersects reports whether any key in footprint is in changed. This is the single
// attribution predicate: a consumer is affected by the change iff its footprint intersects
// the change-set. It is kind-agnostic — a ParameterKey and an ExternalGeometryKey are
// compared the same way — which is exactly what lets a future adaptive reference reuse it.
func Intersects(footprint []Key, changed Set) bool {
	return slices.ContainsFunc(footprint, changed.Has)
}
