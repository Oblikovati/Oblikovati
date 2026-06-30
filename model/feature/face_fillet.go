// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// FaceFilletDefinition rounds the edges shared between two face sets with a rolling-ball blend of
// the given radius — the adjacent-faces case of Inventor's face fillet (FilletConstantRadiusFaceSet,
// #694), where the two sets meet at one or more edges. Picking by face (rather than by edge) is the
// natural selection when a long chain of edges separates two regions. Non-adjacent face sets, where
// the two faces share no edge, are handled by healing the gap to the virtual edge (see
// nonAdjacentFaceFilletBody).
type FaceFilletDefinition struct {
	FaceKeysA [][]byte
	FaceKeysB [][]byte
	Radius    func() float64
}

// FaceFilletFeature is a face fillet between two face sets.
//
// Example: round where the top face meets a side face of a box —
//
//	NewDressUpFeatures(part.Features()).AddFaceFillet(topKeys, sideKeys, func() float64 { return 0.3 })
type FaceFilletFeature struct{ def *FaceFilletDefinition }

// Definition returns the feature's definition.
func (f *FaceFilletFeature) Definition() *FaceFilletDefinition { return f.def }

// Kind names the feature type.
func (f *FaceFilletFeature) Kind() string { return "face-fillet" }

// FilletType reports the public discriminator for this feature: a face fillet.
func (f *FaceFilletFeature) FilletType() types.FilletType { return types.FaceFillet }

// Recompute rounds the edges shared by the two face sets against the running body.
func (f *FaceFilletFeature) Recompute(in Input) (Output, error) {
	return faceFilletBody(in, f.def.FaceKeysA, f.def.FaceKeysB, callOrZero(f.def.Radius), "face-fillet")
}

// faceFilletBody rounds the edges between the two face sets with the constant-radius edge fillet.
// When the sets share edges it rounds them directly; when they share none (non-adjacent) it heals
// the gap to the virtual edge first (nonAdjacentFaceFilletBody).
func faceFilletBody(in Input, faceKeysA, faceKeysB [][]byte, radius float64, feat string) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	if radius <= 0 {
		return Output{}, fmt.Errorf("%s: radius %g must be > 0", feat, radius)
	}
	edgeKeys := sharedEdgeKeys(body, faceKeysA, faceKeysB)
	if len(edgeKeys) == 0 {
		return nonAdjacentFaceFilletBody(in, body, faceKeysA, faceKeysB, radius, feat)
	}
	return filletBody(in, edgeKeys, radius, types.FilletCornerMiter, types.FilletConcaveOutward, blendProfile{}, feat, nil)
}

// nonAdjacentFaceFilletBody rounds two face sets that share no edge (#694): it deletes the faces in
// the gap between them so the two sets HEAL together at the virtual edge their planes define, then
// rounds that recreated edge with the constant-radius edge fillet. The heal recreates the true edge
// (with its real convexity), so the existing fillet — and its tessellation — just works; there is no
// case-dependent boolean to choose. Only faces whose planes intersect (a real virtual edge) are
// supported; parallel faces have no edge to heal to (the slot/full-round case) and error.
func nonAdjacentFaceFilletBody(in Input, body *topo.Body, faceKeysA, faceKeysB [][]byte, radius float64, feat string) (Output, error) {
	gap := gapFaceKeys(body, faceKeysA, faceKeysB)
	if len(gap) == 0 {
		return Output{}, fmt.Errorf("%s: the two face sets are not separated by a fillable gap "+
			"(parallel or disjoint faces are not supported)", feat)
	}
	healed, err := ops.DeleteFaces(planarized(body, feat), gap)
	if err != nil {
		return Output{}, fmt.Errorf("%s: healing the gap between the faces: %w", feat, err)
	}
	edgeKeys := healedEdgeKeys(healed, body, faceKeysA, faceKeysB)
	if len(edgeKeys) == 0 {
		return Output{}, fmt.Errorf("%s: the faces did not heal to a shared edge to round", feat)
	}
	picks := make([]ops.EdgeFilletRadii, len(edgeKeys))
	for i, k := range edgeKeys {
		picks[i] = ops.EdgeFilletRadii{Key: k, R0: radius, R1: radius}
	}
	result, err := ops.FilletEdgesCorner(healed, picks, ops.CornerMiter, ops.FillConcaveOutward)
	if err != nil {
		return Output{}, fmt.Errorf("%s: %w", feat, err)
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}

// gapFaceKeys returns the reference keys of the faces that bridge the gap between the two sets: a
// face neither in A nor B that borders SOME face in A and SOME face in B, and whose normal faces
// partly the same way as both (so it sits in the corner between them). The normal test excludes
// shared neighbours that merely touch both sets (e.g. an end cap perpendicular to the blend axis).
func gapFaceKeys(body *topo.Body, faceKeysA, faceKeysB [][]byte) [][]byte {
	inA, inB := keyLookup(faceKeysA), keyLookup(faceKeysB)
	nsA, nsB := planeNormals(body, faceKeysA), planeNormals(body, faceKeysB)
	var out [][]byte
	for _, f := range body.Faces() {
		k := string(f.ReferenceKey())
		if inA[k] || inB[k] {
			continue
		}
		n, ok := planarFaceNormal(f)
		if !ok || !bordersSet(f, inA) || !bordersSet(f, inB) {
			continue
		}
		if facesToward(n, nsA) && facesToward(n, nsB) {
			out = append(out, f.ReferenceKey())
		}
	}
	return out
}

// healedEdgeKeys returns the edges of the healed body whose two faces lie on the planes of an A face
// and a B face — the virtual edge(s) the heal recreated between the two sets.
func healedEdgeKeys(healed, orig *topo.Body, faceKeysA, faceKeysB [][]byte) [][]byte {
	plA, plB := facePlanes(orig, faceKeysA), facePlanes(orig, faceKeysB)
	var out [][]byte
	for _, e := range healed.Edges() {
		fs := e.Faces()
		if len(fs) != 2 {
			continue
		}
		p0, ok0 := facePlane(fs[0])
		p1, ok1 := facePlane(fs[1])
		if !ok0 || !ok1 {
			continue
		}
		if (onAnyPlane(p0, plA) && onAnyPlane(p1, plB)) || (onAnyPlane(p0, plB) && onAnyPlane(p1, plA)) {
			out = append(out, e.ReferenceKey())
		}
	}
	return out
}

// bordersSet reports whether face f shares an edge with any face in the set.
func bordersSet(f *topo.Face, set map[string]bool) bool {
	for _, e := range f.Edges() {
		for _, nb := range e.Faces() {
			if nb != f && set[string(nb.ReferenceKey())] {
				return true
			}
		}
	}
	return false
}

// facesToward reports whether n has a positive dot with at least one of the set normals.
func facesToward(n math.Vector3, normals []math.Vector3) bool {
	for _, m := range normals {
		if float64(n.Dot(m)) > 1e-9 {
			return true
		}
	}
	return false
}

// planarFaceNormal / facePlane read a face's plane (normal), reporting false for a non-planar face.
func planarFaceNormal(f *topo.Face) (math.Vector3, bool) {
	pl, ok := f.Geometry().(geom.Plane)
	if !ok {
		return math.Vector3{}, false
	}
	return pl.Normal(), true
}

func facePlane(f *topo.Face) (geom.Plane, bool) {
	pl, ok := f.Geometry().(geom.Plane)
	return pl, ok
}

// planeNormals / facePlanes collect the planes (or just normals) of the named faces on body.
func planeNormals(body *topo.Body, keys [][]byte) []math.Vector3 {
	var out []math.Vector3
	for _, p := range facePlanes(body, keys) {
		out = append(out, p.Normal())
	}
	return out
}

func facePlanes(body *topo.Body, keys [][]byte) []geom.Plane {
	var out []geom.Plane
	for _, k := range keys {
		if f, ok := FindOrRecoverFace(body, k); ok {
			if pl, ok := facePlane(f); ok {
				out = append(out, pl)
			}
		}
	}
	return out
}

// onAnyPlane reports whether plane p coincides with any plane in the set (same normal and offset).
func onAnyPlane(p geom.Plane, set []geom.Plane) bool {
	for _, q := range set {
		if float64(p.Normal().Dot(q.Normal())) > 0.999 &&
			abs(float64(p.Origin.VectorTo(q.Origin).Dot(q.Normal()))) < 1e-6 {
			return true
		}
	}
	return false
}

// abs is the float64 absolute value (local to avoid a math import alias clash).
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// sharedEdgeKeys returns the reference key of every edge of body whose two adjacent faces lie one in
// face set A and one in set B — the edges a face fillet between the two sets rounds.
func sharedEdgeKeys(body *topo.Body, faceKeysA, faceKeysB [][]byte) [][]byte {
	inA, inB := keyLookup(faceKeysA), keyLookup(faceKeysB)
	var out [][]byte
	for _, e := range body.Edges() {
		fs := e.Faces()
		if len(fs) != 2 {
			continue
		}
		ka, kb := string(fs[0].ReferenceKey()), string(fs[1].ReferenceKey())
		if (inA[ka] && inB[kb]) || (inA[kb] && inB[ka]) {
			out = append(out, e.ReferenceKey())
		}
	}
	return out
}

// keyLookup indexes reference keys for membership tests (string(key) is a stable map key for bytes).
func keyLookup(keys [][]byte) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[string(k)] = true
	}
	return m
}
