// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"oblikovati.org/math"
)

// ContextID identifies a key context (a versioned topology snapshot) within a
// [KeyManager]. It is part of a key's identity and is preserved across save/load.
type ContextID uint64

// RefKey is an opaque, serializable reference to a topological/model entity. It is
// a value type (no pointers), so it can be stored in a feature definition, a
// document descriptor, or a persisted stream and rebound later. Callers treat it
// as opaque; only the [KeyManager] interprets the payload (the entity's lineage).
type RefKey struct {
	ctx     ContextID
	kind    EntityKind
	payload []byte // the entity's LineageKey at mint time

	// Fallback hints captured at mint time to drive degraded re-binding when the
	// exact entity is gone (M31-F06). They are an in-memory SESSION optimisation: a
	// key reloaded from disk has no hints (they are not serialized — see encoding.go
	// for why the drift-prone anchor must stay out of the identity bytes) and so
	// degrades to exact-only. Both are absent unless the keyed entity exposed the
	// capability.
	parent ancestryHint // the parent lineage, for ancestral sibling recovery
	anchor anchorHint   // a representative point, for geometric tie-breaking

	// scheme is the lineage-encoding version this key was decoded from
	// (SchemeLegacy for a pre-M31 key), or SchemeCurrent for a freshly minted one.
	// It is provenance only: it is NOT part of the key's identity (see Equal) and
	// re-encoding always writes SchemeCurrent (M31-F07, #1157).
	scheme uint8
}

// ancestryHint records the keyed entity's parent lineage key, present only when
// the entity's lineage implemented [AncestralLineage] at mint time.
type ancestryHint struct {
	key []byte
	ok  bool
}

// anchorHint records the keyed entity's representative point, present only when
// the entity implemented [AnchoredEntity] at mint time.
type anchorHint struct {
	point math.Point3
	ok    bool
}

// Context returns the key's owning context id.
func (k RefKey) Context() ContextID { return k.ctx }

// Kind returns the kind of entity the key names.
func (k RefKey) Kind() EntityKind { return k.kind }

// Scheme returns the lineage-encoding version this key was decoded from
// ([SchemeLegacy] for a pre-M31 key), or [SchemeCurrent] for a freshly minted key
// (M31-F07). It is provenance for migration diagnostics, not part of identity.
func (k RefKey) Scheme() uint8 { return k.scheme }

// IsZero reports whether the key is the zero value (names nothing).
func (k RefKey) IsZero() bool {
	return k.ctx == 0 && k.kind == KindUnknown && len(k.payload) == 0
}

// Equal reports whether two keys are identical (same context, kind, and lineage).
func (k RefKey) Equal(other RefKey) bool {
	return k.ctx == other.ctx && k.kind == other.kind && bytes.Equal(k.payload, other.payload)
}

// MatchType classifies a bind attempt. It modernizes COM ReferenceKeyMatchType,
// trimmed to what the lineage-based design needs. Values ascend by quality
// (none < geometric < ancestral < exact), so a larger value is a stronger bind;
// MatchNone stays the zero value (the natural "no match"). The fallback tiers
// (M31-F06, #1156) let an edit that destroys the exact entity recover to a
// surviving sibling instead of fully losing the reference.
type MatchType uint8

const (
	// MatchNone: nothing in the current topology matches the key — the reference
	// is lost (a legitimate, non-fatal outcome).
	MatchNone MatchType = iota
	// MatchGeometric: no lineage match, but a lone ancestral sibling was selected
	// by geometric nearness to the key's mint-time anchor — auto-healed, flagged.
	MatchGeometric
	// MatchAncestral: no exact lineage, but a single surviving sibling shares the
	// key's parent lineage — auto-healed, flagged for review.
	MatchAncestral
	// MatchExact: a single entity with the same kind and lineage was found.
	MatchExact
)

// IsFallback reports whether the match was recovered by a degraded tier
// (ancestral/geometric) rather than exact lineage — the binder's signal to
// resolve the reference to Warning health rather than fully healthy.
func (m MatchType) IsFallback() bool {
	return m == MatchAncestral || m == MatchGeometric
}

// String returns a stable name for diagnostics.
func (m MatchType) String() string {
	switch m {
	case MatchExact:
		return "exact"
	case MatchAncestral:
		return "ancestral"
	case MatchGeometric:
		return "geometric"
	default:
		return "none"
	}
}

// KeyToString renders a key as a URL-safe base64 string for transport and UI.
func KeyToString(k RefKey) string {
	return base64.URLEncoding.EncodeToString(k.Encode())
}

// StringToKey parses a key from its [KeyToString] form.
func StringToKey(s string) (RefKey, error) {
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return RefKey{}, fmt.Errorf("identity: invalid key string: %w", err)
	}
	return DecodeKey(data)
}
