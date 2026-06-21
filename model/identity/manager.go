// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"bytes"
	"fmt"
)

// KeyManager mints, binds and persists reference keys — the ReferenceKeyManager
// (architecture core/05). It owns a set of key contexts; each context is a
// versioned snapshot of one body/document's topology. Binding searches a context's
// current entities for one whose kind and lineage match the key — topological
// naming, not pointer identity. A bind that finds nothing is a legitimate,
// non-fatal outcome ([MatchNone]); see the loss policy in loss.go.
type KeyManager struct {
	nextCtx  uint64
	contexts map[ContextID]*keyContext
}

// NewKeyManager returns an empty manager.
func NewKeyManager() *KeyManager {
	return &KeyManager{contexts: map[ContextID]*keyContext{}}
}

// CreateKeyContext registers a context backed by source (the live topology) and
// returns its id. Context ids start at 1 so the zero id means "no context".
func (m *KeyManager) CreateKeyContext(source EntitySource) ContextID {
	m.nextCtx++
	id := ContextID(m.nextCtx)
	m.contexts[id] = &keyContext{id: id, source: source}
	return id
}

// ReleaseContext forgets a context. Keys into it will no longer bind.
func (m *KeyManager) ReleaseContext(id ContextID) {
	delete(m.contexts, id)
}

// RebindSource re-points a context at a fresh topology source, e.g. after a
// document is reopened and recomputed. It errors for an unknown context.
func (m *KeyManager) RebindSource(id ContextID, source EntitySource) error {
	ctx, ok := m.contexts[id]
	if !ok {
		return fmt.Errorf(errUnknownContext, id)
	}
	ctx.source = source
	return nil
}

// GetReferenceKey mints a key for entity e within context id. It captures the
// entity's lineage, so the key rebinds to the recreated entity after a recompute.
func (m *KeyManager) GetReferenceKey(id ContextID, e Entity) (RefKey, error) {
	if _, ok := m.contexts[id]; !ok {
		return RefKey{}, fmt.Errorf(errUnknownContext, id)
	}
	if e == nil || e.Lineage() == nil {
		return RefKey{}, fmt.Errorf("identity: cannot key a nil entity or entity without lineage")
	}
	payload := append([]byte(nil), e.Lineage().LineageKey()...)
	return RefKey{
		ctx:     id,
		kind:    e.EntityKind(),
		payload: payload,
		parent:  parentHint(e.Lineage()),
		anchor:  anchorOf(e),
		scheme:  SchemeCurrent,
	}, nil
}

// parentHint captures the lineage's parent key for ancestral re-binding, when the
// lineage exposes one (M31-F06). A root lineage yields an absent hint.
func parentHint(l Lineage) ancestryHint {
	al, ok := l.(AncestralLineage)
	if !ok {
		return ancestryHint{}
	}
	pk := al.ParentKey()
	if len(pk) == 0 {
		return ancestryHint{}
	}
	return ancestryHint{key: append([]byte(nil), pk...), ok: true}
}

// anchorOf captures the entity's representative point for geometric tie-breaking,
// when the entity exposes one (M31-F06).
func anchorOf(e Entity) anchorHint {
	ae, ok := e.(AnchoredEntity)
	if !ok {
		return anchorHint{}
	}
	p, ok := ae.Anchor()
	if !ok {
		return anchorHint{}
	}
	return anchorHint{point: p, ok: true}
}

// BindKeyToObject resolves a key to a live entity in its context, returning the
// match type. It tries tiers in descending quality (M31-F06, #1156): an exact
// lineage match, then — only on an exact miss — an ancestral sibling, then a
// geometric tie-break among ambiguous siblings. MatchNone (with a nil entity)
// means the referenced topology is truly gone.
func (m *KeyManager) BindKeyToObject(k RefKey) (Entity, MatchType) {
	ctx, ok := m.contexts[k.ctx]
	if !ok || ctx.source == nil {
		return nil, MatchNone
	}
	ents := ctx.source.Entities()
	if e := exactMatch(k, ents); e != nil {
		return e, MatchExact
	}
	return fallbackMatch(k, ents)
}

// exactMatch returns the lone entity whose kind and full lineage equal the key, or
// nil. This is the only tier that yields healthy state.
func exactMatch(k RefKey, ents []Entity) Entity {
	for _, e := range ents {
		if e.EntityKind() == k.kind && bytes.Equal(e.Lineage().LineageKey(), k.payload) {
			return e
		}
	}
	return nil
}

// CanBindKeyToObject reports whether the key would bind, without returning the
// entity. It checks the live source if present, otherwise the saved snapshot —
// so a key reloaded from disk can be validated before the B-rep is recomputed.
func (m *KeyManager) CanBindKeyToObject(k RefKey) bool {
	ctx, ok := m.contexts[k.ctx]
	if !ok {
		return false
	}
	if ctx.source != nil {
		_, match := m.BindKeyToObject(k)
		return match != MatchNone
	}
	for _, rec := range ctx.snapshot {
		if rec.kind == k.kind && bytes.Equal(rec.lineage, k.payload) {
			return true
		}
	}
	return false
}

// SaveContextToArray captures the context's current topology into a snapshot and
// serializes it, so keys survive save/close/reopen. It errors for an unknown
// context.
func (m *KeyManager) SaveContextToArray(id ContextID) ([]byte, error) {
	ctx, ok := m.contexts[id]
	if !ok {
		return nil, fmt.Errorf(errUnknownContext, id)
	}
	ctx.captureSnapshot()
	return ctx.encode(), nil
}

// LoadContextToArray restores a context from bytes produced by SaveContextToArray,
// preserving its id so previously-minted keys still address it. The restored
// context has only its snapshot; call RebindSource once the document recomputes to
// bind keys to live entities.
func (m *KeyManager) LoadContextToArray(data []byte) (ContextID, error) {
	ctx, err := decodeContext(data)
	if err != nil {
		return 0, err
	}
	m.contexts[ctx.id] = ctx
	if uint64(ctx.id) > m.nextCtx {
		m.nextCtx = uint64(ctx.id)
	}
	return ctx.id, nil
}
