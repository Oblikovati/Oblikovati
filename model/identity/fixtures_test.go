// SPDX-License-Identifier: GPL-2.0-only

package identity

// Shared test fixtures for the identity package. They stand in for the kernel/topo
// entities (Face/Edge/Vertex) that implement [Entity]/[Lineage] in the real model.

// fakeLineage is a bare lineage key with no parent (a root) — enough to exercise
// RefKey encoding and exact identity.
type fakeLineage []byte

func (l fakeLineage) LineageKey() []byte { return l }

// fakeEntity is a kind + lineage string, the minimal [Entity].
type fakeEntity struct {
	kind EntityKind
	lin  string
}

func (e fakeEntity) EntityKind() EntityKind { return e.kind }
func (e fakeEntity) Lineage() Lineage       { return fakeLineage(e.lin) }

// face builds a face entity with the given lineage.
func face(lineage string) Entity { return fakeEntity{kind: KindFace, lin: lineage} }

// keyFor builds the reference key for an entity directly — the manager-free stand-in
// for what minting produced: the identity triple, the current scheme, plus the
// optional ancestral/geometric fallback hints when the entity exposes them. It lets
// the encoding, refkey and recovery tests construct keys without a KeyManager.
func keyFor(ctx ContextID, e Entity) RefKey {
	k := RefKey{ctx: ctx, kind: e.EntityKind(), payload: e.Lineage().LineageKey(), scheme: SchemeCurrent}
	if al, ok := e.Lineage().(AncestralLineage); ok {
		if pk := al.ParentKey(); len(pk) > 0 {
			k.parent = ancestryHint{key: pk, ok: true}
		}
	}
	if ae, ok := e.(AnchoredEntity); ok {
		if p, ok := ae.Anchor(); ok {
			k.anchor = anchorHint{point: p, ok: true}
		}
	}
	return k
}
