// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CapHoles eliminates through-holes / blind pockets that open to the surface (M20-F15-adjacent,
// #721) — the complement of FillInternalVoids, which drops disconnected void shells. Given the
// wall faces of a hole (its cylindrical/prism tube, plus a pocket's bottom), it removes them and
// heals each opening flush by dropping the inner loop it left on the bordering face, whose own
// plane caps the opening. The result keeps all real outer geometry but closes the holes.
//
// Each opening must be an inner loop of a surviving face whose edges are all shared with removed
// faces (i.e. the opening lies in that face — a hole drilled normal to a planar face). It errors
// when the result is not a closed solid, so a caller can fall back rather than ship an open body.
//
// Example: CapHoles(plateWithBore, boreWallKeys) → the solid plate (bore filled).
func CapHoles(body *topo.Body, wallFaceKeys [][]byte) (*topo.Body, error) {
	removed, err := resolveFaceSet(body, wallFaceKeys)
	if err != nil {
		return nil, err
	}
	if len(removed) == 0 {
		return nil, fmt.Errorf("cap-holes: no wall faces selected")
	}
	result := buildSolidFromLoops(cappedLoops(body, removed))
	if r := Validate(result); !r.Valid || !result.IsSolid() {
		return nil, fmt.Errorf("cap-holes: result is not a closed solid (openings not coplanar with their faces?) %v", r.Issues)
	}
	return result, nil
}

// cappedLoops returns every surviving face as a point-ring loop set, dropping the inner loops
// that bounded a removed hole (those heal flush against the face's own plane).
func cappedLoops(body *topo.Body, removed map[uint64]bool) []ploop {
	noMoves := map[uint64]math.Point3{}
	var out []ploop
	for _, f := range body.Faces() {
		if removed[f.ID()] {
			continue
		}
		pl := ploop{normal: f.Geometry().NormalAt(0, 0)}
		for _, l := range f.Loops() {
			if !l.IsOuter() && loopBordersRemoved(l, removed) {
				continue // an opening into the capped hole ⇒ heal flush (drop the inner loop)
			}
			pl.rings = append(pl.rings, healedRing(l, noMoves))
		}
		out = append(out, pl)
	}
	return out
}

// loopBordersRemoved reports whether every edge of a loop is shared with a removed face — i.e.
// the loop is an opening left by removing the hole walls.
func loopBordersRemoved(l *topo.Loop, removed map[uint64]bool) bool {
	for _, u := range l.EdgeUses() {
		if !edgeBordersRemoved(u.Edge(), removed) {
			return false
		}
	}
	return true
}

func edgeBordersRemoved(e *topo.Edge, removed map[uint64]bool) bool {
	for _, f := range e.Faces() {
		if removed[f.ID()] {
			return true
		}
	}
	return false
}

// CapHolesByDiameter detects straight through-holes / pockets whose openings are no wider than
// maxDiameter and caps them (#721). An opening is an inner loop; the hole's walls are the faces
// adjacent to it. Returns the body unchanged when nothing qualifies.
func CapHolesByDiameter(body *topo.Body, maxDiameter float64) (*topo.Body, error) {
	walls := holeWallFaces(body, maxDiameter)
	if len(walls) == 0 {
		return body, nil
	}
	return CapHoles(body, walls)
}

// holeWallFaces returns the reference keys of faces that wall a hole whose opening (an inner
// loop) spans at most maxDiameter — the faces adjacent across the opening's edges.
func holeWallFaces(body *topo.Body, maxDiameter float64) [][]byte {
	wallIDs := map[uint64]bool{}
	var walls [][]byte
	for _, f := range body.Faces() {
		for _, l := range f.Loops() {
			if l.IsOuter() || loopDiameter(l) > maxDiameter {
				continue
			}
			for _, nb := range loopNeighbourFaces(l, f) {
				if !wallIDs[nb.ID()] {
					wallIDs[nb.ID()] = true
					walls = append(walls, nb.ReferenceKey())
				}
			}
		}
	}
	return walls
}

// loopNeighbourFaces returns the faces sharing an edge with loop l other than its owner.
func loopNeighbourFaces(l *topo.Loop, owner *topo.Face) []*topo.Face {
	var out []*topo.Face
	seen := map[uint64]bool{owner.ID(): true}
	for _, u := range l.EdgeUses() {
		for _, nb := range u.Edge().Faces() {
			if !seen[nb.ID()] {
				seen[nb.ID()] = true
				out = append(out, nb)
			}
		}
	}
	return out
}

// loopDiameter estimates an opening's width as the largest extent of its vertices' bounding box.
func loopDiameter(l *topo.Loop) float64 {
	lo := math.P3(stdmath.Inf(1), stdmath.Inf(1), stdmath.Inf(1))
	hi := math.P3(stdmath.Inf(-1), stdmath.Inf(-1), stdmath.Inf(-1))
	for _, u := range l.EdgeUses() {
		p := u.Edge().StartVertex().Point()
		lo = math.P3(stdmath.Min(lo.X, p.X), stdmath.Min(lo.Y, p.Y), stdmath.Min(lo.Z, p.Z))
		hi = math.P3(stdmath.Max(hi.X, p.X), stdmath.Max(hi.Y, p.Y), stdmath.Max(hi.Z, p.Z))
	}
	return stdmath.Max(hi.X-lo.X, stdmath.Max(hi.Y-lo.Y, hi.Z-lo.Z))
}
