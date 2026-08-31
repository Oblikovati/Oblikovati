// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Cylinder-wall splits for the mixed per-face boolean (ADR-0058). A full-band cylinder wall crossed
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
		box := inflateBox(p.wallBox[i], facePairCullPad)
		if wallOverlapsUncovered(box, other) {
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

// wallOverlapsUncovered reports a wall box overlapping an other-operand face the ruling imprint cannot
// cover (a uv or pass face — curved-vs-curved contact stays with the bespoke handlers).
func wallOverlapsUncovered(box math.Box, other *facePartition) bool {
	return boxesOverlapAny(box, other.uvBox) || boxesOverlapAny(box, other.wallBox) || boxesOverlapAny(box, other.passBox)
}

// wallPairImprint is the exact shared imprint of one (wall, planar face) pair: every plane∩cylinder
// ruling line clipped to the tool face's trim, emitted as segments for the wall AND mirrored onto the
// tool's polygonal imprint list. A circle/ellipse curve entering the tool trim declines (v1).
func wallPairImprint(wf, of curvedFace, otherImp [][][2]math.Point3, j int) ([]geom.Curve3, bool) {
	cyl, band, ok := fullCylinderSideBand(wf)
	if !ok {
		return nil, false
	}
	res := geom.ResolutionForSize(2*cyl.Radius + (band.vMax - band.vMin))
	curves, handled := geom.IntersectSurfacesAnalytic(facePlane(of), cyl, res)
	if !handled {
		return nil, false
	}
	var out []geom.Curve3
	for _, cv := range curves {
		segs, ok := wallCurveSegments(cv, of, cyl, band)
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

// wallCurveSegments clips one plane∩cylinder curve to the tool face's exact trim. A ruling line
// yields its polygon intervals as segments; a circle/ellipse that stays wholly clear of the trim
// yields nothing; one that enters it declines (the v1 conic-mirror gap).
func wallCurveSegments(cv geom.Curve3, of curvedFace, cyl geom.Cylinder, band coneSideBand_) ([][2]math.Point3, bool) {
	switch c := cv.(type) {
	case geom.Line:
		var segs [][2]math.Point3
		for _, iv := range faceLineIntervals(of, c.Origin, c.Dir.AsVector()) {
			if iv[1]-iv[0] > 1e-9 { // tol:calibrated — planar imprint overlap length (see arrange2d arrTol)
				segs = append(segs, [2]math.Point3{c.PointAt(iv[0]), c.PointAt(iv[1])})
			}
		}
		return segs, true
	case geom.Circle:
		return nil, !conicTouchesTool(cv, of, cyl, band, 0)
	case geom.EllipseFull:
		return nil, !conicTouchesTool(cv, of, cyl, band, ellipseAxialAmplitude(c, cyl))
	default:
		return nil, false
	}
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
	cyl, band, ok := fullCylinderSideBand(wf)
	if !ok {
		return nil, false
	}
	faces, _, err := cylinderSideSolidSplit(wf, cyl, band, imprint, op, isB, other.inside)
	if err != nil {
		return nil, false
	}
	if op == Difference && isB {
		faces = reverseCurvedFaces(faces)
	}
	return faces, true
}
