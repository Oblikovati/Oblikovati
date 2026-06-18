// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"encoding/base64"
	"fmt"
	"strings"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// facePlane resolves a planar B-rep face WorkRef to a sketch plane, so a picked planar face
// can serve anywhere a plane reference is accepted (offset-from-face, midplane, …). The bool
// is false when ref is not a face reference (the caller falls through to its other kinds); a
// face that is non-planar or no longer binds is an error.
func (g *WorkGeometry) facePlane(ref WorkRef) (sketch.Plane, bool, error) {
	_, isFace, err := parseFaceRef(ref)
	if !isFace || err != nil {
		return sketch.Plane{}, isFace, err
	}
	surf, err := g.surface(ref)
	if err != nil {
		return sketch.Plane{}, true, err
	}
	pl, ok := surf.(geom.Plane)
	if !ok {
		return sketch.Plane{}, true, fmt.Errorf("work geometry: face %q is not planar", ref)
	}
	// Origin the plane at the part origin projected onto it (Inventor's sketch-on-face
	// convention), not the surface's parametric origin — which for an extruded/tessellated cap
	// sits at a rim vertex, so a sketch on the plane would be positioned 2D-(0,0) at the rim
	// (geometry placed there lands off the body). The projection keeps the in-plane axes.
	sp, err := sketch.NewPlane(projectOntoPlane(math.P3(0, 0, 0), pl), pl.UAxis, pl.VAxis)
	return sp, true, err
}

// projectOntoPlane returns the foot of the perpendicular from p to the plane pl (p moved
// along the plane normal until it lies on the plane).
func projectOntoPlane(p math.Point3, pl geom.Plane) math.Point3 {
	n := pl.Normal()
	dist := p.VectorTo(pl.Origin).Dot(n)
	return p.TranslateBy(n.Scale(dist))
}

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

// FaceRefKey decodes a face WorkRef back to its reference key, reporting ok=false for a ref that is
// not a face reference (#157 selection mutation resolves picked-by-reference faces this way).
func FaceRefKey(ref WorkRef) ([]byte, bool) { return decodeRefKey(string(ref), faceRefPrefix) }

// VertexRefKey decodes a vertex WorkRef back to its reference key, reporting ok=false otherwise.
func VertexRefKey(ref WorkRef) ([]byte, bool) { return decodeRefKey(string(ref), vertexRefPrefix) }

// decodeRefKey strips prefix and base64-decodes the key, reporting ok=false on a prefix mismatch
// or a malformed payload.
func decodeRefKey(s, prefix string) ([]byte, bool) {
	if !strings.HasPrefix(s, prefix) {
		return nil, false
	}
	key, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, prefix))
	if err != nil {
		return nil, false
	}
	return key, true
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
