// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The sphere bucket of the mixed per-face boolean (ADR-0061 stage 3). A sphere had no chart at all: it
// went to the pass-through bucket, which can carry a face only when it is provably clear of the other
// operand, so a tool that met it declined the whole boolean. The half-space cut is the only reason a
// sphere had a split pipeline of its own, and charting it here is what lets that pipeline be deleted
// (ADR-0062).
//
// A plane sections a sphere in a CIRCLE — one closed form, no cases — and the circle is written into
// both sides' imprint lists so the two faces split on identical coordinates and their fragments weld.

// sphereImprints plans every (sphere face, EXACT-FRAME face of other) pair's shared imprint, writing
// the SAME circle into both sides' lists so the two faces split on identical coordinates and their
// fragments weld. The receiving face is in the exact-frame bucket because promoteConicReceivers moved
// it there: a sampled polyline on one side and an exact circle on the other cannot weld, which is the
// same reason a wall's conic section promotes its receiver.
//
// ok=false declines the boolean when a sphere meets a face this pairing does not cover.
func sphereImprints(p, other *facePartition, otherUV [][]geom.Curve3) ([][]geom.Curve3, bool) {
	out := make([][]geom.Curve3, len(p.sphere))
	for i, sf := range p.sphere {
		box := inflateBox(p.sphereBox[i])
		if sphereOverlapsUncovered(sf, box, other) {
			return nil, false
		}
		for j, uf := range other.uv {
			if !box.Intersects(inflateBox(other.uvBox[j])) {
				continue
			}
			curves, ok := sphereUVSharedImprint(sf, uf)
			if !ok {
				return nil, false
			}
			out[i] = append(out[i], curves...)
			otherUV[j] = append(otherUV[j], curves...)
		}
	}
	return out, true
}

// sphereOverlapsUncovered reports a sphere meeting a face of the other operand this pairing does not
// cover. A POLYGONAL planar face is uncovered only when the sphere's section with its plane actually
// ENTERS its trim — a box overlap alone is not contact, and the bounding prism a half-space cut is has
// side walls whose boxes span the whole body while their planes miss the sphere entirely. A face the
// section does enter should have been promoted to the exact-frame bucket; one that was not (it carries
// detached holes, or its frame declined) is a genuine gap and declines.
//
// A wall, another sphere or a pass face still declines on box overlap: curved-versus-curved contact
// stays with the bespoke recognisers until the crossings are charted (ADR-0061 stage 4).
func sphereOverlapsUncovered(sf curvedFace, box math.Box, other *facePartition) bool {
	for i := range other.planar {
		if box.Intersects(paddedFaceBox(other.planar[i])) && sphereSectionEnters(sf, other.planarFull[i]) {
			return true
		}
	}
	for _, b := range other.wallBox {
		if box.Intersects(inflateBox(b)) {
			return true
		}
	}
	for _, b := range other.sphereBox {
		if box.Intersects(inflateBox(b)) {
			return true
		}
	}
	for _, b := range other.passBox {
		if box.Intersects(inflateBox(b)) {
			return true
		}
	}
	return false
}

// sphereSectionEnters reports whether the plane of of cuts the sphere in a circle that enters of's own
// trim — the exact contact test, not a box overlap.
func sphereSectionEnters(sf, of curvedFace) bool {
	s, ok := geom.SphereOf(sf.surface)
	if !ok {
		return true // not a sphere after all: be conservative
	}
	curves, handled := geom.IntersectSurfacesAnalytic(facePlane(of), s, geom.ResolutionForSize(2*s.Radius))
	if !handled {
		return true
	}
	for _, cv := range curves {
		if island, exact := conicIslandInFace(cv, of); !exact || island {
			return true
		}
	}
	return false
}

// sphereUVSharedImprint is the exact shared imprint of one (sphere, exact-frame planar face) pair: the
// plane∩sphere CIRCLE — one closed form, no cases — kept when it enters the planar face's trim and
// dropped when it stays clear.
func sphereUVSharedImprint(sf, uf curvedFace) ([]geom.Curve3, bool) {
	s, ok := geom.SphereOf(sf.surface)
	if !ok {
		return nil, false
	}
	curves, handled := geom.IntersectSurfacesAnalytic(facePlane(uf), s, geom.ResolutionForSize(2*s.Radius))
	if !handled {
		return nil, false
	}
	var out []geom.Curve3
	for _, cv := range curves {
		island, exact := conicIslandInFace(cv, uf)
		if !exact {
			return nil, false // the section's relation to the trim is not decided in closed form
		}
		if island {
			out = append(out, cv)
		}
	}
	return out, true
}

// sphereSplitFaces trims each sphere face by its imprints through the loop-framed sphere chart,
// classifying kept cells by the boolean's keep table over the other operand's membership. A sphere with
// no imprint keeps the whole-face pass-through classification.
func sphereSplitFaces(p facePartition, imprints [][]geom.Curve3, other insideOracle, op Op, isB bool) ([]curvedFace, bool) {
	var out []curvedFace
	for i, sf := range p.sphere {
		faces, ok := sphereSplitOne(sf, imprints[i], other, op, isB)
		if !ok {
			return nil, false
		}
		out = append(out, faces...)
	}
	return out, true
}

// sphereSplitOne trims one sphere face (or classifies it whole when it has no imprints).
func sphereSplitOne(sf curvedFace, imprint []geom.Curve3, other insideOracle, op Op, isB bool) ([]curvedFace, bool) {
	if len(imprint) == 0 {
		return passThroughKept([]curvedFace{sf}, other, op, isB)
	}
	s, ok := sphereFaceOf(sf)
	if !ok {
		return nil, false
	}
	c := newSphereFaceUV(sf, s, op, isB, other.inside)
	faces, _, err := trimByImprint(c, sf, s, imprint, sphereFaceMaterial(c))
	if err != nil {
		return nil, false
	}
	faces = boundedTrims(faces)
	if op == Difference && isB {
		faces = reverseCurvedFaces(faces)
	}
	return faces, true
}

// boundedTrims drops a trimmed face that came back with NO boundary, unless it is the only one. A
// sphere's parameter rectangle is closed at the poles by degenerate edges, and a cell bounded by
// nothing else emits a face whose every edge finalizeLoops removes — the whole surface a second time,
// beside the cap that was actually cut. A boundary-less face is meaningful only when the trim left the
// surface whole.
func boundedTrims(faces []curvedFace) []curvedFace {
	if len(faces) <= 1 {
		return faces
	}
	out := make([]curvedFace, 0, len(faces))
	for _, f := range faces {
		if len(f.loops) > 0 {
			out = append(out, f)
		}
	}
	return out
}
