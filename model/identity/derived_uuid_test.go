// SPDX-License-Identifier: GPL-2.0-only

package identity

import (
	"regexp"
	"testing"
)

const (
	docA = "11111111-2222-3333-4444-555555555555"
	docB = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
)

var canonicalUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// TestSketchEntityKeyDeterministic: the same inputs always derive the same UUID — the
// property the whole persistent-reference scheme rests on.
func TestSketchEntityKeyDeterministic(t *testing.T) {
	k1, err := SketchEntityKey(docA, 42)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	k2, _ := SketchEntityKey(docA, 42)
	if k1 != k2 {
		t.Errorf("non-deterministic: %q != %q", k1, k2)
	}
	if !canonicalUUID.MatchString(k1) {
		t.Errorf("not a canonical v5 UUID: %q", k1)
	}
}

// TestKeysUniqueAcrossDocuments is the user's core requirement: the same local id in two
// different documents must derive two different UUIDs (no cross-document collision ever).
func TestKeysUniqueAcrossDocuments(t *testing.T) {
	a, _ := SketchEntityKey(docA, 7)
	b, _ := SketchEntityKey(docB, 7)
	if a == b {
		t.Errorf("same local id in different documents collided: %q", a)
	}
}

// TestKeysUniqueAcrossLocalIDs: distinct local ids in one document derive distinct UUIDs.
func TestKeysUniqueAcrossLocalIDs(t *testing.T) {
	a, _ := SketchEntityKey(docA, 1)
	b, _ := SketchEntityKey(docA, 2)
	if a == b {
		t.Errorf("distinct local ids collided: %q", a)
	}
}

// TestSketchAndEntityKindsDoNotCollide: a sketch and an entity with the same local id and
// document derive different UUIDs, because the kind tag namespaces the derived name.
func TestSketchAndEntityKindsDoNotCollide(t *testing.T) {
	s, _ := SketchKey(docA, 5)
	e, _ := SketchEntityKey(docA, 5)
	if s == e {
		t.Errorf("sketch and entity keys collided for the same id: %q", s)
	}
}

// TestDeriveRejectsMalformedGUID: a non-UUID document guid is a clear error, not a silent
// bad key.
func TestDeriveRejectsMalformedGUID(t *testing.T) {
	for _, bad := range []string{"", "not-a-uuid", "11111111-2222-3333-4444"} {
		if _, err := SketchEntityKey(bad, 1); err == nil {
			t.Errorf("SketchEntityKey(%q) should fail", bad)
		}
	}
}

// TestDeriveAcceptsUnhyphenatedGUID: a 32-hex-digit guid without hyphens parses too.
func TestDeriveAcceptsUnhyphenatedGUID(t *testing.T) {
	hyphenated, _ := SketchEntityKey(docA, 9)
	plain, err := SketchEntityKey("11111111222233334444555555555555", 9)
	if err != nil {
		t.Fatalf("unhyphenated guid: %v", err)
	}
	if hyphenated != plain {
		t.Errorf("hyphenation changed the derived key: %q != %q", hyphenated, plain)
	}
}
