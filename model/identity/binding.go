// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"bytes"

	"oblikovati.org/math"
)

// fallbackMatch recovers a reference whose exact entity is gone by degrading
// through the literature's tiers (Kripac 1997; FreeCAD 1.0 element maps): a lone
// surviving sibling that shares the key's parent lineage binds ancestrally; when
// several siblings survive, geometric nearness to the key's mint-time anchor picks
// one. Both are auto-healed (caller maps them to Warning, not Sick). A key with no
// recorded parent — a root lineage, or a key reloaded from disk before F07 versions
// the encoding — has nothing to fall back to and stays MatchNone (M31-F06, #1156).
func fallbackMatch(k RefKey, ents []Entity) (Entity, MatchType) {
	if !k.parent.ok {
		return nil, MatchNone
	}
	siblings := survivingSiblings(k, ents)
	switch len(siblings) {
	case 0:
		return nil, MatchNone
	case 1:
		return siblings[0], MatchAncestral
	default:
		return nearestSibling(k, siblings)
	}
}

// survivingSiblings returns the entities of the key's kind whose lineage declares
// the same parent as the key — the candidates for ancestral recovery. Entities
// whose lineage cannot name a parent are not siblings and are skipped.
func survivingSiblings(k RefKey, ents []Entity) []Entity {
	var out []Entity
	for _, e := range ents {
		if e.EntityKind() != k.kind {
			continue
		}
		al, ok := e.Lineage().(AncestralLineage)
		if ok && bytes.Equal(al.ParentKey(), k.parent.key) {
			out = append(out, e)
		}
	}
	return out
}

// nearestSibling disambiguates several same-parent siblings by distance to the
// key's mint-time anchor, returning MatchGeometric for a clear winner. Without an
// anchor, with no anchored siblings, or when the two closest tie, the choice is not
// defensible and the reference stays lost (MatchNone) rather than binding wrongly.
func nearestSibling(k RefKey, siblings []Entity) (Entity, MatchType) {
	if !k.anchor.ok {
		return nil, MatchNone
	}
	best, second := closestTwo(k.anchor.point, siblings)
	if best.entity == nil {
		return nil, MatchNone
	}
	if second.entity != nil && best.dist == second.dist {
		return nil, MatchNone
	}
	return best.entity, MatchGeometric
}

// rankedSibling pairs a sibling with its squared distance to the anchor.
type rankedSibling struct {
	entity Entity
	dist   math.Scalar
}

// closestTwo returns the nearest and second-nearest anchored siblings to p. A
// returned entry has a nil entity when fewer than that many siblings expose an
// anchor; an equal best/second distance signals an unbreakable geometric tie.
func closestTwo(p math.Point3, siblings []Entity) (best, second rankedSibling) {
	for _, e := range siblings {
		d, ok := anchorDistanceSq(p, e)
		if !ok {
			continue
		}
		cand := rankedSibling{entity: e, dist: d}
		switch {
		case best.entity == nil || d < best.dist:
			second = best
			best = cand
		case second.entity == nil || d < second.dist:
			second = cand
		}
	}
	return best, second
}

// anchorDistanceSq returns the squared distance from p to e's anchor, or false when
// e exposes no anchor (such a sibling cannot be ranked geometrically).
func anchorDistanceSq(p math.Point3, e Entity) (math.Scalar, bool) {
	ae, ok := e.(AnchoredEntity)
	if !ok {
		return 0, false
	}
	q, ok := ae.Anchor()
	if !ok {
		return 0, false
	}
	v := p.VectorTo(q)
	return v.LengthSquared(), true
}
