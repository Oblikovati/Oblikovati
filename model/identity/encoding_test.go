// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"encoding/binary"
	"testing"
)

// legacyEncode builds a pre-M31 (unversioned) key blob in the original layout —
// [ctx u64][kind u32][len u32][payload], no magic — so the migration path can be
// exercised against the exact bytes old documents persisted.
func legacyEncode(ctx ContextID, kind EntityKind, payload []byte) []byte {
	buf := make([]byte, 0, 16+len(payload))
	buf = binary.LittleEndian.AppendUint64(buf, uint64(ctx))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(kind))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(payload)))
	return append(buf, payload...)
}

func TestEncodeStampsCurrentScheme(t *testing.T) {
	m := NewKeyManager()
	ctx := m.CreateKeyContext(&fakeSource{})
	key, _ := m.GetReferenceKey(ctx, fakeEntity{kind: KindEdge, lin: "ext#1/edge#3"})

	if key.Scheme() != SchemeCurrent {
		t.Fatalf("minted key scheme = %d, want current %d", key.Scheme(), SchemeCurrent)
	}
	back, err := DecodeKey(key.Encode())
	if err != nil {
		t.Fatalf("DecodeKey: %v", err)
	}
	if back.Scheme() != SchemeCurrent {
		t.Errorf("round-tripped scheme = %d, want current %d", back.Scheme(), SchemeCurrent)
	}
	if !back.Equal(key) {
		t.Errorf("round trip changed identity: %+v vs %+v", back, key)
	}
}

func TestVersionedEnvelopeCarriesMagic(t *testing.T) {
	m := NewKeyManager()
	ctx := m.CreateKeyContext(&fakeSource{})
	key, _ := m.GetReferenceKey(ctx, face("cap"))

	if !hasMagic(key.Encode()) {
		t.Error("a freshly minted key is not written in the versioned envelope")
	}
}

// TestLegacyKeyDecodesAndMigrates is the core migration guarantee: a key persisted
// by a pre-M31 build still loads (scheme reported as legacy) and re-binds to the
// unchanged entity, rather than orphaning the reference.
func TestLegacyKeyDecodesAndMigrates(t *testing.T) {
	m := NewKeyManager()
	src := &fakeSource{entities: []Entity{face("base"), fakeEntity{kind: KindEdge, lin: "brep:edge#4"}}}
	ctx := m.CreateKeyContext(src)

	blob := legacyEncode(ctx, KindEdge, []byte("brep:edge#4"))
	if hasMagic(blob) {
		t.Fatal("legacy blob unexpectedly carries the versioned magic")
	}

	key, err := DecodeKey(blob)
	if err != nil {
		t.Fatalf("DecodeKey(legacy): %v", err)
	}
	if key.Scheme() != SchemeLegacy {
		t.Errorf("legacy key scheme = %d, want legacy %d", key.Scheme(), SchemeLegacy)
	}

	got, match := m.BindKeyToObject(key)
	if match != MatchExact {
		t.Fatalf("legacy key match = %v, want exact (unchanged topology)", match)
	}
	if string(got.Lineage().LineageKey()) != "brep:edge#4" {
		t.Errorf("legacy key bound to %q, want brep:edge#4", got.Lineage().LineageKey())
	}
}

// TestReEncodeUpgradesLegacyKey shows save-time migration: a legacy key, once
// decoded, re-encodes into the current versioned envelope.
func TestReEncodeUpgradesLegacyKey(t *testing.T) {
	blob := legacyEncode(7, KindFace, []byte("L"))
	key, err := DecodeKey(blob)
	if err != nil {
		t.Fatalf("DecodeKey: %v", err)
	}

	upgraded := key.Encode()
	if !hasMagic(upgraded) {
		t.Fatal("re-encoded legacy key is still unversioned")
	}
	back, err := DecodeKey(upgraded)
	if err != nil {
		t.Fatalf("DecodeKey(upgraded): %v", err)
	}
	if back.Scheme() != SchemeCurrent || !back.Equal(key) {
		t.Errorf("upgrade lost provenance/identity: scheme %d, equal %v", back.Scheme(), back.Equal(key))
	}
}

func TestDecodeRejectsUnknownScheme(t *testing.T) {
	m := NewKeyManager()
	ctx := m.CreateKeyContext(&fakeSource{})
	key, _ := m.GetReferenceKey(ctx, face("x"))

	blob := key.Encode()
	blob[len(keyMagic)] = SchemeCurrent + 1 // bump the scheme byte past what we know
	if _, err := DecodeKey(blob); err == nil {
		t.Error("DecodeKey accepted a key from an unknown future scheme")
	}
}

func TestDecodeRejectsTruncatedVersionedKey(t *testing.T) {
	m := NewKeyManager()
	ctx := m.CreateKeyContext(&fakeSource{})
	key, _ := m.GetReferenceKey(ctx, fakeEntity{kind: KindEdge, lin: "ext#1/edge#3"})

	blob := key.Encode()
	if _, err := DecodeKey(blob[:len(blob)-2]); err == nil {
		t.Error("DecodeKey accepted a versioned key truncated mid-payload")
	}
	if _, err := DecodeKey(keyMagic[:]); err == nil {
		t.Error("DecodeKey accepted bare magic with no body")
	}
}

func TestSchemeIsNotPartOfIdentity(t *testing.T) {
	// Same triple via the legacy and current envelopes must compare Equal — scheme
	// is provenance, not identity — so an attribute anchored under an old key is
	// still found after the key is re-minted.
	legacy, err := DecodeKey(legacyEncode(3, KindFace, []byte("face#1")))
	if err != nil {
		t.Fatalf("DecodeKey: %v", err)
	}
	current := RefKey{ctx: 3, kind: KindFace, payload: []byte("face#1"), scheme: SchemeCurrent}
	if !legacy.Equal(current) {
		t.Error("legacy and current keys for the same reference are not Equal")
	}
	if string(legacy.Encode()) != string(current.Encode()) {
		t.Error("Equal keys must encode identically regardless of decoded scheme")
	}
}
