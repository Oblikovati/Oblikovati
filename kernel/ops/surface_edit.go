// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"errors"
	"fmt"
	"strconv"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"

	stdmath "math"
	"oblikovati.org/kernel/ops/internal/probe"
)

// Surface-editing geometry (M10-F02). Phase A operates on planar surface bodies: a
// half-space trim (Sutherland–Hodgman clip of the boundary polygon), a planar offset
// (translate along the face normal), and mid-surface extraction from antiparallel
// planar face pairs. Curved-surface trimming/extension and offsetting are the
// phase-C face-splitting / NURBS cases: each is REFUSED at classification with a named
// decline (declineSurfaceEdit) that names the offending configuration and the shape the
// operation supports, before any geometry is built (#3393) — never a generic
// "not implemented" placeholder.

// ErrSurfaceEditUnsupported classifies every surface-edit decline: a configuration the phase-A planar
// engine does not handle (a curved face, a folded multi-face offset), refused before any geometry is
// built. A caller tells a decline from a real failure with errors.Is(err, ErrSurfaceEditUnsupported);
// the wrapped message names the offending value and the supported shape (#3393).
var ErrSurfaceEditUnsupported = errors.New("surface edit: unsupported configuration")

// declineSurfaceEdit builds a named classification decline: which op, the offending configuration
// (with its value), and the shape the op supports — the CLAUDE.md exception-message contract — wrapping
// ErrSurfaceEditUnsupported so callers can classify it.
func declineSurfaceEdit(op, offending, supports string) error {
	return fmt.Errorf("%s declined: %s; supports %s: %w", op, offending, supports, ErrSurfaceEditUnsupported)
}

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
// sheetPatch is one planar face of a surface body: its boundary polygon and outward normal.
type sheetPatch struct {
	poly   []math.Point3
	normal math.Vector3
}

// buildSheet welds the patches into one planar-faced surface body: coincident boundary vertices
// merge and faces sharing an edge reconnect (one edge per undirected vertex pair). A single
// patch yields a one-face sheet; several patches yield a quilt. Patches with <3 points are
// dropped.
//
//nolint:funlen // assembles a sheet/quilt body element-by-element (verts, shared edges, faces); length is the geometry, not logic.
func buildSheet(patches []sheetPatch, feat string) *topo.Body {
	w := &sheetWelder{index: map[[3]int64]int{}}
	rings := make([][]int, 0, len(patches))
	normals := make([]math.Vector3, 0, len(patches))
	for _, p := range patches {
		if r := w.ring(p.poly); len(r) >= 3 {
			rings = append(rings, r)
			normals = append(normals, p.normal)
		}
	}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(feat, "body", 0)))
	tv := make([]*topo.Vertex, len(w.points))
	for i, pt := range w.points {
		tv[i] = bld.AddVertex(pt, topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	edges := map[[2]int]*topo.Edge{}
	edge := func(a, b int) *topo.Edge {
		k := [2]int{a, b}
		if a > b {
			k = [2]int{b, a}
		}
		if e, ok := edges[k]; ok {
			return e
		}
		e := bld.AddEdge(geom.NewLineSegment(w.points[k[0]], w.points[k[1]]), tv[k[0]], tv[k[1]], topo.NewLineage(topo.Tok(feat, "edge", len(edges))))
		edges[k] = e
		return e
	}
	provByFace := map[*topo.Face]topo.Lineage{}
	for fi, r := range rings {
		surf, _ := geom.NewPlane(w.points[r[0]], normals[fi])
		uses := make([]topo.Use, len(r))
		for i := range r {
			a, b := r[i], r[(i+1)%len(r)]
			uses[i] = topo.Use{Edge: edge(a, b), Reversed: a > b}
		}
		lineage := topo.NewLineage(topo.Tok(feat, "patch", fi))
		provByFace[bld.AddFace(surf, lineage, topo.OuterLoop(uses...))] = lineage
	}
	body := bld.Build()
	// ADR-0043 provenance: the welded edges above carry a build-order weld counter that renumbers under
	// an upstream edit. Each patch face has a stable lineage (feat:patch#fi), so name a shared seam by
	// its two patches and a boundary edge by its one patch (geometrically ranked among that patch's
	// boundary edges), so a reference to a sheet edge survives a re-weld.
	body.RelineageByFaceProvenance(provByFace, topo.Tok(feat, "x", 0), topo.Tok(feat, "seg", 0))
	return body
}

// sheetWelder merges coincident boundary points onto a shared index list (a fine grid snap).
type sheetWelder struct {
	index  map[[3]int64]int
	points []math.Point3
}

func (w *sheetWelder) add(p math.Point3) int {
	const grid = 1e-7
	k := [3]int64{int64(stdmath.Round(float64(p.X) / grid)), int64(stdmath.Round(float64(p.Y) / grid)), int64(stdmath.Round(float64(p.Z) / grid))}
	if i, ok := w.index[k]; ok {
		return i
	}
	w.index[k] = len(w.points)
	w.points = append(w.points, p)
	return len(w.points) - 1
}

// ring welds a boundary polygon to vertex indices, dropping consecutive (and wrap-around) dups.
func (w *sheetWelder) ring(poly []math.Point3) []int {
	var out []int
	for _, p := range poly {
		i := w.add(p)
		if len(out) == 0 || out[len(out)-1] != i {
			out = append(out, i)
		}
	}
	if len(out) > 1 && out[0] == out[len(out)-1] {
		out = out[:len(out)-1]
	}
	return out
}

// TrimByPlane trims a planar surface body (single or multi-face) with a cutting plane, keeping
// the half on the plane's positive (keepPositive) or negative side. Each face's boundary polygon
// is clipped against the plane; kept faces are welded back into one sheet (a shared fold edge
// clips identically on both faces, so it reconnects). Trimming a body with any curved face
// (NURBS surface–surface trimming) is the remaining phase-C work.
func TrimByPlane(body *topo.Body, origin math.Point3, normal math.Vector3, keepPositive bool, feat string) (*topo.Body, error) {
	faces := planarFaces(body)
	if len(faces) == 0 || len(faces) != len(body.Faces()) {
		return nil, declineSurfaceEdit("TrimByPlane",
			fmt.Sprintf("body has %d curved face(s) of %d total", len(body.Faces())-len(faces), len(body.Faces())),
			"an all-planar surface body (curved surface–surface trimming is phase C, unimplemented)")
	}
	var patches []sheetPatch
	for _, f := range faces {
		clipped := clipHalfSpace(facePolygon(f), origin, normal, keepPositive)
		if len(clipped) >= 3 {
			patches = append(patches, sheetPatch{poly: clipped, normal: f.Geometry().(geom.Plane).Normal()})
		}
	}
	if len(patches) == 0 {
		return nil, errors.New("trim: nothing remains on the keep side of the cutting plane")
	}
	return buildSheet(patches, feat), nil
}

// BodyPlane returns the common plane of a planar surface body (its faces' shared plane) — for use
// as a Trim cutting tool (#1880). It reports false for an empty, non-planar, or multi-plane body
// (a curved tool surface is phase C).
func BodyPlane(body *topo.Body) (geom.Plane, bool) {
	faces := body.Faces()
	if len(faces) == 0 {
		return geom.Plane{}, false
	}
	pl, ok := faces[0].Geometry().(geom.Plane)
	if !ok {
		return geom.Plane{}, false
	}
	for _, f := range faces[1:] {
		other, ok := f.Geometry().(geom.Plane)
		if !ok || !sameDirection(pl.Normal(), other.Normal()) || stdmath.Abs(float64(geom.SignedDistanceToPlane(pl, other.Origin))) > 1e-6 {
			return geom.Plane{}, false // not a single coplanar tool surface
		}
	}
	return pl, true
}

// clipHalfSpace clips a planar polygon against one half-space (Sutherland–Hodgman),
// keeping vertices on the plane's keep side and inserting edge–plane intersections.
func clipHalfSpace(poly []math.Point3, origin math.Point3, normal math.Vector3, keepPositive bool) []math.Point3 {
	var out []math.Point3
	n := len(poly)
	for i := range n {
		cur, nxt := poly[i], poly[(i+1)%n]
		ds := keepDistance(cur, origin, normal, keepPositive)
		de := keepDistance(nxt, origin, normal, keepPositive)
		if ds >= 0 {
			out = append(out, cur)
		}
		if (ds < 0) != (de < 0) {
			out = append(out, planeCrossing(cur, nxt, ds, de))
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

// planeCrossing returns the point where segment a→b crosses the cutting plane,
// given the signed keep-distances at each end.
func planeCrossing(a, b math.Point3, da, db float64) math.Point3 {
	return a.Lerp(b, da/(da-db))
}

// OffsetSurface offsets a single-face planar surface body by distance along its face
// normal, producing a new parallel surface body. Curved-face offset is phase C.
func OffsetSurface(body *topo.Body, distance float64, feat string) (*topo.Body, error) {
	faces := planarFaces(body)
	if len(faces) == 0 || len(faces) != len(body.Faces()) {
		return nil, declineSurfaceEdit("OffsetSurface",
			fmt.Sprintf("body has %d curved face(s) of %d total", len(body.Faces())-len(faces), len(body.Faces())),
			"an all-planar surface body (curved-face offset is phase C, unimplemented)")
	}
	// Each face translates along its own normal. That reconnects only when faces sharing an edge
	// have the same normal (a coplanar quilt or a single face); a folded sheet would split at the
	// fold (its offset needs intersecting adjacent offset planes — a later refinement).
	n0 := faces[0].Geometry().(geom.Plane).Normal()
	patches := make([]sheetPatch, len(faces))
	for i, f := range faces {
		nrm := f.Geometry().(geom.Plane).Normal()
		if i > 0 && !sameDirection(nrm, n0) {
			return nil, declineSurfaceEdit("OffsetSurface",
				fmt.Sprintf("face %d normal %v is not parallel to face 0 normal %v (a folded sheet)", i, nrm, n0),
				"a coplanar quilt or a single planar face (a folded multi-face offset needs intersecting adjacent offset planes, unimplemented)")
		}
		shift := nrm.Scale(distance)
		src := facePolygon(f)
		moved := make([]math.Point3, len(src))
		for j, p := range src {
			moved[j] = p.TranslateBy(shift)
		}
		patches[i] = sheetPatch{poly: moved, normal: nrm}
	}
	return buildSheet(patches, feat), nil
}

// ExtendByEdge extends a planar surface's boundary edge outward by distance, growing the face:
// the edge's two endpoints slide along the in-plane direction perpendicular to the edge, away
// from the face. A multi-face body or a curved face is the remaining phase-C (NURBS) work.
func ExtendByEdge(body *topo.Body, edgeKey []byte, distance float64, feat string) (*topo.Body, error) {
	edge, ok := body.FindEdgeByKey(edgeKey)
	if !ok {
		return nil, errors.New("extend: edge reference lost")
	}
	faces := edge.Faces()
	if len(faces) != 1 {
		return nil, declineSurfaceEdit("ExtendByEdge",
			fmt.Sprintf("edge is shared by %d faces", len(faces)),
			"a boundary edge on exactly one planar face (multi-face extend is phase C, unimplemented)")
	}
	pl, ok := faces[0].Geometry().(geom.Plane)
	if !ok {
		return nil, declineSurfaceEdit("ExtendByEdge",
			fmt.Sprintf("host face geometry is %T, not a plane", faces[0].Geometry()),
			"a planar host face (curved-surface extend is phase C, unimplemented)")
	}
	poly := facePolygon(faces[0])
	a, b := edge.StartVertex().Point(), edge.EndVertex().Point()
	shift := extendDir(pl.Normal(), a, b, probe.Centroid(poly)).Scale(distance)
	moved := make([]math.Point3, len(poly))
	for i, p := range poly {
		if coincidentPt(p, a) || coincidentPt(p, b) {
			moved[i] = p.TranslateBy(shift)
		} else {
			moved[i] = p
		}
	}
	return buildSheet([]sheetPatch{{poly: moved, normal: pl.Normal()}}, feat), nil
}

// extendDir is the in-plane unit direction perpendicular to edge a→b pointing away from the
// face centroid c (so an extended boundary grows outward).
func extendDir(normal math.Vector3, a, b, c math.Point3) math.Vector3 {
	perp := unit3(normal.Cross(a.VectorTo(b)))
	if float64(perp.Dot(a.Midpoint(b).VectorTo(c))) > 0 { // points toward the interior → flip
		perp = perp.Scale(-1)
	}
	return perp
}

func coincidentPt(a, b math.Point3) bool { return float64(a.DistanceTo(b)) < 1e-7 }

// sameDirection reports whether two normals point the same way (within tolerance).
func sameDirection(a, b math.Vector3) bool {
	return float64(unit3(a).Dot(unit3(b))) > 1-1e-7
}

func unit3(v math.Vector3) math.Vector3 {
	if u, err := math.UnitVector3FromVector(v); err == nil {
		return u.AsVector()
	}
	return v
}

// MidPatch is one extracted mid-surface: the surface body lying halfway between a
// paired set of faces, plus the recorded wall thickness between them (for FEA, M18).
// Min/Max bound the thickness across the pair (equal for perfectly parallel walls;
// they differ when the walls taper), mirroring Inventor's MidSurfaceThickness (#1885).
type MidPatch struct {
	Body      *topo.Body
	Thickness float64 // representative (centroid) separation
	Min       float64
	Max       float64
}

// MidSurfaces extracts mid-surfaces from a solid's antiparallel planar face pairs whose
// separation is within [minThickness, maxThickness] (the thin-wall pairs). Each yields a patch
// on the mid-plane and the recorded thickness range. Curved walls are phase C.
func MidSurfaces(body *topo.Body, minThickness, maxThickness float64, feat string) ([]MidPatch, error) {
	faces := planarFaces(body)
	if len(faces) < 2 {
		return nil, errors.New("mid-surface: body has fewer than two planar faces")
	}
	var patches []MidPatch
	used := make([]bool, len(faces))
	for i := range faces {
		j := matchOpposite(faces, used, i, minThickness, maxThickness)
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

// MidSurfacesByPairs extracts mid-surfaces for an explicit set of face-key pairs (Inventor's
// manual pairing, for ribs/bosses where auto-pairing fails), reporting each pair's thickness
// range. A lost or non-planar face makes the op error so the feature can go Sick (#1885).
func MidSurfacesByPairs(body *topo.Body, pairs [][2][]byte, feat string) ([]MidPatch, error) {
	if len(pairs) == 0 {
		return nil, errors.New("mid-surface: no face pairs")
	}
	patches := make([]MidPatch, 0, len(pairs))
	for _, pr := range pairs {
		a, aok := body.FindFaceByKey(pr[0])
		b, bok := body.FindFaceByKey(pr[1])
		if !aok || !bok {
			return nil, errors.New("mid-surface: face pair reference lost")
		}
		if _, ok := a.Geometry().(geom.Plane); !ok {
			return nil, errors.New("mid-surface: paired face is not planar")
		}
		if _, ok := b.Geometry().(geom.Plane); !ok {
			return nil, errors.New("mid-surface: paired face is not planar")
		}
		patches = append(patches, midPatch(a, b, feat, len(patches)))
	}
	return patches, nil
}

// matchOpposite finds an unused face antiparallel to face i and separated from it by a distance
// within [minThickness, maxThickness] (a thin-wall pair), returning its index or -1.
func matchOpposite(faces []*topo.Face, used []bool, i int, minThickness, maxThickness float64) int {
	ni := faces[i].Geometry().(geom.Plane).Normal()
	ci := probe.Centroid(facePolygon(faces[i]))
	for j := i + 1; j < len(faces); j++ {
		if used[j] {
			continue
		}
		nj := faces[j].Geometry().(geom.Plane).Normal()
		if ni.Dot(nj) > -0.999 { // not antiparallel
			continue
		}
		sep := stdmath.Abs(ci.VectorTo(probe.Centroid(facePolygon(faces[j]))).Dot(ni))
		if sep > 1e-9 && sep >= minThickness && sep <= maxThickness {
			return j
		}
	}
	return -1
}

// midPatch builds the mid-surface patch between two antiparallel faces by shifting
// face a's polygon halfway toward b, and records the separation and its range.
func midPatch(a, b *topo.Face, feat string, idx int) MidPatch {
	na := a.Geometry().(geom.Plane).Normal()
	polyA := facePolygon(a)
	sep := stdmath.Abs(probe.Centroid(polyA).VectorTo(probe.Centroid(facePolygon(b))).Dot(na))
	shift := na.Scale(sep / 2) // a's normal points toward b's side or away; sign set below
	if probe.Centroid(polyA).VectorTo(probe.Centroid(facePolygon(b))).Dot(na) < 0 {
		shift = shift.Negate()
	}
	moved := make([]math.Point3, len(polyA))
	for i, p := range polyA {
		moved[i] = p.TranslateBy(shift)
	}
	lo, hi := midThicknessRange(polyA, b.Geometry().(geom.Plane))
	return MidPatch{
		Body:      buildSheet([]sheetPatch{{poly: moved, normal: na}}, feat+"-mid-"+strconv.Itoa(idx)),
		Thickness: sep, Min: lo, Max: hi,
	}
}

// midThicknessRange returns the min and max perpendicular distance from face a's vertices to
// face b's plane — the wall-thickness range across the pair (equal for parallel walls).
func midThicknessRange(polyA []math.Point3, planeB geom.Plane) (float64, float64) {
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, p := range polyA {
		d := stdmath.Abs(float64(geom.SignedDistanceToPlane(planeB, p)))
		lo, hi = stdmath.Min(lo, d), stdmath.Max(hi, d)
	}
	return lo, hi
}
