// SPDX-License-Identifier: GPL-2.0-only

package topo

import "bytes"

// keyed is any topological entity addressable by a persistent reference key — the
// constraint the per-body reference-key index is built over (#1580).
type keyed interface {
	ReferenceKey() []byte
}

// buildKeyIndex groups entities by their reference key. Normally one entity per key;
// MORE than one is a topological-naming collision the ADR-0043 P0 resolution guard
// turns into an honest error instead of a silent first-match — so the index stores
// every entity per key, never just the first. Bucket order is the entity iteration
// order, so a first-match accessor (Find*ByKey) returns the same entity it always did.
func buildKeyIndex[T keyed](items []T) map[string][]T {
	idx := make(map[string][]T, len(items))
	for _, it := range items {
		k := string(it.ReferenceKey())
		idx[k] = append(idx[k], it)
	}
	return idx
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
		}
	}
	return out
}
