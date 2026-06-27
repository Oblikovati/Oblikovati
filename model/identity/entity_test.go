// SPDX-License-Identifier: GPL-2.0-only

package identity

import "testing"

// TestExternalKey checks the external-anchor bridge: it round-trips through the key codec, equal
// external bytes produce an equal key (so a re-minted reference key re-finds its attributes),
// distinct bytes produce distinct keys, and ExternalRef recovers the wrapped bytes.
func TestExternalKey(t *testing.T) {
	ref := []byte{4, 0x11, 0x22, 0x33} // a kernel/topo reference key (kind byte + lineage)
	k := ExternalKey(ref)

	if k.Kind() != KindExternal {
		t.Errorf("kind = %v, want external", k.Kind())
	}
	got, ok := k.ExternalRef()
	if !ok || string(got) != string(ref) {
		t.Errorf("ExternalRef = (%v, %v), want (%v, true)", got, ok, ref)
	}
	// Round-trips through Encode/DecodeKey (persistence) preserving the external ref.
	dec, err := DecodeKey(k.Encode())
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !dec.Equal(k) {
		t.Errorf("decoded key != original")
	}
	if r2, ok := dec.ExternalRef(); !ok || string(r2) != string(ref) {
		t.Errorf("decoded ExternalRef = (%v, %v), want %v", r2, ok, ref)
	}
	// Equal bytes → equal encoded key (recompute survival); distinct bytes → distinct.
	if string(ExternalKey(ref).Encode()) != string(k.Encode()) {
		t.Error("equal external bytes did not produce an equal key")
	}
	if string(ExternalKey([]byte{4, 0x11, 0x22, 0x99}).Encode()) == string(k.Encode()) {
		t.Error("distinct external bytes produced the same key")
	}
	// A document key is not an external anchor.
	if _, ok := DocumentKey().ExternalRef(); ok {
		t.Error("DocumentKey reported as external")
	}
}
