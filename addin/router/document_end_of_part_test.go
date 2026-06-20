// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestEndOfPartRoundTrip reads the active part's marker (at the end by default), rolls it back to an
// earlier feature index, and restores it — checking each reply reflects the marker state (#141).
func TestEndOfPartRoundTrip(t *testing.T) {
	r, s := seededSession(t)

	var got wire.EndOfPartResult
	call(t, r, s, wire.MethodDocumentGetEndOfPart, `{}`, &got)
	if got.Position != -1 || got.RolledBack {
		t.Fatalf("default marker = %+v, want position -1 / not rolled back", got)
	}

	var back wire.EndOfPartResult
	call(t, r, s, wire.MethodDocumentSetEndOfPart, mustJSON(t, wire.SetEndOfPartArgs{Position: 0}), &back)
	if back.Position != 0 || !back.RolledBack {
		t.Errorf("after roll-back to 0 = %+v, want position 0 / rolled back", back)
	}

	var end wire.EndOfPartResult
	call(t, r, s, wire.MethodDocumentSetEndOfPart, mustJSON(t, wire.SetEndOfPartArgs{Position: -1}), &end)
	if end.Position != -1 || end.RolledBack {
		t.Errorf("after restore-to-end = %+v, want position -1 / not rolled back", end)
	}
}
