// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// facePiece is a far face's solved runout arc together with the far edges that bound it at the apex
// (0 = the A/B flank sentinel). The bounding edge ids let transformLoop orient the piece to the
// loop's actual traversal sense (a loop entering from the exit edge emits tOut→tIn).
type facePiece struct {
	piece cornerPiece
	entry uint64 // far edge on the A side (fanFace.entryEdge; 0 = A flank)
	exit  uint64 // far edge on the B side (fanFace.exitEdge; 0 = B flank)
}

// buildSpreadMaps solves each fan and returns two views the rebuild consumes: spreads[face][apexID]
// gives the far face its arc piece (transformLoop's new arm), and caps[apexID] gives the ordered
// A→B pieces the cylinder end is subdivided into (cylinderFace) so the cylinder cap and the far
// pieces weld edge-for-edge at the shared split points — the whole reason the runout closes into a
// solid rather than 4 open edges. A solver error, or an apex with no coincident body vertex, skips
// the fan (validateRunoutFans is the hard pre-pass reject that stops the op before this point on a
// solver error; the apex-miss skip is defense-in-depth against a mis-keyed fan); a failed fan never
// emits a partial spread.
func buildSpreadMaps(fans []endCornerFan, body *topo.Body) (map[*topo.Face]map[uint64]facePiece, map[uint64][]cornerPiece) {
	spreads := map[*topo.Face]map[uint64]facePiece{}
	caps := map[uint64][]cornerPiece{}
	faceByID := indexFaces(body)
	for _, fan := range fans {
		sp, err := solveRunoutSpread(fan)
		if err != nil {
			continue
		}
		vid, ok := vertexIDForApex(body, fan.apex)
		if !ok {
			continue // apex has no coincident body vertex — skip rather than mis-key under id 0
		}
		addFanSpread(spreads, faceByID, fan, sp, vid)
		caps[vid] = orderedPieces(fan, sp)
	}
	return spreads, caps
}

// indexFaces maps every body face by its id, so a fan's opaque far-face id resolves to the live face.
func indexFaces(body *topo.Body) map[uint64]*topo.Face {
	out := make(map[uint64]*topo.Face, len(body.Faces()))
	for _, f := range body.Faces() {
		out[f.ID()] = f
	}
	return out
}

// vertexIDForApex returns the id of the body vertex coincident with the fan apex (the fan is
// topology-free — it carries the apex point, not the vertex — so the rebuild re-locates it here).
// ok=false when no vertex matches: silently keying the fan under id 0 would mis-key it onto whatever
// spread the caller keeps at 0, so the caller must skip the fan on a miss instead.
func vertexIDForApex(body *topo.Body, apex math.Point3) (uint64, bool) {
	for _, v := range body.Vertices() {
		if v.Point().DistanceTo(apex) < 1e-9 {
			return v.ID(), true
		}
	}
	return 0, false
}

// addFanSpread records each far face's arc piece under the apex vertex id, tagged with the far edges
// bounding it so transformLoop can orient the arc to the loop's traversal sense.
func addFanSpread(spreads map[*topo.Face]map[uint64]facePiece, faceByID map[uint64]*topo.Face, fan endCornerFan, sp runoutSpread, vid uint64) {
	for _, ff := range fan.fan {
		pc, ok := sp.pieces[ff.face]
		if !ok {
			continue
		}
		f := faceByID[ff.face]
		if spreads[f] == nil {
			spreads[f] = map[uint64]facePiece{}
		}
		spreads[f][vid] = facePiece{piece: pc, entry: ff.entryEdge, exit: ff.exitEdge}
	}
}

// orderedPieces returns the fan's arc pieces in A→B order — ta→split0→…→tb — so concatenating them
// tiles the cylinder end exactly (the weld-twice invariant guarantees consecutive pieces share a
// split point).
func orderedPieces(fan endCornerFan, sp runoutSpread) []cornerPiece {
	out := make([]cornerPiece, 0, len(fan.fan))
	for _, ff := range fan.fan {
		if pc, ok := sp.pieces[ff.face]; ok {
			out = append(out, pc)
		}
	}
	return out
}

// pruneEndCorners drops the trihedral end-corner entries whose vertex is owned by a fan, so a
// valence>3 vertex is rounded by the spread arm alone (endFaceAt had wrongly picked ONE far face to
// carry a trihedral arc, dropping the rest — the runout face-drop bug).
func pruneEndCorners(ends map[*topo.Face]map[uint64]corner, fanV map[uint64]bool) {
	for _, m := range ends {
		for vid := range m {
			if fanV[vid] {
				delete(m, vid)
			}
		}
	}
}

// addRunoutApex emits one far face's runout arc in place of the apex vertex, oriented to the loop's
// traversal sense (the edges arriving at prevEdge and leaving at nextEdge disambiguate the direction).
func addRunoutApex(fl *filletLoop, fp facePiece, prevEdge, nextEdge uint64) {
	from, to := orientFacePiece(fp, prevEdge, nextEdge)
	addSpreadPiece(fl, fp.piece, from, to)
}

// orientFacePiece returns the piece's endpoints in the loop's traversal order. The piece runs
// tIn (entry/A side) → tOut (exit/B side); a loop that arrives via the exit edge (or, at the B
// flank, leaves via the entry edge) walks it reversed.
func orientFacePiece(fp facePiece, prevEdge, nextEdge uint64) (from, to math.Point3) {
	if loopEntersFromEntry(fp, prevEdge, nextEdge) {
		return fp.piece.tIn, fp.piece.tOut
	}
	return fp.piece.tOut, fp.piece.tIn
}

// loopEntersFromEntry reports whether the loop reaches the apex on the piece's entry (A) side and
// leaves on its exit (B) side. It matches whichever bounding edge is a real far edge (the flank
// sentinels are 0): if the exit edge is real, forward means it leaves the apex; else the entry edge
// is real and forward means it arrives at the apex.
func loopEntersFromEntry(fp facePiece, prevEdge, nextEdge uint64) bool {
	if fp.exit != 0 {
		return nextEdge == fp.exit
	}
	return prevEdge == fp.entry
}

// addSpreadPiece emits one far face's runout arc from→to (re-derived through the piece's midpoint so
// a reversed traversal keeps the same curve), replacing the apex vertex — mirrors addCornerRound's
// two-point shape (arc into `from`, straight out of `to`).
func addSpreadPiece(fl *filletLoop, pc cornerPiece, from, to math.Point3) {
	fl.add(from, spreadArc(pc, from, to))
	fl.add(to, nil)
}

// spreadArc rebuilds the piece's arc oriented from→to, or nil when the piece degenerated to a
// straight segment (a collinear section, Task 5). The midpoint round-trips through PointAt(0.5),
// matching arcMidpoints' convention.
func spreadArc(pc cornerPiece, from, to math.Point3) geom.Curve3 {
	if pc.curve == nil {
		return nil
	}
	arc, err := geom.Arc3dByThreePoints(from, pc.curve.PointAt(0.5), to)
	if err != nil {
		return nil
	}
	return arc
}

// hasFacePiece reports whether v is a fan apex owned by this face's spread.
func hasFacePiece(spread map[uint64]facePiece, v *topo.Vertex) bool {
	_, ok := spread[v.ID()]
	return ok
}

// capEndSegs turns a fan's ordered A→B pieces into the cylinder end's boundary segments, so the
// cylinder cap is split at the same points (ta→split0→…→tb) the far faces weld to.
func capEndSegs(pieces []cornerPiece) []endSeg {
	out := make([]endSeg, 0, len(pieces))
	for _, pc := range pieces {
		if pc.curve == nil {
			out = append(out, endSeg{from: pc.tIn, to: pc.tOut})
			continue
		}
		out = append(out, endSeg{from: pc.tIn, to: pc.tOut, curve: pc.curve, mid: pc.curve.PointAt(0.5), arc: true})
	}
	return out
}
