// SPDX-License-Identifier: GPL-2.0-only

package ipt

// The DC feature graph — DEPENDENCY RESOLUTION (M48 #2228 split of featuregraph.go). The profile side
// of the graph: an extrude names a BoundaryPatch (property 1), and a FaceBound node points back at
// that patch while naming the Sketch2D it bounds, so the profile binding is patch -> faceBound ->
// sketch. These exported queries walk the decoded nodes to resolve every extrude's depth and sketch,
// and the part's revolve profile, through those references — replacing the "extrude i consumes sketch
// i" guess that is provably wrong on real parts (BigChunkyPlate's 13th extrude uses its 15th sketch).

import (
	"encoding/binary"
)

// ExtrudeDepths returns the depth of every extrude feature, in feature order, each read from that
// feature's own referenced distance parameter.
func ExtrudeDepths(d *Document) []float64 {
	nodes := dcNodes(d)
	var out []float64
	for _, n := range nodes {
		if n.typ != featureNodeType {
			continue
		}
		if v, ok := extrudeDepth(nodes, n.payload); ok {
			out = append(out, v)
		}
	}
	return out
}

// The profile side of the feature graph: an extrude names a BoundaryPatch (property 1), and a
// FaceBound node points BACK at that patch while naming the Sketch2D it bounds. So the profile
// binding is patch -> faceBound -> sketch, which is what replaces "extrude i consumes sketch i" —
// a guess that is provably wrong on real parts (BigChunkyPlate's 13th extrude uses its 15th
// sketch). Grounded in InventorLoader: Read_22947391 "BoundaryPatch", Read_424EB7D7 "FaceBound",
// Read_F9884C43 "FaceBoundOuter", ReadHeaderFaceBound.
const (
	boundaryPatchNodeType  = 0x22947391
	faceBoundNodeType      = 0x424EB7D7
	faceBoundOuterNodeType = 0xF9884C43
)

// propProfile is the extrude's BoundaryPatch property (InventorLoader Create_FxExtrude_New's
// `patch = getProperty(properties, 0x01)`).
const propProfile = 1

// faceBoundHeaderEnd is where a FaceBound's three reference lists begin; its `sketch` and `proxy`
// refs follow them and a u32 pair.
const faceBoundHeaderEnd = 30

// ExtrudeProfiles returns, for each extrude in feature order, the index of the sketch it actually
// consumes (indices into GraphSketches). An extrude whose profile can't be resolved yields -1, so
// callers skip it rather than build on a guessed sketch.
func ExtrudeProfiles(d *Document) []int {
	nodes := dcNodes(d)
	_, sketchIndex := sketchOrdinals(nodes)
	patchToSketch := boundPatchSketches(nodes, sketchIndex)
	var out []int
	for _, n := range nodes {
		if n.typ != featureNodeType {
			continue
		}
		if _, ok := extrudeDepth(nodes, n.payload); !ok {
			continue // not an extrude; keep this list aligned with DecodeExtrudes
		}
		out = append(out, profileOf(n.payload, patchToSketch))
	}
	return out
}

// profileOf maps one extrude to its sketch index via its BoundaryPatch property.
func profileOf(pay []byte, patchToSketch map[int]int) int {
	props, ok := featureProperties(pay)
	if !ok || len(props) <= propProfile {
		return -1
	}
	if i, ok := patchToSketch[props[propProfile]]; ok {
		return i
	}
	return -1
}

// revolveAxisNodeType is the node a Revolution feature names at property 2 (propDirection), in
// place of an extrude's DirectionAxis: its own axis/centreline reference. It is what tells a revolve
// feature apart from an extrude (which names 0xCE52DF40 there) — verified on the ReelToReel revolve
// parts, where every revolve's property 2 resolves to this type and no extrude's does.
const revolveAxisNodeType = 0x8EF06C89

// RevolveProfileSketch returns the GraphSketches index of the sketch the part's Revolution feature
// consumes as its profile, resolved through the feature's BoundaryPatch property — the SAME
// patch -> FaceBound -> Sketch2D chain ExtrudeProfiles uses. This replaces GUESSING the revolve base
// (the first sketch that merely looks revolvable), which mis-picks a CUT profile on a multi-feature
// machined part and revolves garbage. A revolve feature is a generic Fx feature that is not an
// extrude (no depth) and names its axis at property 2 as a revolveAxisNodeType node. ok=false when
// the part has no such feature or its profile patch can't be mapped, so callers keep their fallback.
func RevolveProfileSketch(d *Document) (int, bool) {
	nodes := dcNodes(d)
	_, sketchIndex := sketchOrdinals(nodes)
	patchToSketch := boundPatchSketches(nodes, sketchIndex)
	for _, n := range nodes {
		if n.typ != featureNodeType {
			continue
		}
		if _, isExtrude := extrudeDepth(nodes, n.payload); isExtrude {
			continue
		}
		props, ok := featureProperties(n.payload)
		if !ok || len(props) <= propProfile || !isRevolveFeature(nodes, props) {
			continue
		}
		if i, ok := patchToSketch[props[propProfile]]; ok {
			return i, true
		}
	}
	return 0, false
}

// isRevolveFeature reports whether a feature's properties are a revolve's — property 2 (propDirection)
// names a revolveAxisNodeType node, where an extrude names a DirectionAxis and a hole names its kind
// at property 0.
func isRevolveFeature(nodes []dcNode, props []int) bool {
	if len(props) <= propDirection {
		return false
	}
	ax, ok := nodeAt(nodes, props[propDirection])
	return ok && ax.typ == revolveAxisNodeType
}

// boundPatchSketches maps each BoundaryPatch ordinal to the index of the sketch that bounds it,
// by walking the FaceBound nodes that name the patch as their proxy.
func boundPatchSketches(nodes []dcNode, sketchIndex map[int]int) map[int]int {
	out := map[int]int{}
	for _, n := range nodes {
		if n.typ != faceBoundNodeType && n.typ != faceBoundOuterNodeType {
			continue
		}
		sketchRef, patchRef, ok := faceBoundRefs(n.payload)
		if !ok {
			continue
		}
		if i, ok := sketchIndex[sketchRef]; ok {
			out[patchRef] = i
		}
	}
	return out
}

// faceBoundRefs reads a FaceBound's `sketch` and `proxy` references, which follow its three
// variable-length reference lists and a u32 pair.
func faceBoundRefs(pay []byte) (sketchRef, patchRef int, ok bool) {
	i := faceBoundHeaderEnd
	for k := 0; k < 3; k++ {
		_, next, ok := refList2(pay, i)
		if !ok {
			return 0, 0, false
		}
		i = next
	}
	i += 8 // the u32 pair before the refs
	if i+8 > len(pay) {
		return 0, 0, false
	}
	sketchRef = int(binary.LittleEndian.Uint32(pay[i:]) & refIndexMask)
	patchRef = int(binary.LittleEndian.Uint32(pay[i+4:]) & refIndexMask)
	return sketchRef, patchRef, true
}
