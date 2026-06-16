// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/math"

// overlayTolerance is the chord tolerance the client-graphics body-overlay resolver tessellates
// at — fine enough for a crisp highlight without the renderer's adaptive cost.
const overlayTolerance = 0.01

// resolveOverlayMesh is the client-graphics body-mesh resolver (M16-F05 #641): given a body's
// persistent reference key, it finds that body among the visible bodies and returns its
// tessellated world-space triangle mesh, so a surface overlay primitive renders the real body
// in an add-in's override color without shipping a mesh over the wire. ok is false when no
// visible body carries the key.
func (s *Session) resolveOverlayMesh(bodyKey string, _ uint64) (pos []math.Point3, norm []math.Vector3, idx []int, ok bool) {
	if bodyKey == "" {
		return nil, nil, nil, false
	}
	for _, b := range s.VisibleBodies() {
		if string(b.ReferenceKey()) == bodyKey {
			fs := s.FacetStore().CalculateFacets(b, overlayTolerance)
			return fs.Mesh.Positions, fs.Mesh.Normals, fs.Mesh.Indices, true
		}
	}
	return nil, nil, nil, false
}
