// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Public body imprinting (M07-F05, Oblikovati/Oblikovati#628): the reference
// TransientBRep.ImprintBodies. Each body's faces split along its face–face
// intersections with the other body, WITHOUT removing material — both results
// keep their full skins, with new edges where the bodies touch. Planar-faced
// bodies only (the same constraint as the exact boolean); curved operands
// return ErrNonPlanar.

// ImprintResult carries one imprinted copy plus the faces that were touched
// (split or coincident with the other body).
type ImprintResult struct {
	Body          *topo.Body
	TouchedFaces  []*topo.Face
	ImprintedEdge []*topo.Edge
}

// ImprintBodies imprints a and b against each other, returning new bodies.
//
// Example: ra, rb, err := brep.ImprintBodies(a, b)
func ImprintBodies(a, b *topo.Body) (ImprintResult, ImprintResult, error) {
	fa, oka := facesOf(a)
	fb, okb := facesOf(b)
	if !oka || !okb {
		return ImprintResult{}, ImprintResult{}, ErrNonPlanar
	}
	impA, impB := imprintAll(fa, fb)
	ra, err := rebuildImprinted(fa, impA)
	if err != nil {
		return ImprintResult{}, ImprintResult{}, err
	}
	rb, err := rebuildImprinted(fb, impB)
	if err != nil {
		return ImprintResult{}, ImprintResult{}, err
	}
	return ra, rb, nil
}

// rebuildImprinted splits each face along its imprints and stitches ALL pieces
// back (no classification — nothing is removed; the stitcher re-derives
// solidity from edge pairing), tracking the touched faces and the new
// imprint-lying edges.
func rebuildImprinted(faces []planarFace, imprints [][][2]math.Point3) (ImprintResult, error) {
	var kept []subFace
	touched := map[string]bool{}
	for i, f := range faces {
		pieces := splitFace(f, imprints[i])
		for k := range pieces {
			if len(pieces) == 1 {
				pieces[k].lineage = f.lineage
			} else {
				pieces[k].lineage = splitLineage(f.lineage, k)
			}
		}
		if len(imprints[i]) > 0 {
			touched[string(f.lineage.Key())] = true
		}
		kept = append(kept, pieces...)
	}
	body, _, err := stitch(kept)
	if err != nil || body == nil {
		return ImprintResult{}, err
	}
	return collectImprintTouches(body, touched, allImprints(imprints)), nil
}

// collectImprintTouches resolves the touched faces and imprint edges on the
// rebuilt body: a face whose lineage starts with a touched source lineage, an
// edge whose midpoint lies on an imprint segment.
func collectImprintTouches(body *topo.Body, touched map[string]bool, segs [][2]math.Point3) ImprintResult {
	out := ImprintResult{Body: body}
	for _, f := range body.Faces() {
		if lineagePrefixIn(f.Lineage(), touched) {
			out.TouchedFaces = append(out.TouchedFaces, f)
		}
	}
	for _, e := range body.Edges() {
		if edgeOnSegments(e, segs) {
			out.ImprintedEdge = append(out.ImprintedEdge, e)
		}
	}
	return out
}

func allImprints(imprints [][][2]math.Point3) [][2]math.Point3 {
	var out [][2]math.Point3
	for _, per := range imprints {
		out = append(out, per...)
	}
	return out
}

// lineagePrefixIn reports whether the lineage or any token-prefix of it is in
// the touched set (split pieces carry the parent lineage + a split token).
func lineagePrefixIn(l topo.Lineage, touched map[string]bool) bool {
	tokens := l.Tokens()
	for n := len(tokens); n > 0; n-- {
		if touched[string(topo.NewLineage(tokens[:n]...).Key())] {
			return true
		}
	}
	return false
}

// edgeOnSegments reports whether the edge's midpoint lies on any imprint
// segment (within the boolean weld tolerance).
func edgeOnSegments(e *topo.Edge, segs [][2]math.Point3) bool {
	c := e.Geometry()
	lo, hi := c.Domain()
	mid := c.PointAt((lo + hi) / 2)
	for _, s := range segs {
		if pointOnSegment3(mid, s[0], s[1], 1e-7) {
			return true
		}
	}
	return false
}

func pointOnSegment3(p, a, b math.Point3, tol float64) bool {
	ab := a.VectorTo(b)
	denom := float64(ab.LengthSquared())
	if denom == 0 {
		return float64(p.DistanceTo(a)) <= tol
	}
	t := float64(a.VectorTo(p).Dot(ab)) / denom
	if t < 0 || t > 1 {
		return false
	}
	on := a.TranslateBy(ab.Scale(math.Scalar(t)))
	return float64(p.DistanceTo(on)) <= tol
}
