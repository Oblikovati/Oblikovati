// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"bytes"
	"testing"
)

// --- fakes standing in for kernel/topo entities (which arrive in M07) ---

type fakeLineage []byte

func (l fakeLineage) LineageKey() []byte { return l }

type fakeEntity struct {
	kind EntityKind
	lin  string
}

func (e fakeEntity) EntityKind() EntityKind { return e.kind }
func (e fakeEntity) Lineage() Lineage       { return fakeLineage(e.lin) }

func face(lineage string) Entity { return fakeEntity{kind: KindFace, lin: lineage} }

type fakeSource struct{ entities []Entity }

func (s *fakeSource) Entities() []Entity { return s.entities }

// --- tests ---

func TestKeyRebindsAfterRecomputeRecreatesFace(t *testing.T) {
	m := NewKeyManager()
	src := &fakeSource{entities: []Entity{face("L1"), face("L2")}}
	ctx := m.CreateKeyContext(src)

	key, err := m.GetReferenceKey(ctx, src.entities[1]) // the L2 face
	if err != nil {
		t.Fatalf("GetReferenceKey: %v", err)
	}

	// Recompute: the B-rep is destroyed and rebuilt — brand-new entity objects,
	// but the surviving face carries the same lineage.
	src.entities = []Entity{face("L1"), face("L2")}
	got, match := m.BindKeyToObject(key)
	if match != MatchExact {
		t.Fatalf("match after recompute = %v, want exact", match)
	}
	if !bytes.Equal(got.Lineage().LineageKey(), []byte("L2")) {
		t.Errorf("rebound to lineage %q, want L2", got.Lineage().LineageKey())
	}
}

func TestCanBindFalseWhenTopologyVanishes(t *testing.T) {
	m := NewKeyManager()
	src := &fakeSource{entities: []Entity{face("keep"), face("doomed")}}
	ctx := m.CreateKeyContext(src)
	key, _ := m.GetReferenceKey(ctx, face("doomed"))

	if !m.CanBindKeyToObject(key) {
		t.Fatal("CanBind false while the face still exists")
	}
	// A feature edit removes the referenced face entirely.
	src.entities = []Entity{face("keep")}
	if m.CanBindKeyToObject(key) {
		t.Error("CanBind true after the topology genuinely vanished")
	}
	if got, match := m.BindKeyToObject(key); match != MatchNone || got != nil {
		t.Errorf("Bind of a lost key = (%v,%v), want (nil, none)", got, match)
	}
}

func TestKindMustMatch(t *testing.T) {
	m := NewKeyManager()
	src := &fakeSource{entities: []Entity{fakeEntity{kind: KindEdge, lin: "L"}}}
	ctx := m.CreateKeyContext(src)
	key, _ := m.GetReferenceKey(ctx, face("L")) // same lineage, but a FACE key
	if _, match := m.BindKeyToObject(key); match != MatchNone {
		t.Error("a face key bound to an edge with the same lineage")
	}
}

func TestKeysPersistAcrossSaveCloseReopen(t *testing.T) {
	m := NewKeyManager()
	src := &fakeSource{entities: []Entity{face("A"), face("B")}}
	ctx := m.CreateKeyContext(src)
	key, _ := m.GetReferenceKey(ctx, face("B"))

	blob, err := m.SaveContextToArray(ctx)
	if err != nil {
		t.Fatalf("SaveContextToArray: %v", err)
	}
	keyStr := KeyToString(key)

	// Reopen in a fresh manager: the context id is preserved so the key still
	// addresses it, and the key validates against the saved snapshot before any
	// recompute has happened.
	m2 := NewKeyManager()
	ctx2, err := m2.LoadContextToArray(blob)
	if err != nil {
		t.Fatalf("LoadContextToArray: %v", err)
	}
	if ctx2 != ctx {
		t.Errorf("reloaded context id = %d, want %d preserved", ctx2, ctx)
	}
	key2, err := StringToKey(keyStr)
	if err != nil {
		t.Fatalf("StringToKey: %v", err)
	}
	if !m2.CanBindKeyToObject(key2) {
		t.Error("reloaded key cannot validate against the saved snapshot")
	}

	// After recompute, the live source is re-pointed and the key binds to the
	// recreated face.
	rebuilt := &fakeSource{entities: []Entity{face("A"), face("B")}}
	if err := m2.RebindSource(ctx2, rebuilt); err != nil {
		t.Fatalf("RebindSource: %v", err)
	}
	if _, match := m2.BindKeyToObject(key2); match != MatchExact {
		t.Error("reloaded key did not bind after recompute")
	}
}

func TestReleaseContextStopsBinding(t *testing.T) {
	m := NewKeyManager()
	src := &fakeSource{entities: []Entity{face("X")}}
	ctx := m.CreateKeyContext(src)
	key, _ := m.GetReferenceKey(ctx, face("X"))
	m.ReleaseContext(ctx)
	if m.CanBindKeyToObject(key) {
		t.Error("key still binds after its context was released")
	}
}

func TestManagerErrorPaths(t *testing.T) {
	m := NewKeyManager()
	if _, err := m.GetReferenceKey(ContextID(99), face("X")); err == nil {
		t.Error("GetReferenceKey accepted an unknown context")
	}
	src := &fakeSource{entities: nil}
	ctx := m.CreateKeyContext(src)
	if _, err := m.GetReferenceKey(ctx, nil); err == nil {
		t.Error("GetReferenceKey accepted a nil entity")
	}
	if err := m.RebindSource(ContextID(99), src); err == nil {
		t.Error("RebindSource accepted an unknown context")
	}
	if _, err := m.SaveContextToArray(ContextID(99)); err == nil {
		t.Error("SaveContextToArray accepted an unknown context")
	}
	if _, err := m.LoadContextToArray([]byte{1, 2, 3}); err == nil {
		t.Error("LoadContextToArray accepted truncated data")
	}
	// Binding into an unknown context is a clean miss, not a crash.
	if _, match := m.BindKeyToObject(RefKey{ctx: 99, kind: KindFace}); match != MatchNone {
		t.Error("bind into unknown context did not return none")
	}
}
