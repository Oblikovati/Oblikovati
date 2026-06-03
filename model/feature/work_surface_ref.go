// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/math"
)

// Surface-tangent work planes are built on a B-rep face the user picked, not on
// another work feature. A face is named by its persistent topological reference key
// (kernel topo lineage key — the same key dress-up features hold), re-bound to the
// running body each recompute via FindFaceByKey, so the datum follows the face through
// upstream edits (parametric-cad §7). To keep work features uniformly addressed by a
// single [WorkRef], a face key is folded into a WorkRef with a "face/" prefix and a
// URL-safe base64 of the binary key (RawURLEncoding has no '/' or '+', so it never
// collides with the WorkRef path separator).

const faceRefPrefix = "face/"

// faceRef encodes a B-rep face reference key as a WorkRef.
//
//	wp := planes.AddByTorusMidPlane(faceRef(face.ReferenceKey()))
func faceRef(key []byte) WorkRef {
	return WorkRef(faceRefPrefix + base64.RawURLEncoding.EncodeToString(key))
}

// parseFaceRef decodes a face WorkRef back to its binary reference key. The bool is
// false when ref is not a face reference (so callers can distinguish "not a face" from
// a corrupt key, which is returned as an error).
func parseFaceRef(ref WorkRef) ([]byte, bool, error) {
	s := string(ref)
	if !strings.HasPrefix(s, faceRefPrefix) {
		return nil, false, nil
	}
	key, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, faceRefPrefix))
	if err != nil {
		return nil, true, fmt.Errorf("work geometry: corrupt face reference %q: %w", ref, err)
	}
	return key, true, nil
}

// surface resolves a face WorkRef to its surface geometry by re-binding the reference
// key against the running body. It errors if ref is not a face reference, if no body
// has been built yet, or if the key no longer binds (the face was removed upstream) —
// the last case is what makes a tangent work plane go Sick and surface for re-selection.
func (g *WorkGeometry) surface(ref WorkRef) (geom.Surface, error) {
	key, isFace, err := parseFaceRef(ref)
	if err != nil {
		return nil, err
	}
	if !isFace {
		return nil, fmt.Errorf("work geometry: %q is not a face reference", ref)
	}
	if len(g.bodies) == 0 {
		return nil, fmt.Errorf("work geometry: no body yet to resolve face %q", ref)
	}
	for _, b := range g.bodies {
		if f, ok := b.FindFaceByKey(key); ok {
			return f.Geometry(), nil
		}
	}
	return nil, fmt.Errorf("work geometry: face reference %q is lost", ref)
}

const vertexRefPrefix = "vertex/"

// VertexRef encodes a B-rep vertex's reference key as a point WorkRef, so a picked solid
// vertex can be a point input to a work feature (e.g. a three-point plane through three
// model vertices). Resolved against the running body, like a face reference.
func VertexRef(key []byte) WorkRef {
	return WorkRef(vertexRefPrefix + base64.RawURLEncoding.EncodeToString(key))
}

// vertexPoint resolves a vertex WorkRef to its position against the running body. The
// bool is false when ref is not a vertex reference (so [WorkGeometry.point] can fall
// through to its other reference kinds); a vertex reference that no longer binds is an
// error (the work feature then goes Sick).
func (g *WorkGeometry) vertexPoint(ref WorkRef) (math.Point3, bool, error) {
	s := string(ref)
	if !strings.HasPrefix(s, vertexRefPrefix) {
		return math.Point3{}, false, nil
	}
	key, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, vertexRefPrefix))
	if err != nil {
		return math.Point3{}, true, fmt.Errorf("work geometry: corrupt vertex reference %q: %w", ref, err)
	}
	for _, b := range g.bodies {
		if v, ok := b.FindVertexByKey(key); ok {
			return v.Point(), true, nil
		}
	}
	return math.Point3{}, true, fmt.Errorf("work geometry: vertex reference %q is lost", ref)
}
