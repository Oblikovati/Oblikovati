// SPDX-License-Identifier: GPL-2.0-only

package event

import (
	"context"
	"reflect"
	"sync"
)

// Bus is a typed publish/subscribe hub. It is safe for concurrent use; handlers
// run synchronously on the emitting goroutine, after the lock is released, so a
// handler may subscribe or unsubscribe without deadlocking. A handler that needs
// to mutate the model should enqueue a command rather than mutate mid-emit
// (frame-safe delivery, core/00).
type Bus struct {
	mu   sync.RWMutex
	subs map[subKey][]handlerEntry
	seq  uint64
}

// subKey identifies a handler slot by event Go type and phase.
type subKey struct {
	typ   reflect.Type
	phase Phase
}

type handlerEntry struct {
	id uint64
	fn any // a Handler[E] for the key's event type
}

// NewBus returns an empty bus.
func NewBus() *Bus {
	return &Bus{subs: map[subKey][]handlerEntry{}}
}

// Subscription identifies one registered handler so it can be cancelled.
type Subscription struct {
	bus *Bus
	key subKey
	id  uint64
}

// Subscribe registers h for events of type E in phase p and returns a handle to
// cancel it. Subscribing the same type in both phases takes two calls.
func Subscribe[E Event](b *Bus, p Phase, h Handler[E]) Subscription {
	k := subKey{typ: typeOf[E](), phase: p}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	id := b.seq
	b.subs[k] = append(b.subs[k], handlerEntry{id: id, fn: h})
	return Subscription{bus: b, key: k, id: id}
}

// Cancel removes the subscription, reporting whether it was found.
func (s Subscription) Cancel() bool {
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()
	entries := s.bus.subs[s.key]
	for i, e := range entries {
		if e.id == s.id {
			s.bus.subs[s.key] = append(entries[:i], entries[i+1:]...)
			return true
		}
	}
	return false
}

// Emit delivers e to every Before-or-After handler for its type and returns the
// aggregate outcome. It uses a background context; see [EmitContext] for deadlines.
func Emit[E Event](b *Bus, p Phase, e E) Outcome {
	return EmitContext(context.Background(), b, p, e)
}

// EmitContext delivers e with an explicit context (e.g. a veto deadline for
// add-ins). Every handler runs; the aggregate outcome is the strongest disposition
// any handler returned (Abort > Handled > NotHandled), carrying the first veto
// reason. A vetoing Before outcome tells the caller to cancel the operation.
func EmitContext[E Event](ctx context.Context, b *Bus, p Phase, e E) Outcome {
	handlers := b.snapshot(subKey{typ: typeOf[E](), phase: p})
	result := Continue()
	for _, entry := range handlers {
		out := entry.fn.(Handler[E])(Context{Ctx: ctx, Phase: p}, e)
		result = stronger(result, out)
	}
	return result
}

// snapshot copies the handler slice for a key under the read lock, so handlers can
// safely mutate subscriptions while being invoked.
func (b *Bus) snapshot(k subKey) []handlerEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()
	src := b.subs[k]
	out := make([]handlerEntry, len(src))
	copy(out, src)
	return out
}

// stronger keeps the higher-priority disposition, preserving the first veto reason.
func stronger(acc, next Outcome) Outcome {
	if next.Code > acc.Code {
		return next
	}
	return acc
}

// typeOf returns the reflect.Type of the event type E for use as a dispatch key.
func typeOf[E Event]() reflect.Type {
	return reflect.TypeFor[E]()
}
