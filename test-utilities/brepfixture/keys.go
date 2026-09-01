// SPDX-License-Identifier: GPL-2.0-only

package brepfixture

import (
	"testing"

	"oblikovati.org/kernel/topo"
)

// TopFaceKey returns the reference key of a body's +Z face. Face selection is by reference key
// everywhere in the kernel, so a test that shells, replaces or offsets "the top" needs the key,
// not the face — and needs it to survive the operation.
//
// Example: k := brepfixture.TopFaceKey(t, box) // the face a shell removes
// topFaceKey returns the reference key of the +Z (top) face.
func TopFaceKey(tb testing.TB, b *topo.Body) []byte {
	tb.Helper()
	for _, f := range b.Faces() {
		if f.Geometry().NormalAt(0, 0).Z > 0.99 {
			return f.ReferenceKey()
		}
	}
	tb.Fatal("no +Z face found")
	return nil
}
