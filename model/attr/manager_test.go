// SPDX-License-Identifier: GPL-2.0-only

package attr

import (
	"testing"

	"oblikovati/model/identity"
)

// keyFor mints a reference key for a face with the given lineage, the way a
// selection would before attaching attributes to it.
func keyFor(t *testing.T, m *identity.KeyManager, ctx identity.ContextID, lineage string) identity.RefKey {
	t.Helper()
	key, err := m.GetReferenceKey(ctx, fakeFace(lineage))
	if err != nil {
		t.Fatalf("GetReferenceKey: %v", err)
	}
	return key
}

type fakeFaceEntity string

func (fakeFaceEntity) EntityKind() identity.EntityKind { return identity.KindFace }
func (e fakeFaceEntity) Lineage() identity.Lineage     { return fakeLineage(e) }

type fakeLineage string

func (l fakeLineage) LineageKey() []byte { return []byte(l) }

func fakeFace(lineage string) identity.Entity { return fakeFaceEntity(lineage) }

type fakeSource struct{ faces []string }

func (s *fakeSource) Entities() []identity.Entity {
	out := make([]identity.Entity, len(s.faces))
	for i, l := range s.faces {
		out[i] = fakeFace(l)
	}
	return out
}

func TestAttributeOnFaceSurvivesRecomputeAndReload(t *testing.T) {
	km := identity.NewKeyManager()
	src := &fakeSource{faces: []string{"cap", "side"}}
	ctx := km.CreateKeyContext(src)
	capKey := keyFor(t, km, ctx, "cap")

	mgr := NewAttributeManager()
	mgr.AttributeSets(capKey).Set("acme").Put("finish", StringValue("anodized"))

	// Recompute: a fresh face object is created, but its lineage is unchanged, so
	// the key minted now is equal and still finds the attribute.
	src.faces = []string{"cap", "side"}
	rebound := keyFor(t, km, ctx, "cap")
	ss, ok := mgr.Lookup(rebound)
	if !ok {
		t.Fatal("attribute lost across recompute (key did not re-anchor)")
	}
	if a, ok := ss.Set("acme").Attribute("finish"); !ok {
		t.Error("attribute missing after recompute")
	} else if s, _ := a.Value().Str(); s != "anodized" {
		t.Errorf("attribute value = %q after recompute, want anodized", s)
	}

	// Reload: the manager is serialized and read back; the same key still finds it.
	reloaded, err := DecodeAttributes(mgr.Encode())
	if err != nil {
		t.Fatalf("DecodeAttributes: %v", err)
	}
	ss2, ok := reloaded.Lookup(capKey)
	if !ok {
		t.Fatal("attribute lost across reload")
	}
	if a, _ := ss2.Set("acme").Attribute("finish"); a == nil {
		t.Error("attribute missing after reload")
	}
}

func TestAddinPrivateDataRoundTrips(t *testing.T) {
	km := identity.NewKeyManager()
	ctx := km.CreateKeyContext(&fakeSource{})
	key := keyFor(t, km, ctx, "vertex-7")

	mgr := NewAttributeManager()
	set := mgr.AttributeSets(key).Set("com.acme.addin")
	set.Put("count", IntValue(7))
	set.Put("blob", BytesValue([]byte{0xde, 0xad, 0xbe, 0xef}))
	set.Put("enabled", BoolValue(true))

	back, err := DecodeAttributes(mgr.Encode())
	if err != nil {
		t.Fatalf("DecodeAttributes: %v", err)
	}
	ss, _ := back.Lookup(key)
	got, _ := ss.Set("com.acme.addin").Attribute("blob")
	raw, _ := got.Value().Raw()
	if len(raw) != 4 || raw[3] != 0xef {
		t.Errorf("blob round trip = %v, want the 4 bytes", raw)
	}
	if c, _ := ss.Set("com.acme.addin").Attribute("count"); c.ValueType() != Integer {
		t.Error("integer attribute lost its type across round trip")
	}
}

func TestFindAttributesQuery(t *testing.T) {
	km := identity.NewKeyManager()
	ctx := km.CreateKeyContext(&fakeSource{})
	mgr := NewAttributeManager()
	for _, lineage := range []string{"f1", "f2", "f3"} {
		key := keyFor(t, km, ctx, lineage)
		mgr.AttributeSets(key).Set("tags").Put("reviewed", BoolValue(true))
	}
	// One extra object with a different set must not match.
	other := keyFor(t, km, ctx, "f4")
	mgr.AttributeSets(other).Set("other").Put("reviewed", BoolValue(true))

	hits := mgr.FindAttributes("tags", "reviewed")
	if len(hits) != 3 {
		t.Fatalf("FindAttributes hits = %d, want 3", len(hits))
	}
	// Empty attr name matches any attribute in the set.
	if len(mgr.FindAttributes("tags", "")) != 3 {
		t.Error("empty attribute filter did not match all attributes in the set")
	}
	if len(mgr.FindAttributes("nonesuch", "reviewed")) != 0 {
		t.Error("query matched a nonexistent set")
	}
}

func TestManagerRemoveAndCount(t *testing.T) {
	km := identity.NewKeyManager()
	ctx := km.CreateKeyContext(&fakeSource{})
	mgr := NewAttributeManager()
	key := keyFor(t, km, ctx, "f1")
	mgr.AttributeSets(key).Set("s").Put("a", IntValue(1))
	if mgr.Count() != 1 {
		t.Fatalf("Count = %d, want 1", mgr.Count())
	}
	if !mgr.Remove(key) {
		t.Error("Remove returned false for an existing anchor")
	}
	if mgr.Count() != 0 || mgr.Remove(key) {
		t.Error("anchor not removed")
	}
}

func TestDecodeAttributesRejectsTruncated(t *testing.T) {
	km := identity.NewKeyManager()
	ctx := km.CreateKeyContext(&fakeSource{})
	mgr := NewAttributeManager()
	mgr.AttributeSets(keyFor(t, km, ctx, "f1")).Set("s").Put("a", StringValue("value"))
	blob := mgr.Encode()
	if _, err := DecodeAttributes(blob[:len(blob)-2]); err == nil {
		t.Error("DecodeAttributes accepted truncated data")
	}
}
