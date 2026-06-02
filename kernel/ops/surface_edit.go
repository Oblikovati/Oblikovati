// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"errors"
	"strconv"

	stdmath "math"

	"github.com/Oblikovati/oblikovati/build"
	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// Surface-editing geometry (M10-F02). Phase A operates on planar surface bodies: a
// half-space trim (Sutherland–Hodgman clip of the boundary polygon), a planar offset
// (translate along the face normal), and mid-surface extraction from antiparallel
// planar face pairs. Curved-surface trimming/extension and offsetting are the
// phase-C face-splitting / NURBS cases, deferred behind NotYetImplemented.

// planarFaces returns the body's faces whose geometry is a plane.
func planarFaces(body *topo.Body) []*topo.Face {
	var out []*topo.Face
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Plane); ok {
			out = append(out, f)
		}
	}
	return out
}

// facePolygon returns a planar face's outer boundary as model-space points in
// traversal order (each vertex once).
func facePolygon(f *topo.Face) []math.Point3 {
	var pts []math.Point3
	for _, l := range f.Loops() {
		if !l.IsOuter() {
			continue
		}
		for _, u := range l.EdgeUses() {
			pts = append(pts, useStart(u))
		}
	}
	return pts
}

// useStart returns the model-space start point of an oriented edge use.
func useStart(u *topo.EdgeUse) math.Point3 {
	if u.Reversed() {
		return u.Edge().EndVertex().Point()
	}
	return u.Edge().StartVertex().Point()
}

// buildPlanarBody builds a one-face planar surface body from an ordered model-space
// polygon and a surface normal, with stable per-feature lineage.
func buildPlanarBody(poly []math.Point3, normal math.Vector3, feat string) *topo.Body {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(feat, "body", 0)))
	surf, _ := geom.NewPlane(poly[0], normal)
	n := len(poly)
	v := make([]*topo.Vertex, n)
	for i, p := range poly {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	uses := make([]topo.Use, n)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		e := bld.AddEdge(geom.NewLineSegment(poly[i], poly[j]), v[i], v[j], topo.NewLineage(topo.Tok(feat, "edge", i)))
		uses[i] = topo.Fwd(e)
	}
	bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "patch", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}

// TrimByPlane trims a single-face planar surface body with a cutting plane, keeping
// the half on the plane's positive (keepPositive) or negative side. It clips the
// boundary polygon against the cutting plane and rebuilds the trimmed patch. General
// surface–surface trimming (curved tools, multi-face splitting) is phase C.
func TrimByPlane(body *topo.Body, origin math.Point3, normal math.Vector3, keepPositive bool, feat string) (*topo.Body, error) {
	faces := planarFaces(body)
	if len(faces) != 1 {
		return nil, build.NotYetImplemented("PBI-111-trim-multiface-or-curved")
	}
	clipped := clipHalfSpace(facePolygon(faces[0]), origin, normal, keepPositive)
	if len(clipped) < 3 {
		return nil, errors.New("trim: nothing remains on the keep side of the cutting plane")
	}
	return buildPlanarBody(clipped, faces[0].Geometry().(geom.Plane).Normal(), feat), nil
}

// clipHalfSpace clips a planar polygon against one half-space (Sutherland–Hodgman),
// keeping vertices on the plane's keep side and inserting edge–plane intersections.
func clipHalfSpace(poly []math.Point3, origin math.Point3, normal math.Vector3, keepPositive bool) []math.Point3 {
	var out []math.Point3
	n := len(poly)
	for i := 0; i < n; i++ {
		cur, nxt := poly[i], poly[(i+1)%n]
		ds := keepDistance(cur, origin, normal, keepPositive)
		de := keepDistance(nxt, origin, normal, keepPositive)
		if ds >= 0 {
			out = append(out, cur)
		}
		if (ds < 0) != (de < 0) {
			out = append(out, lerpToPlane(cur, nxt, ds, de))
		}
	}
	return out
}

// keepDistance is the signed distance from p to the cutting plane, positive on the
// kept side.
func keepDistance(p, origin math.Point3, normal math.Vector3, keepPositive bool) float64 {
	d := origin.VectorTo(p).Dot(normal)
	if keepPositive {
		return d
	}
	return -d
}

// lerpToPlane returns the point where segment a→b crosses the cutting plane, given
// the signed keep-distances at each end.
func lerpToPlane(a, b math.Point3, da, db float64) math.Point3 {
	t := da / (da - db)
	return a.TranslateBy(a.VectorTo(b).Scale(t))
}

// OffsetSurface offsets a single-face planar surface body by distance along its face
// normal, producing a new parallel surface body. Curved-face offset is phase C.
func OffsetSurface(body *topo.Body, distance float64, feat string) (*topo.Body, error) {
	faces := planarFaces(body)
	if len(faces) != 1 {
		return nil, build.NotYetImplemented("PBI-112-offset-multiface-or-curved")
	}
	normal := faces[0].Geometry().(geom.Plane).Normal()
	shift := normal.Scale(distance)
	src := facePolygon(faces[0])
	moved := make([]math.Point3, len(src))
	for i, p := range src {
		moved[i] = p.TranslateBy(shift)
	}
	return buildPlanarBody(moved, normal, feat), nil
}

// MidPatch is one extracted mid-surface: the surface body lying halfway between a
// paired set of faces, plus the recorded wall thickness between them (for FEA, M18).
type MidPatch struct {
	Body      *topo.Body
	Thickness float64
}

// MidSurfaces extracts mid-surfaces from a solid's antiparallel planar face pairs
// whose separation is within maxThickness (the thin-wall pairs). Each yields a patch
// on the mid-plane and the recorded thickness. Curved walls are phase C.
func MidSurfaces(body *topo.Body, maxThickness float64, feat string) ([]MidPatch, error) {
	faces := planarFaces(body)
	if len(faces) < 2 {
		return nil, errors.New("mid-surface: body has fewer than two planar faces")
	}
	var patches []MidPatch
	used := make([]bool, len(faces))
	for i := 0; i < len(faces); i++ {
		j := matchOpposite(faces, used, i, maxThickness)
		if j < 0 {
			continue
		}
		used[i], used[j] = true, true
		patches = append(patches, midPatch(faces[i], faces[j], feat, len(patches)))
	}
	if len(patches) == 0 {
		return nil, errors.New("mid-surface: no thin face pair within the thickness threshold")
	}
	return patches, nil
}

// matchOpposite finds an unused face antiparallel to face i and separated from it by
// at most maxThickness (a thin-wall pair), returning its index or -1.
func matchOpposite(faces []*topo.Face, used []bool, i int, maxThickness float64) int {
	ni := faces[i].Geometry().(geom.Plane).Normal()
	ci := centroid(facePolygon(faces[i]))
	for j := i + 1; j < len(faces); j++ {
		if used[j] {
			continue
		}
		nj := faces[j].Geometry().(geom.Plane).Normal()
		if ni.Dot(nj) > -0.999 { // not antiparallel
			continue
		}
		sep := stdmath.Abs(ci.VectorTo(centroid(facePolygon(faces[j]))).Dot(ni))
		if sep > 1e-9 && sep <= maxThickness {
			return j
		}
	}
	return -1
}

// midPatch builds the mid-surface patch between two antiparallel faces by shifting
// face a's polygon halfway toward b, and records the separation as the thickness.
func midPatch(a, b *topo.Face, feat string, idx int) MidPatch {
	na := a.Geometry().(geom.Plane).Normal()
	polyA := facePolygon(a)
	sep := stdmath.Abs(centroid(polyA).VectorTo(centroid(facePolygon(b))).Dot(na))
	shift := na.Scale(sep / 2) // a's normal points toward b's side or away; sign set below
	if centroid(polyA).VectorTo(centroid(facePolygon(b))).Dot(na) < 0 {
		shift = shift.Negate()
	}
	moved := make([]math.Point3, len(polyA))
	for i, p := range polyA {
		moved[i] = p.TranslateBy(shift)
	}
	return MidPatch{Body: buildPlanarBody(moved, na, feat+"-mid-"+strconv.Itoa(idx)), Thickness: sep}
}

// centroid returns the average of a polygon's vertices.
func centroid(poly []math.Point3) math.Point3 {
	var sx, sy, sz float64
	for _, p := range poly {
		sx, sy, sz = sx+p.X, sy+p.Y, sz+p.Z
	}
	n := float64(len(poly))
	return math.P3(sx/n, sy/n, sz/n)
}
