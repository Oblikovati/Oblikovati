// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"strings"
	"testing"

	"github.com/Oblikovati/oblikovati/model/health"
)

// fakeConsumer stands in for a feature/dimension that depends on a referenced
// entity through a key. It mirrors the contract every real consumer follows: on
// recompute it resolves its key and adopts the resulting health — it never panics.
type fakeConsumer struct {
	inputKey RefKey
	health   health.Health
}

func (c *fakeConsumer) recompute(m *KeyManager) {
	_, c.health = m.Resolve(c.inputKey)
}

func TestLostReferenceGoesSickAndIsFixable(t *testing.T) {
	m := NewKeyManager()
	src := &fakeSource{entities: []Entity{face("hole-edge"), face("base")}}
	ctx := m.CreateKeyContext(src)

	key, _ := m.GetReferenceKey(ctx, face("hole-edge"))
	consumer := &fakeConsumer{inputKey: key}

	consumer.recompute(m)
	if !consumer.health.OK() {
		t.Fatalf("consumer sick while its reference exists: %+v", consumer.health)
	}

	// A later edit deletes the referenced entity — the consumer goes sick, not fatal.
	src.entities = []Entity{face("base")}
	consumer.recompute(m)
	if consumer.health.Status != health.Sick {
		t.Fatalf("consumer health = %v, want sick after reference loss", consumer.health.Status)
	}
	if !strings.Contains(consumer.health.Reason, ErrReferenceLost.Error()) {
		t.Errorf("sick reason = %q, want it to mention reference loss", consumer.health.Reason)
	}

	// Re-selection repairs it: the user points the consumer at a surviving entity.
	newKey, _ := m.GetReferenceKey(ctx, face("base"))
	consumer.inputKey = newKey
	consumer.recompute(m)
	if !consumer.health.OK() {
		t.Errorf("consumer not healthy after re-selection: %+v", consumer.health)
	}
}

func TestResolveReturnsEntityWhenHealthy(t *testing.T) {
	m := NewKeyManager()
	src := &fakeSource{entities: []Entity{face("L")}}
	ctx := m.CreateKeyContext(src)
	key, _ := m.GetReferenceKey(ctx, face("L"))

	entity, h := m.Resolve(key)
	if !h.OK() || entity == nil {
		t.Fatalf("Resolve of a valid key = (%v, %+v), want entity + healthy", entity, h)
	}
}
