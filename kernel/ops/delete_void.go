// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"bytes"
	"fmt"

	"oblikovati.org/kernel/topo"
)

// The FaceShell arm of Delete Face (#1884): selecting a face that lies on an internal void
// (cavity) shell deletes that whole void and restores the enclosed mass, mirroring Inventor's
// DeleteFaceFeatures.Add(FaceShell). This is distinct from deleting outer-shell faces (which
// heals or opens the body — see delete_face.go / drop_faces.go); the feature dispatches on
// whether the selection sits on a void shell.

// FacesOnVoidShell reports whether every selected face lies on an internal void shell — the
// signal the Delete Face feature uses to route to void removal rather than a face delete.
// An empty selection, a lost key, or any face on the outer shell reports false.
func FacesOnVoidShell(b *topo.Body, faceKeys [][]byte, q Quality) bool {
	if len(faceKeys) == 0 {
		return false
	}
	for _, k := range faceKeys {
		sh := shellOfFaceKey(b, k)
		if sh == nil || !ShellIsVoid(sh, q) {
			return false
		}
	}
	return true
}

// RemoveVoidShellByFaces drops every internal void shell that a selected face belongs to,
// restoring the enclosed mass. It errors when a selected face reference is lost or names a face
// that is not on an internal void shell, so the feature goes Sick rather than mangle the body.
func RemoveVoidShellByFaces(b *topo.Body, faceKeys [][]byte, q Quality) (*topo.Body, error) {
	drop := map[*topo.Shell]bool{}
	for _, k := range faceKeys {
		sh := shellOfFaceKey(b, k)
		if sh == nil {
			return nil, fmt.Errorf("delete-face void: face reference %q lost", k)
		}
		if !ShellIsVoid(sh, q) {
			return nil, fmt.Errorf("delete-face void: face %q is not on an internal void shell", k)
		}
		drop[sh] = true
	}
	kept := make([]*topo.Shell, 0, len(b.Shells()))
	for _, sh := range b.Shells() {
		if !drop[sh] {
			kept = append(kept, sh)
		}
	}
	return topo.BodyFromShells(b.Lineage(), b.IsSolid(), kept...), nil
}

// shellOfFaceKey returns the shell carrying the face whose reference key equals key, or nil.
func shellOfFaceKey(b *topo.Body, key []byte) *topo.Shell {
	for _, sh := range b.Shells() {
		for _, f := range sh.Faces() {
			if bytes.Equal(f.ReferenceKey(), key) {
				return sh
			}
		}
	}
	return nil
}
