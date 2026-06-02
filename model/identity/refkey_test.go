// SPDX-License-Identifier: GPL-2.0-only

package identity

import "testing"

func TestKeyEncodeDecodeRoundTrip(t *testing.T) {
	m := NewKeyManager()
	ctx := m.CreateKeyContext(&fakeSource{})
	key, _ := m.GetReferenceKey(ctx, fakeEntity{kind: KindEdge, lin: "edge#7/from-face#2"})

	back, err := DecodeKey(key.Encode())
	if err != nil {
		t.Fatalf("DecodeKey: %v", err)
	}
	if !back.Equal(key) {
		t.Errorf("decoded key %+v != original %+v", back, key)
	}
	if back.Kind() != KindEdge || back.Context() != ctx {
		t.Errorf("decoded kind/context = %v/%d, want edge/%d", back.Kind(), back.Context(), ctx)
	}
}

func TestKeyStringRoundTrip(t *testing.T) {
	m := NewKeyManager()
	ctx := m.CreateKeyContext(&fakeSource{})
	key, _ := m.GetReferenceKey(ctx, face("cap"))

	back, err := StringToKey(KeyToString(key))
	if err != nil {
		t.Fatalf("StringToKey: %v", err)
	}
	if !back.Equal(key) {
		t.Errorf("string round trip changed the key")
	}
}

func TestDecodeKeyRejectsBadInput(t *testing.T) {
	if _, err := DecodeKey([]byte{1, 2, 3}); err == nil {
		t.Error("DecodeKey accepted a too-short buffer")
	}
	// Header claims a 5-byte payload but supplies none.
	bad := RefKey{ctx: 1, kind: KindFace, payload: []byte("xxxxx")}.Encode()
	bad = bad[:len(bad)-3] // truncate the payload
	if _, err := DecodeKey(bad); err == nil {
		t.Error("DecodeKey accepted a length/payload mismatch")
	}
	if _, err := StringToKey("not valid base64!!"); err == nil {
		t.Error("StringToKey accepted invalid base64")
	}
}

func TestZeroAndEqual(t *testing.T) {
	if !(RefKey{}).IsZero() {
		t.Error("zero RefKey not reported as zero")
	}
	a := RefKey{ctx: 1, kind: KindFace, payload: []byte("L")}
	b := RefKey{ctx: 1, kind: KindFace, payload: []byte("L")}
	c := RefKey{ctx: 1, kind: KindEdge, payload: []byte("L")}
	if !a.Equal(b) {
		t.Error("identical keys not equal")
	}
	if a.Equal(c) {
		t.Error("keys of different kinds reported equal")
	}
	if a.IsZero() {
		t.Error("non-zero key reported as zero")
	}
}

func TestEnumStrings(t *testing.T) {
	kinds := map[EntityKind]string{
		KindFace: "face", KindEdge: "edge", KindVertex: "vertex", KindBody: "body",
		KindFeature: "feature", KindParameter: "parameter", EntityKind(99): "unknown",
	}
	for k, want := range kinds {
		if got := k.String(); got != want {
			t.Errorf("EntityKind(%d).String() = %q, want %q", k, got, want)
		}
	}
	if MatchExact.String() != "exact" || MatchNone.String() != "none" {
		t.Error("MatchType.String mismatch")
	}
}

func TestLoadContextRejectsTruncatedRecord(t *testing.T) {
	m := NewKeyManager()
	ctx := m.CreateKeyContext(&fakeSource{entities: []Entity{face("some-lineage")}})
	blob, err := m.SaveContextToArray(ctx)
	if err != nil {
		t.Fatalf("SaveContextToArray: %v", err)
	}
	// Lop off the tail so the record header promises more bytes than remain.
	for _, cut := range []int{len(blob) - 1, 13, 10} {
		if _, err := NewKeyManager().LoadContextToArray(blob[:cut]); err == nil {
			t.Errorf("LoadContextToArray accepted blob truncated to %d bytes", cut)
		}
	}
}
