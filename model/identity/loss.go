// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"errors"
	"fmt"

	"oblikovati.org/model/health"
)

// ErrReferenceLost is the standard error behind a lost-reference health state. A
// consumer that depended on a now-unbindable key reports this — binding loss is a
// normal consequence of editing (a feature change can delete the referenced face),
// so the contract is to go sick and surface for re-selection, never to crash
// (parametric-cad §7, architecture core/05).
var ErrReferenceLost = errors.New("reference lost")

// Resolve binds a key and translates the outcome into modeling health — the single
// place the reference-loss policy lives, applied identically by every consumer
// (features, dimensions, mates). An exact match yields healthy state; a degraded
// match (ancestral/geometric, M31-F06) yields [health.Warning] so the reference is
// kept usable but flagged for the user to confirm; only a true loss yields
// [health.Sick] with a reason naming the entity kind, so the UI can offer
// re-selection. The repair is simply binding a fresh key (see
// [KeyManager.GetReferenceKey]) and resolving again.
func (m *KeyManager) Resolve(k RefKey) (Entity, health.Health) {
	entity, match := m.BindKeyToObject(k)
	switch {
	case match == MatchNone:
		return nil, health.Sicken(fmt.Sprintf("%s: %s reference no longer binds", ErrReferenceLost, k.kind))
	case match.IsFallback():
		return entity, health.Warn(fmt.Sprintf("%s reference auto-healed via %s match; confirm selection", k.kind, match))
	default:
		return entity, health.Healthy
	}
}
