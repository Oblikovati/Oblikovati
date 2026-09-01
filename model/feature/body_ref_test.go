// SPDX-License-Identifier: GPL-2.0-only

package feature

import "testing"

// TestBodyRefRoundTrips: a body reference key folds into a body/<url-base64> WorkRef and decodes
// back to the exact bytes, so a picked whole body round-trips through model.selection/model.select
// (#1492). The encoding uses RawURLEncoding, so a key containing the WorkRef path separator or
// padding-sensitive bytes survives intact.
func TestBodyRefRoundTrips(t *testing.T) {
	t.Parallel()
	key := []byte{0x00, 0x2f, 0x2b, 0xff, 'b', 'o', 'd', 'y'} // includes '/' (0x2f) and '+' (0x2b)
	ref := BodyRef(key)
	if got := string(ref); got[:len("body/")] != "body/" {
		t.Fatalf("BodyRef = %q, want a body/ prefix", got)
	}
	back, ok := BodyRefKey(ref)
	if !ok {
		t.Fatalf("BodyRefKey(%q) reported not-a-body-ref", ref)
	}
	if string(back) != string(key) {
		t.Errorf("round-trip key = %x, want %x", back, key)
	}
}

// TestBodyRefKeyRejectsOtherForms: BodyRefKey must not claim a face/vertex ref or a corrupt
// payload, so ResolveRefOnBodies dispatches each form to the right finder and never binds a
// face key as a body.
func TestBodyRefKeyRejectsOtherForms(t *testing.T) {
	t.Parallel()
	if _, ok := BodyRefKey(FaceRef([]byte("f"))); ok {
		t.Error("BodyRefKey accepted a face ref")
	}
	if _, ok := BodyRefKey(VertexRef([]byte("v"))); ok {
		t.Error("BodyRefKey accepted a vertex ref")
	}
	if _, ok := BodyRefKey(WorkRef("body/not*base64")); ok {
		t.Error("BodyRefKey accepted a corrupt base64 payload")
	}
}
