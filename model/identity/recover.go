// SPDX-License-Identifier: GPL-2.0-only

package identity

import "oblikovati.org/math"

// RecoverLost attempts the degraded binding tiers (M31-F06; Kripac 1997) for a
// reference whose EXACT entity is gone from the current topology — the live
// counterpart of [KeyManager.BindKeyToObject]'s fallback, reachable without a key
// context for callers that resolve raw kernel reference keys (the dress-up features,
// ADR-0043 P6). It is a thin, pure wrapper over the same tested binder, so there is
// one recovery implementation, not two.
//
// parentKey is the lost key's PARENT lineage (its lineage with the most-specific
// step dropped): a lone surviving sibling that shares it binds ancestrally. anchor,
// when non-nil, is the reference's mint-time point: several surviving siblings are
// then disambiguated by geometric nearness to it (anchor nil → ancestral only).
// ents enumerates the current entities of the body. The result is [MatchNone] (nil
// entity) whenever no recovery is defensible — an empty parent, several siblings
// with no anchor, or a geometric tie — so recovery never binds the wrong entity.
//
// Example: e, m := identity.RecoverLost(identity.KindEdge, parent, nil, edgeEnts)
func RecoverLost(kind EntityKind, parentKey []byte, anchor *math.Point3, ents []Entity) (Entity, MatchType) {
	if len(parentKey) == 0 {
		return nil, MatchNone
	}
	k := RefKey{kind: kind, parent: ancestryHint{key: parentKey, ok: true}}
	if anchor != nil {
		k.anchor = anchorHint{point: *anchor, ok: true}
	}
	return fallbackMatch(k, ents)
}
