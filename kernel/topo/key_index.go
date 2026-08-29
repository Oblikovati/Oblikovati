// SPDX-License-Identifier: GPL-2.0-only

package topo

import "bytes"

// keyed is any topological entity addressable by a persistent reference key — the
// constraint the per-body reference-key index is built over (#1580).
type keyed interface {
	ReferenceKey() []byte
}

// aliasKeyed is a keyed entity that ALSO resolves under additional reference keys — a coplanar-merged
// boolean face carries each merged parent's key so a pick on any parent survives the merge (ADR-0057
// multi-parent identity). Only *Face implements it; the index/scan check for it structurally so the
// generic key path stays one implementation.
type aliasKeyed interface {
	AliasKeys() [][]byte
}

// buildKeyIndex groups entities by their reference key. Normally one entity per key;
// MORE than one is a topological-naming collision the ADR-0043 P0 resolution guard
// turns into an honest error instead of a silent first-match — so the index stores
// every entity per key, never just the first. Bucket order is the entity iteration
// order, so a first-match accessor (Find*ByKey) returns the same entity it always did.
// An entity that also implements aliasKeyed is additionally indexed under each alias key, so a
// coplanar-merged face resolves from every merged parent's key (ADR-0057).
func buildKeyIndex[T keyed](items []T) map[string][]T {
	idx := make(map[string][]T, len(items))
	for _, it := range items {
		idx[string(it.ReferenceKey())] = append(idx[string(it.ReferenceKey())], it)
		if ak, ok := any(it).(aliasKeyed); ok {
			for _, k := range ak.AliasKeys() {
				idx[string(k)] = append(idx[string(k)], it)
			}
		}
	}
	return idx
}

// containsKey reports whether keys already holds key.
func containsKey(keys [][]byte, key []byte) bool {
	for _, k := range keys {
		if bytes.Equal(k, key) {
			return true
		}
	}
	return false
}

// scanByKey is the un-indexed fallback used before a body is finalized: it linearly
// collects entities matching key. A partially-built body (RegroupShells querying
// mid-regroup, see [Body]) has no stable index yet because its geometry is still
// changing, so building one would memoize an incomplete answer.
func scanByKey[T keyed](items []T, key []byte) []T {
	var out []T
	for _, it := range items {
		if bytes.Equal(it.ReferenceKey(), key) {
			out = append(out, it)
			continue
		}
		if ak, ok := any(it).(aliasKeyed); ok && containsKey(ak.AliasKeys(), key) {
			out = append(out, it)
		}
	}
	return out
}
