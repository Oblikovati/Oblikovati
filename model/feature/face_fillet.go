// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/topo"
)

// FaceFilletDefinition rounds the edges shared between two face sets with a rolling-ball blend of
// the given radius — the adjacent-faces case of Inventor's face fillet (FilletConstantRadiusFaceSet,
// #694), where the two sets meet at one or more edges. Picking by face (rather than by edge) is the
// natural selection when a long chain of edges separates two regions. Non-adjacent face sets, where
// the ball bridges a gap between faces that share no edge, are a follow-up.
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

// faceFilletBody resolves the edges shared between the two face sets on the running body and rounds
// them with the constant-radius edge-fillet kernel. No shared edge is an error (the feature goes
// Sick): non-adjacent face sets, which need a gap-bridging rolling-ball path, are not yet supported.
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
		return Output{}, fmt.Errorf("%s: the two face sets share no edge "+
			"(non-adjacent face fillet is not yet supported)", feat)
	}
	return filletBody(in, edgeKeys, radius, types.FilletCornerMiter, types.FilletConcaveOutward, feat)
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
