// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"bytes"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/identity"
)

// This file is the anticorruption adapter between the kernel's topology
// (kernel/topo) and the identity package's tiered binder (model/identity), the one
// place both are visible without inverting the dependency rule (ADR-0043 P6). It
// lets a lost dress-up edge reference be recovered through the same tested binder
// the F06 key manager uses, instead of going Sick on the first exact miss.

// parentOfKey returns the PARENT lineage of a kernel reference key — the key's
// lineage with its most-specific token dropped — computed identically for a lost key
// and for every candidate sibling, so the two compare. A reference key is a kind
// byte followed by the lineage string "feat:role#idx/feat:role#idx/…" (kernel/topo
// referenceKey + Lineage.Key); the parent is everything before the final '/'. A
// single-token (root) lineage has no parent and returns nil — no ancestral fallback.
func parentOfKey(refKey []byte) []byte {
	s := refKey
	if len(s) > 0 && s[0] < 0x20 { // strip the leading kind byte (kinds are < 0x20)
		s = s[1:]
	}
	i := bytes.LastIndexByte(s, '/')
	if i < 0 {
		return nil
	}
	return append([]byte(nil), s[:i]...)
}

// edgeLineage adapts a kernel edge's reference key to identity.Lineage +
// AncestralLineage. ParentKey derives the parent the same way for every entity (see
// parentOfKey), which is what makes ancestral sibling matching consistent.
type edgeLineage []byte // the edge's kernel reference key (kind byte + lineage)

func (l edgeLineage) LineageKey() []byte {
	if len(l) > 0 && l[0] < 0x20 {
		return append([]byte(nil), l[1:]...)
	}
	return append([]byte(nil), l...)
}

func (l edgeLineage) ParentKey() []byte { return parentOfKey(l) }

// edgeEntity adapts a *topo.Edge to identity.Entity (+ AncestralLineage for
// ancestral recovery, + AnchoredEntity for the geometric tie-break used by P6b). It
// carries the live edge so a recovered match can be unwrapped back to it.
type edgeEntity struct{ e *topo.Edge }

func (a edgeEntity) EntityKind() identity.EntityKind { return identity.KindEdge }
func (a edgeEntity) Lineage() identity.Lineage       { return edgeLineage(a.e.ReferenceKey()) }

// Anchor reports the edge midpoint as its representative point — the geometric
// tie-breaker the binder uses only when several siblings share a parent (P6b). It shares
// edgeMidpoint with the create-time capture so witness and ranking use one definition.
func (a edgeEntity) Anchor() (math.Point3, bool) {
	return edgeMidpoint(a.e)
}

// edgeEntities views every edge of the body as an identity.Entity for the binder.
func edgeEntities(body *topo.Body) []identity.Entity {
	edges := body.Edges()
	out := make([]identity.Entity, len(edges))
	for i, e := range edges {
		out[i] = edgeEntity{e}
	}
	return out
}

// currentKeys returns each resolved edge's CURRENT reference key — the same bytes as
// the stored key for an exact match, the recovered sibling's live key for a healed one.
// Callers that feed kernel ops (which re-resolve by exact key) use it so a healed edge
// is addressed by a key the kernel can match.
func currentKeys(edges []*topo.Edge) [][]byte {
	keys := make([][]byte, len(edges))
	for i, e := range edges {
		keys[i] = e.ReferenceKey()
	}
	return keys
}

// anchorFor returns the mint-time anchor stored for a reference key, or nil when none was
// captured (an older recipe, an edit-mode retained key) — in which case recovery falls back
// to the ancestral tier only.
func anchorFor(refKey []byte, anchors map[string]math.Point3) *math.Point3 {
	if anchors == nil {
		return nil
	}
	if p, ok := anchors[string(refKey)]; ok {
		return &p
	}
	return nil
}

// recoverEdge attempts to recover a lost/ambiguous edge reference through the tiered binder:
// a lone surviving sibling sharing the key's parent lineage binds ancestrally; with a mint-time
// anchor, several surviving siblings are disambiguated by nearness (the geometric tier, P6b).
// It returns the recovered edge and the match tier, or (nil, MatchNone) when no defensible
// recovery exists.
func recoverEdge(refKey []byte, anchor *math.Point3, ents []identity.Entity) (*topo.Edge, identity.MatchType) {
	ent, mt := identity.RecoverLost(identity.KindEdge, parentOfKey(refKey), anchor, ents)
	if ent == nil {
		return nil, mt
	}
	return ent.(edgeEntity).e, mt
}
