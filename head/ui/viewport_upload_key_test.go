// SPDX-License-Identifier: GPL-2.0-only

package ui

import "testing"

// geomUploadKey turns the atlas key string into the uint64 the native renderer compares against its
// resident-geometry key to skip a re-upload (#1422). The contract the native side relies on: an empty
// key is the 0 "always upload" sentinel; a non-empty key is stable across calls (same key in, same
// uint64 out) and never 0 (so it can actually match and skip); distinct keys map to distinct values.
func TestGeomUploadKey(t *testing.T) {
	if got := geomUploadKey(""); got != 0 {
		t.Errorf("geomUploadKey(\"\") = %d, want 0 (the always-upload sentinel)", got)
	}
	const k = "part:42|sel:none|ov:1a2b"
	first, second := geomUploadKey(k), geomUploadKey(k)
	if first != second {
		t.Errorf("geomUploadKey not stable: %d != %d for the same key — a static orbit would re-upload every frame", first, second)
	}
	if first == 0 {
		t.Error("geomUploadKey(non-empty) = 0 — would alias the always-upload sentinel and never skip")
	}
	if other := geomUploadKey(k + "|changed"); other == first {
		t.Errorf("geomUploadKey collided distinct keys onto %d — a real geometry change would be skipped (stale)", other)
	}
}
