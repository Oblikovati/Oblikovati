// SPDX-License-Identifier: GPL-2.0-only

package attr

import (
	"testing"

	"oblikovati.org/model/identity"
)

// keyFor builds the reference key a selection would carry for an entity with the
// given lineage. It uses the public external-key constructor, so the test needs no
// KeyManager: two calls for the same lineage produce Equal keys — exactly the
// property the attribute manager relies on to re-anchor an attribute after recompute.
func keyFor(lineage string) identity.RefKey { return identity.ExternalKey([]byte(lineage)) }

func TestAttributeOnFaceSurvivesRecomputeAndReload(t *testing.T) {
	capKey := keyFor("cap")

	mgr := NewAttributeManager()
	mgr.AttributeSets(capKey).Set("acme").Put("finish", StringValue("anodized"))

	// Recompute: a fresh face object is created, but its lineage is unchanged, so a
	// key minted now is Equal and still finds the attribute.
	rebound := keyFor("cap")
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
	key := keyFor("vertex-7")

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
	mgr := NewAttributeManager()
	for _, lineage := range []string{"f1", "f2", "f3"} {
		mgr.AttributeSets(keyFor(lineage)).Set("tags").Put("reviewed", BoolValue(true))
	}
	// One extra object with a different set must not match.
	mgr.AttributeSets(keyFor("f4")).Set("other").Put("reviewed", BoolValue(true))

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
	mgr := NewAttributeManager()
	key := keyFor("f1")
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
	mgr := NewAttributeManager()
	mgr.AttributeSets(keyFor("f1")).Set("s").Put("a", StringValue("value"))
	blob := mgr.Encode()
	if _, err := DecodeAttributes(blob[:len(blob)-2]); err == nil {
		t.Error("DecodeAttributes accepted truncated data")
	}
}

// TestAnchorsAndExternalKeyRoundTrip checks the per-target enumeration and that an external-anchor
// (a wire reference key wrapped by identity.ExternalKey) survives serialize/reload and re-resolves.
func TestAnchorsAndExternalKeyRoundTrip(t *testing.T) {
	mgr := NewAttributeManager()
	mgr.AttributeSets(identity.DocumentKey()).Set("traceon").Put("default", FloatValue(0))
	mgr.AttributeSets(identity.ExternalKey([]byte("body-A"))).Set("traceon").Put("voltage", FloatValue(1000))
	mgr.AttributeSets(identity.ExternalKey([]byte("body-B"))).Set("traceon").Put("voltage", FloatValue(-500))

	if n := len(mgr.Anchors()); n != 3 {
		t.Fatalf("anchors = %d, want 3 (document + two bodies)", n)
	}

	back, err := DecodeAttributes(mgr.Encode())
	if err != nil {
		t.Fatalf("DecodeAttributes: %v", err)
	}
	ss, ok := back.Lookup(identity.ExternalKey([]byte("body-A")))
	if !ok {
		t.Fatal("external anchor lost across reload (key did not re-resolve)")
	}
	a, _ := ss.Set("traceon").Attribute("voltage")
	if a == nil {
		t.Fatal("external attribute missing after reload")
	}
	if v, _ := a.Value().Float(); v != 1000 {
		t.Errorf("body-A voltage = %g after reload, want 1000", v)
	}
}
