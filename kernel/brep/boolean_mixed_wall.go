// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Ruled-wall splits for the mixed per-face boolean (ADR-0058). A full-band cylinder OR cone wall crossed
// by the other operand's planar faces splits through the ruled (u,v) chart the curved boolean already
// uses (cylinderSideSolidSplit): the imprint is the plane∩cylinder intersection clipped EXACTLY to the
// tool face's polygonal trim, and the same clipped curves are MIRRORED onto the tool faces so both
// sides split on identical coordinates. v1 scope: RULING-line contacts (a tool face parallel to the
// axis — the full-height slot class); a circle/ellipse contact needs a conic imprint on a polygonal
// tool face (the planeFaceUV imprint extension) and declines until that lands. A wall with no
// interaction keeps the whole-face pass-through classification.

// wallImprints computes, for every full-band cylinder wall of p, the imprint curves against the other
// operand's planar faces, mirroring the ruling segments onto the other side's polygonal imprint lists
// (otherImp, index-aligned with other.planar). ok=false declines: a conic (circle/ellipse) contact, an
// unhandled surface pair, or a wall overlapping a face kind the imprint cannot cover.
func wallImprints(p, other *facePartition, otherImp [][][2]math.Point3) ([][]geom.Curve3, bool) {
	out := make([][]geom.Curve3, len(p.wall))
	for i, wf := range p.wall {
		box := inflateBox(p.wallBox[i])
		if wallOverlapsUncovered(wf, box, other) {
			return nil, false
		}
		for j := range other.planarFull {
			if !box.Intersects(paddedFaceBox(other.planar[j])) {
				continue
			}
			curves, ok := wallPairImprint(wf, other.planarFull[j], otherImp, j)
			if !ok {
				return nil, false
			}
			out[i] = append(out[i], curves...)
		}
	}
	return out, true
}

// wallOverlapsUncovered reports a wall overlapping an other-operand face the ruling imprint cannot
// cover (a wall or pass face — curved-vs-curved contact stays with the bespoke handlers). A uv face is
// NOT uncovered any more: pairUVWallImprints imprints every uv×wall pair in closed form (#3460).
//
// A box overlap alone is not contact. Where the two SURFACES are provably apart everywhere
// (geom.SurfacesApart) the pair cannot touch however their boxes sit, so it does not make the wall
// uncovered — an emboss pad seated on a chamfer cone overlaps that cone's box completely while
// riding a constant sagitta clear of it (#3459).
func wallOverlapsUncovered(wf curvedFace, box math.Box, other *facePartition) bool {
	return overlapsUnprovenPair(wf, box, other.wall, other.wallBox) ||
		overlapsUnprovenPair(wf, box, other.pass, other.passBox)
}

// overlapsUnprovenPair reports whether wf's box overlaps any of the given faces WITHOUT a
// surface-separation proof for that pair.
func overlapsUnprovenPair(wf curvedFace, box math.Box, faces []curvedFace, boxes []math.Box) bool {
	for i, b := range boxes {
		if !box.Intersects(b) {
			continue
		}
		if i < len(faces) && geom.SurfacesApart(wf.surface, faces[i].surface, facePairCullPad) {
			continue
		}
		return true
	}
	return false
}

// ruledSide is a wall face resolved to the ruled surface it lies on, its rim band, and its axis —
// everything the imprint and the split need without knowing whether it is a cylinder or a cone.
type ruledSide struct {
	surface geom.Surface
	band    coneSideBand_
	axis    math.Vector3
}

// ruledSideBandOf resolves a wall face to its ruled side. A cylinder is the degenerate cone, and the
// band model already carries a separate bottom and top radius, so both go through one description
// (ADR-0058: the dispatch buckets faces by REPRESENTATION — here "ruled band" — not by surface type).
// ok=false for anything that is not a full periodic side band.
func ruledSideBandOf(f curvedFace) (ruledSide, bool) {
	if cyl, band, ok := fullCylinderSideBand(f); ok {
		return ruledSide{surface: cyl, band: band, axis: cyl.AxisDir.AsVector()}, true
	}
	if cone, band, ok := fullConeSideBand(f); ok {
		return ruledSide{surface: cone, band: band, axis: cone.AxisDir.AsVector()}, true
	}
	return ruledSide{}, false
}

// size is the band's characteristic length, for the resolution its intersections are taken at.
func (r ruledSide) size() float64 {
	return 2*stdmath.Max(r.band.rBot, r.band.rTop) + (r.band.vMax - r.band.vMin)
}

// wallPairImprint is the exact shared imprint of one (wall, planar face) pair: every plane∩wall
// ruling line clipped to the tool face's trim, emitted as segments for the wall AND mirrored onto the
// tool's polygonal imprint list. A circle/ellipse curve entering the tool trim declines (v1).
func wallPairImprint(wf, of curvedFace, otherImp [][][2]math.Point3, j int) ([]geom.Curve3, bool) {
	rs, ok := ruledSideBandOf(wf)
	if !ok {
		return nil, false
	}
	curves, handled := geom.IntersectSurfacesAnalytic(facePlane(of), rs.surface, geom.ResolutionForSize(rs.size()))
	if !handled {
		return nil, false
	}
	var out []geom.Curve3
	for _, cv := range curves {
		segs, ok := wallCurveSegments(cv, of, rs.axis, rs.band)
		if !ok {
			return nil, false
		}
		for _, s := range segs {
			out = append(out, geom.NewLineSegment(s[0], s[1]))
			otherImp[j] = append(otherImp[j], s)
		}
	}
	return out, true
}

// wallCurveSegments clips one plane∩wall curve to the tool face's exact trim. A ruling line yields
// its polygon intervals as segments; a circle/ellipse that stays wholly clear of the trim yields
// nothing; one that enters it declines (the v1 conic-mirror gap). It takes the wall's AXIS rather
// than a cylinder, so a cone side band goes through the same clip (ADR-0058: the wall route is the
// ruled route, not the cylinder route).
func wallCurveSegments(cv geom.Curve3, of curvedFace, axis math.Vector3, band coneSideBand_) ([][2]math.Point3, bool) {
	if line, ok := cv.(geom.Line); ok {
		var segs [][2]math.Point3
		for _, iv := range faceLineIntervals(of, line.Origin, line.Dir.AsVector()) {
			if iv[1]-iv[0] > 1e-9 { // tol:calibrated — planar imprint overlap length (see arrange2d arrTol)
				segs = append(segs, [2]math.Point3{line.PointAt(iv[0]), line.PointAt(iv[1])})
			}
		}
		return segs, true
	}
	// Every other section a plane can cut from a ruled wall is a CONIC — a circle or ellipse when
	// the plane crosses the axis, a hyperbola branch when it runs parallel to one (an emboss pad's
	// side face on a chamfer cone). All three get the same verdict: clear of the tool's trim is no
	// imprint, entering it moves the receiving face to the exact-frame bucket, which carries the
	// curve itself rather than these straight segments (promoteConicReceivers).
	cf, ok := geom.AsConic(cv)
	if !ok {
		return nil, false
	}
	return nil, !conicTouchesTool(cv, of, axis, band, cf.AxialAmplitude(axis))
}

// wallSplitFaces trims each wall by its imprints through the ruled chart, classifying kept cells by
// the boolean's keep table over the other operand's membership oracle; a wall with no imprints keeps
// the whole-face pass-through classification; a kept Difference tool wall reverses into the cavity.
func wallSplitFaces(p facePartition, imprints [][]geom.Curve3, other insideOracle, op Op, isB bool) ([]curvedFace, bool) {
	var out []curvedFace
	for i, wf := range p.wall {
		faces, ok := wallSplitOne(wf, imprints[i], other, op, isB)
		if !ok {
			return nil, false
		}
		out = append(out, faces...)
	}
	return out, true
}

// wallSplitOne trims one wall (or classifies it whole when it has no imprints).
func wallSplitOne(wf curvedFace, imprint []geom.Curve3, other insideOracle, op Op, isB bool) ([]curvedFace, bool) {
	if len(imprint) == 0 {
		return passThroughKept([]curvedFace{wf}, other, op, isB)
	}
	rs, ok := ruledSideBandOf(wf)
	if !ok {
		return nil, false
	}
	faces, _, err := curvedSideSolidSplit(wf, rs.surface, rs.band, imprint, op, isB, other.inside)
	if err != nil {
		return nil, false
	}
	if op == Difference && isB {
		faces = reverseCurvedFaces(faces)
	}
	return faces, true
}
