// SPDX-License-Identifier: GPL-2.0-only

package topo

import "bytes"

// Occurrence-relative reference resolution (Oblikovati/Oblikovati#735). An assembly
// machining feature runs on placed component bodies, each transformed under a per-occurrence
// lineage prefix (the recompute prepends an "assemblyFeature:occ#i" token to every entity's
// lineage). A bare component-local reference key therefore never matches a placed edge/face
// by exact key. But because the prefix is only PREPENDED, the component's lineage survives as
// a SUFFIX of the placed entity's lineage — so a feature can store the component-local key and
// recover each participant's full key by matching that suffix, then hand the full key to the
// existing dress-up ops (chamfer/fillet/move-face), which resolve by exact key.

// LineageSuffixOf strips the leading entity-kind byte from a reference key, yielding the bare
// lineage key — the portion that survives as a suffix when the entity's body is transformed
// under a lineage-prefixing derive. Use it to turn a component-local reference key into a
// suffix to match against a placed body.
func LineageSuffixOf(referenceKey []byte) []byte {
	if len(referenceKey) == 0 {
		return referenceKey
	}
	return referenceKey[1:]
}

// EdgeReferenceKeysWithLineageSuffix returns the full reference keys of every edge whose
// lineage key ends with suffix at a token boundary — recovering, on a placed/transformed
// body, the edges that descend from a component-local edge regardless of any prepended
// lineage prefix (#735). An empty suffix matches nothing.
func (b *Body) EdgeReferenceKeysWithLineageSuffix(suffix []byte) [][]byte {
	if len(suffix) == 0 {
		return nil
	}
	var out [][]byte
	for _, e := range b.Edges() {
		if lineageKeyHasSuffix(e.Lineage().Key(), suffix) {
			out = append(out, e.ReferenceKey())
		}
	}
	return out
}

// FaceReferenceKeysWithLineageSuffix is the face twin of
// [Body.EdgeReferenceKeysWithLineageSuffix].
func (b *Body) FaceReferenceKeysWithLineageSuffix(suffix []byte) [][]byte {
	if len(suffix) == 0 {
		return nil
	}
	var out [][]byte
	for _, f := range b.Faces() {
		if lineageKeyHasSuffix(f.Lineage().Key(), suffix) {
			out = append(out, f.ReferenceKey())
		}
	}
	return out
}

// lineageKeyHasSuffix reports whether full equals suffix or ends with "/" + suffix, so the
// match always falls on a lineage-token boundary ("edge#1" never matches "edge#10").
func lineageKeyHasSuffix(full, suffix []byte) bool {
	if bytes.Equal(full, suffix) {
		return true
	}
	return len(full) > len(suffix) && full[len(full)-len(suffix)-1] == '/' && bytes.HasSuffix(full, suffix)
}
