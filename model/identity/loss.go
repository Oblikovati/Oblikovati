// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"errors"
	"fmt"

	"oblikovati/model/health"
)

// ErrReferenceLost is the standard error behind a lost-reference health state. A
// consumer that depended on a now-unbindable key reports this — binding loss is a
// normal consequence of editing (a feature change can delete the referenced face),
// so the contract is to go sick and surface for re-selection, never to crash
// (parametric-cad §7, architecture core/05).
var ErrReferenceLost = errors.New("reference lost")

// Resolve binds a key and translates the outcome into modeling health — the single
// place the reference-loss policy lives, applied identically by every consumer
// (features, dimensions, mates). A found entity yields healthy state; a lost
// reference yields [health.Sick] with a reason naming the entity kind, so the UI
// can offer re-selection. The repair is simply binding a fresh key (see
// [KeyManager.GetReferenceKey]) and resolving again.
func (m *KeyManager) Resolve(k RefKey) (Entity, health.Health) {
	entity, match := m.BindKeyToObject(k)
	if match == MatchNone {
		return nil, health.Sicken(fmt.Sprintf("%s: %s reference no longer binds", ErrReferenceLost, k.kind))
	}
	return entity, health.Healthy
}
