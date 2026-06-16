// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// EdgeFilletRadii is one picked edge with its blend radius at each end: R0 at the edge's
// start vertex, R1 at its end vertex. Equal radii give a constant fillet; differing radii a
// variable fillet whose radius runs linearly along the edge (#323).
type EdgeFilletRadii struct {
	Key    []byte
	R0, R1 float64
}

// FilletEdges rounds the selected convex straight edges of a planar solid with a constant-
// radius rolling-ball blend: each edge between two planar faces is replaced by a cylinder
// face of radius r tangent to both, the two faces are retrimmed back to the tangent lines,
// and the end faces gain a quarter-arc at the rounded corner. All edges are resolved and
// solved on the original body, then applied in a single rebuild, so independent edges that
// share a face (e.g. the four verticals of a box) all retrim that face correctly. Convex,
// straight edges with one extra face at each end (box/prism edges); chains, corners where
// fillets meet, and concave edges are follow-ups.
func FilletEdges(body *topo.Body, edgeKeys [][]byte, r float64) (*topo.Body, error) {
	picks := make([]EdgeFilletRadii, len(edgeKeys))
	for i, k := range edgeKeys {
		picks[i] = EdgeFilletRadii{Key: k, R0: r, R1: r}
	}
	return FilletEdgesVarying(body, picks)
}

// FilletEdgesVarying is FilletEdges with a per-end radius for each edge. A variable edge's
// blend is a generalized cone (the rolling ball grows linearly along the edge), built as
// planar trapezoids between successive rulings: adjacent rulings meet at the cone's apex on
// the edge line, so each strip face is EXACTLY planar — the only approximation is the end
// arcs as chords, the same density convention as a hole's faceted cylinder.
func FilletEdgesVarying(body *topo.Body, picks []EdgeFilletRadii) (*topo.Body, error) {
	return FilletEdgesCorner(body, picks, CornerMiter)
}

// CornerStrategy selects how a corner where two filleted edges meet a vertex whose third edge stays
// sharp is treated (mirrors api types.FilletCornerType). Round and setback both augment the selection
// with the sharp third edge — round at constant radius, setback as a taper that runs out to 0 — so
// the corner resolves as a 3-edge sphere blend; miter keeps the two edges' cylinders meeting at a seam.
type CornerStrategy int

const (
	// CornerMiter mutually trims the two cylinders along their intersection seam (a crease).
	CornerMiter CornerStrategy = iota
	// CornerSetback rounds the corner into a sphere and tapers the third edge to a run-out (set-back).
	CornerSetback
	// CornerRound rounds the corner fully by also filleting the third edge at constant radius (sphere).
	CornerRound
)

// FilletEdgesCorner is FilletEdgesVarying with an explicit 2-edge corner strategy. Lone curved
// (rim/arc) picks ignore it (they have no shared corner). CornerRound augments the selection with the
// sharp third edge of each 2-edge corner so the corner resolves as a watertight 3-edge sphere blend.
func FilletEdgesCorner(body *topo.Body, picks []EdgeFilletRadii, corner CornerStrategy) (*topo.Body, error) {
	if rim := loneRimPick(body, picks); rim != nil {
		return FilletCylinderRim(body, rim.Key, rim.R0) // a circular cylinder/cap rim → toroidal band
	}
	if arc := loneArcPick(body, picks); arc != nil {
		return FilletCylinderArc(body, arc.Key, arc.R0) // a cylinder/cap arc → torus + setback end-caps
	}
	edges, err := resolveFilletPicks(body, picks)
	if err != nil {
		return nil, err
	}
	switch corner {
	case CornerRound:
		edges = roundThirdEdges(edges) // fillet the third edge at constant radius → 3-edge sphere
	case CornerSetback:
		edges = setbackThirdEdges(edges) // taper the third edge (r→0 run-out) → smooth set-back sphere
	}
	return filletResolvedEdges(body, edges)
}

// filletResolvedEdges solves the corners and edge fillets of an already-resolved pick list and
// assembles the validated result body. Round/setback corners have already been reduced to 3-edge
// sphere blends by augmenting the third edge, so the corner solver only ever sees miters and blends.
func filletResolvedEdges(body *topo.Body, edges []filletPick) (*topo.Body, error) {
	blends, miters, err := computeCorners(edges)
	if err != nil {
		return nil, err
	}
	fils := make([]edgeFillet, 0, len(edges))
	for _, p := range edges {
		ef, err := computeEdgeFillet(body, p, blends, miters)
		if err != nil {
			return nil, err
		}
		fils = append(fils, ef)
	}
	res := assembleBody(filletResultFaces(body, fils, blends), "fillet")
	if rep := Validate(res); !rep.Valid || !res.IsSolid() {
		return nil, fmt.Errorf("fillet: result is not a valid solid %v", rep.Issues)
	}
	return res, nil
}

// filletPick is one resolved fillet input: the edge and its per-end radii.
type filletPick struct {
	edge   *topo.Edge
	r0, r1 float64
}

// varying reports whether the pick's radius changes along the edge.
func (p filletPick) varying() bool { return p.r0 != p.r1 }

// resolveFilletPicks resolves the edge reference keys against the body, erroring on a lost
// key or a non-positive radius.
func resolveFilletPicks(body *topo.Body, picks []EdgeFilletRadii) ([]filletPick, error) {
	out := make([]filletPick, 0, len(picks))
	for _, p := range picks {
		if p.R0 < 0 || p.R1 < 0 || p.R0+p.R1 <= 0 {
			return nil, fmt.Errorf("fillet: radii %g/%g must be >= 0 with at least one > 0 (a run-out tapers to 0 at one end)", p.R0, p.R1)
		}
		e, ok := body.FindEdgeByKey(p.Key)
		if !ok {
			return nil, fmt.Errorf("fillet: edge reference lost: %x", p.Key)
		}
		out = append(out, filletPick{edge: e, r0: p.R0, r1: p.R1})
	}
	return out, nil
}

// corner is one rounded end of a filleted edge: the cylinder centre at that end, the tangent
// points on faces a/b, and the arc midpoint (the cylinder point nearest the sharp corner).
// At a blend corner the centre is the corner sphere's centre and the tangent points are the
// sphere's tangents (the cylinder ends there and its arc joins the sphere patch). A variable
// fillet's corner additionally carries the arc sampled as chords (ta…tb), shared between the
// blend's ruling strips and the end face so they stay watertight.
type corner struct {
	a, b    *topo.Face
	cen     math.Point3 // cylinder centre at this end (sphere centre when blended)
	ta, tb  math.Point3
	mid     math.Point3
	chords  []math.Point3 // variable fillet only: the end arc as chord samples ta…tb
	endFace *topo.Face    // the flat end cap to arc (nil at a blend or miter corner)
	vertex  *topo.Vertex
	blend   bool
	miter   bool          // two-fillet corner: the end is bounded by seam (no end face, no sphere)
	seam    []math.Point3 // miter only: the seam chords from ta to tb, shared with the other cylinder
	runout  bool          // variable fillet only: r=0 here, the blend collapses to an apex on the edge
}

// tOf returns the tangent point on face f (a or b).
func (c corner) tOf(f *topo.Face) math.Point3 {
	if f == c.a {
		return c.ta
	}
	return c.tb
}

// edgeFillet is a fully solved fillet of one edge: its two faces, the cylinder (constant
// radius only), the two rounded corners, and whether the radius varies along the edge.
type edgeFillet struct {
	a, b    *topo.Face
	cyl     geom.Cylinder
	c0, c1  corner
	edge    *topo.Edge
	varying bool
}

// computeEdgeFillet solves the rolling-ball geometry for one convex straight edge, using a
// corner blend at either endpoint that is a shared corner. A varying pick gets its end arcs
// sampled as chords (shared by the ruling strips and the end faces).
func computeEdgeFillet(body *topo.Body, p filletPick, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter) (edgeFillet, error) {
	e := p.edge
	if cyl, pl, ok := cylinderPlaneEdge(e); ok {
		return edgeFillet{}, curvedFilletError(e, cyl, pl) // fillet of a fillet — Phase A: classify & report
	}
	a, b, nA, nB, err := edgePlanarFaces(e)
	if err != nil {
		return edgeFillet{}, err
	}
	axis, err := math.UnitVector3FromVector(e.StartVertex().Point().VectorTo(e.EndVertex().Point()))
	if err != nil {
		return edgeFillet{}, fmt.Errorf("fillet: degenerate edge")
	}
	offDir := nA.Add(nB).Scale(-1 / (1 + nA.Dot(nB))) // per-unit-radius centre offset into the solid
	rMid := (p.r0 + p.r1) / 2
	if mid := e.StartVertex().Point().Midpoint(e.EndVertex().Point()); !PointInsideBody(body, mid.TranslateBy(offDir.Scale(rMid))) {
		return edgeFillet{}, fmt.Errorf("fillet: edge is not convex (only convex edges are supported)")
	}
	in := cornerInputs{a: a, b: b, nA: nA, nB: nB, offDir: offDir, axis: axis.AsVector()}
	return solvedEdgeFillet(e, p, in, blends, miters)
}

// solvedEdgeFillet assembles the edgeFillet once the edge's frame is known: corners per end
// radius, then either the chord-sampled varying blend or the constant cylinder.
func solvedEdgeFillet(e *topo.Edge, p filletPick, in cornerInputs, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter) (edgeFillet, error) {
	c0, c1, err := edgeCorners(e, p, in, blends, miters)
	if err != nil {
		return edgeFillet{}, err
	}
	if p.varying() {
		sampleCornerChords(&c0, &c1, in)
		return edgeFillet{a: in.a, b: in.b, c0: c0, c1: c1, edge: e, varying: true}, nil
	}
	cyl, err := geom.NewCylinder(c0.cen, in.axis, p.r0)
	if err != nil {
		return edgeFillet{}, err
	}
	return edgeFillet{a: in.a, b: in.b, cyl: cyl, c0: c0, c1: c1, edge: e}, nil
}

// edgeCorners solves the rounded corners at both endpoints of an edge (each blended when its
// vertex is a shared corner), with the pick's per-end radius.
func edgeCorners(e *topo.Edge, p filletPick, in cornerInputs, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter) (c0, c1 corner, err error) {
	if c0, err = cornerAt(e.StartVertex(), in, p.r0, blends[e.StartVertex().ID()], miters[e.StartVertex().ID()], p.varying()); err != nil {
		return corner{}, corner{}, err
	}
	c1, err = cornerAt(e.EndVertex(), in, p.r1, blends[e.EndVertex().ID()], miters[e.EndVertex().ID()], p.varying())
	return c0, c1, err
}

// filletChordsPerTurn matches holeFacets' density: chords are sized as if the full circle
// had this many sides, so a 90° wedge gets 8.
const filletChordsPerTurn = 32

// runoutTol is the radius at or below which a variable fillet is treated as a run-out: the blend
// collapses to a single apex on the edge (no end face), so the fillet fades smoothly into the corner.
const runoutTol = 1e-9

// cornerChordCount is the number of chord segments spanning the corner's rolling-ball wedge — sized
// as if the full circle had filletChordsPerTurn sides (a 90° wedge gets 8), with a floor of 4.
func cornerChordCount(in cornerInputs) int {
	wedge := stdmath.Acos(float64(in.nA.Dot(in.nB)))
	k := int(stdmath.Ceil(wedge / (2 * stdmath.Pi / filletChordsPerTurn)))
	if k < 4 {
		k = 4
	}
	return k
}

// sampleCornerChords samples both corners' arcs at the same angular stations, so chord j of
// one corner pairs with chord j of the other as a straight ruling of the blend cone.
func sampleCornerChords(c0, c1 *corner, in cornerInputs) {
	k := cornerChordCount(in)
	c0.chords = arcChords(*c0, in, k)
	c1.chords = arcChords(*c1, in, k)
}

// arcChords samples a corner's arc ta…tb as k+1 points: cen + r·slerp(nA→nB), the exact
// rolling-ball contact directions at evenly spaced stations.
func arcChords(c corner, in cornerInputs, k int) []math.Point3 {
	r := c.cen.DistanceTo(c.ta)
	out := make([]math.Point3, k+1)
	for j := 0; j <= k; j++ {
		dir := slerpVec(in.nA, in.nB, float64(j)/float64(k))
		out[j] = c.cen.TranslateBy(dir.Scale(r))
	}
	return out
}

// edgePlanarFaces returns the edge's two faces and their outward normals, erroring unless
// the edge bounds exactly two planar faces.
func edgePlanarFaces(e *topo.Edge) (a, b *topo.Face, nA, nB math.Vector3, err error) {
	faces := e.Faces()
	if len(faces) != 2 {
		return nil, nil, nA, nB, fmt.Errorf("fillet: edge bounds %d faces, need 2", len(faces))
	}
	pa, oka := faces[0].Geometry().(geom.Plane)
	pb, okb := faces[1].Geometry().(geom.Plane)
	if !oka || !okb {
		return nil, nil, nA, nB, fmt.Errorf("fillet: both faces of the edge must be planar")
	}
	return faces[0], faces[1], pa.Normal(), pb.Normal(), nil
}

// cornerInputs bundles the per-edge data a corner needs. offDir is the centre offset from
// the edge into the solid PER UNIT RADIUS (a variable fillet's centre line follows offDir
// scaled by the local radius).
type cornerInputs struct {
	a, b   *topo.Face
	nA, nB math.Vector3
	offDir math.Vector3
	axis   math.Vector3
}

// cornerAt solves a fillet corner at vertex v with the local radius r. Without a blend it is
// a simple end: centre v+offDir·r, tangent points r along each face normal, an arc on the end
// face. With a blend (v is a shared corner) the centre is the blend sphere's centre and the
// tangent points are the sphere's tangents on the two faces; the corner-end arc joins the
// sphere patch (no end face), and the arc is registered on the blend.
func cornerAt(v *topo.Vertex, in cornerInputs, r float64, blend *cornerBlend, miter *cornerMiter, variable bool) (corner, error) {
	if r <= runoutTol { // a variable fillet tapered to 0: the blend collapses to an apex on the edge here
		p := v.Point()
		return corner{a: in.a, b: in.b, vertex: v, cen: p, ta: p, tb: p, mid: p, runout: true}, nil
	}
	cen := v.Point().TranslateBy(in.offDir.Scale(r)) // the rolling-ball centre, on the cylinder axis
	ta := cen.TranslateBy(in.nA.Scale(r))
	tb := cen.TranslateBy(in.nB.Scale(r))
	var end *topo.Face
	var seam []math.Point3
	switch {
	case miter != nil:
		ta, tb, seam = miterTangents(in, miter) // the end is the seam, not an end-face arc
	case blend != nil:
		cen, ta, tb = blend.center, blend.tan[in.a.ID()], blend.tan[in.b.ID()]
	default:
		if end = endFaceAt(v, in.a, in.b); end == nil {
			return corner{}, fmt.Errorf("fillet: edge endpoint has no end face to round")
		}
	}
	// mid is computed AFTER the switch so a blend corner's arc midpoint uses the sphere centre.
	mid := cen.TranslateBy(perpToward(cen, v.Point(), in.axis).Scale(r))
	c := corner{a: in.a, b: in.b, endFace: end, vertex: v, cen: cen, ta: ta, tb: tb, mid: mid, blend: blend != nil, miter: miter != nil, seam: seam}
	registerBlendArc(blend, c, in, variable)
	return c, nil
}

// registerBlendArc records the corner's boundary arc on the sphere patch when v is a blend corner. A
// variable edge stores the arc as the cone's chord polyline so the patch and cone meet edge-for-edge.
func registerBlendArc(blend *cornerBlend, c corner, in cornerInputs, variable bool) {
	if blend == nil {
		return
	}
	arc := blendArc{ta: c.ta, tb: c.tb, mid: c.mid}
	if variable {
		arc.chords = arcChords(c, in, cornerChordCount(in))
	}
	blend.arcs = append(blend.arcs, arc)
}

// miterTangents returns this edge's corner tangents and the seam oriented ta→tb: the shared
// face carries the seam's top (sTop, on the shared face), the outer face carries its bottom
// (sBot, on the now-shortened sharp edge). The seam is the SAME point list for both edges of
// the miter — reversed for the edge whose A face is the outer one — so the two cylinders weld
// along it watertight.
func miterTangents(in cornerInputs, m *cornerMiter) (ta, tb math.Point3, seam []math.Point3) {
	if in.a == m.shared {
		return m.seam[0], m.sBot, m.seam
	}
	return m.sBot, m.seam[0], reversePoints(m.seam)
}

// reversePoints returns a reversed copy of pts.
func reversePoints(pts []math.Point3) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}

// perpToward returns the unit direction from cen toward p projected into the plane
// perpendicular to axis — the in-cross-section direction to the rounded corner.
func perpToward(cen, p math.Point3, axis math.Vector3) math.Vector3 {
	d := cen.VectorTo(p)
	perp := d.Sub(axis.Scale(d.Dot(axis)))
	u, err := math.UnitVector3FromVector(perp)
	if err != nil {
		return d
	}
	return u.AsVector()
}

// endFaceAt returns the face meeting at v that is neither a nor b (the end cap the fillet
// rounds), or nil if there is none.
func endFaceAt(v *topo.Vertex, a, b *topo.Face) *topo.Face {
	for _, e := range v.Edges() {
		for _, f := range e.Faces() {
			if f != a && f != b {
				return f
			}
		}
	}
	return nil
}

// blendArc is one boundary arc of a corner sphere patch (shared with a cylinder fillet). chords is
// nil for an analytic single arc (a constant cylinder), or the chord polyline ta…tb when the arc is
// shared with a VARIABLE cone, whose faceted end must match the patch edge-for-edge to stay watertight.
type blendArc struct {
	ta, tb, mid math.Point3
	chords      []math.Point3
}

// cornerBlend is a spherical corner patch where several filleted edges meet at one vertex:
// the rolling-ball sphere tangent to the corner's faces, its tangent point on each face
// (keyed by face id), and the arcs (filled in as the edges are solved) that bound the patch.
type cornerBlend struct {
	vertex *topo.Vertex
	center math.Point3
	sphere geom.Sphere
	tan    map[uint64]math.Point3
	arcs   []blendArc
}

// computeCorners finds the shared corners of the filleted edge set and solves a corner
// treatment for each, keyed by corner vertex id:
//
//   - three filleted edges at a trihedral (3-face) vertex → a spherical corner patch (blend);
//   - two filleted edges that share a face, the third edge staying sharp → a miter seam where
//     the two rolling-ball cylinders mutually trim (miter).
//
// All edges meeting at a corner must use ONE constant radius — a variable edge's faceted end
// chords cannot meet a corner watertight, and a blend/seam has a single radius — so those and
// any other configuration error clearly.
func computeCorners(picks []filletPick) (map[uint64]*cornerBlend, map[uint64]*cornerMiter, error) {
	groups := map[uint64][]filletPick{}
	for _, p := range picks {
		groups[p.edge.StartVertex().ID()] = append(groups[p.edge.StartVertex().ID()], p)
		groups[p.edge.EndVertex().ID()] = append(groups[p.edge.EndVertex().ID()], p)
	}
	blends := map[uint64]*cornerBlend{}
	miters := map[uint64]*cornerMiter{}
	for vid, ps := range groups {
		if len(ps) < 2 {
			continue
		}
		cb, cm, err := solveCorner(vid, ps)
		if err != nil {
			return nil, nil, err
		}
		if cb != nil {
			blends[vid] = cb
		}
		if cm != nil {
			miters[vid] = cm
		}
	}
	return blends, miters, nil
}

// solveCorner solves the corner treatment at vertex vid where the picks ps meet: a sphere blend
// (3 edges, trihedral vertex) or a miter seam (2 edges sharing a face), at the corner's one shared
// radius. Exactly one of (blend, miter) is returned; any other configuration errors.
func solveCorner(vid uint64, ps []filletPick) (*cornerBlend, *cornerMiter, error) {
	r, err := cornerRadius(vid, ps)
	if err != nil {
		return nil, nil, err
	}
	v := vertexByID(edgesOf(ps), vid)
	faces := facesAtVertex(v)
	switch {
	case len(ps) == 3 && len(faces) == 3:
		cb, err := solveBlend(v, faces, r)
		return cb, nil, err
	case len(ps) == 2:
		if p := varyingPick(ps); p != nil {
			return nil, nil, fmt.Errorf("fillet: a variable-radius edge (radii %g→%g) cannot share a 2-edge miter corner (its cone has no seam with a cylinder); round the third edge for a setback instead", p.r0, p.r1)
		}
		cm, err := solveMiter(v, ps, r)
		return nil, cm, err
	default:
		return nil, nil, fmt.Errorf("fillet: corner where %d filleted edges meet a %d-face vertex is not a supported blend (need 3 edges at a trihedral vertex, or 2 edges sharing a face)", len(ps), len(faces))
	}
}

// cornerRadius returns the radius every pick carries AT the shared corner vertex vid — the radius of
// the corner sphere. A variable edge is allowed (e.g. a setback's tapered third edge) as long as its
// radius at this corner matches the others; only the far ends may differ.
func cornerRadius(vid uint64, ps []filletPick) (float64, error) {
	r := radiusAtVertex(ps[0], vid)
	for _, p := range ps {
		if rv := radiusAtVertex(p, vid); rv != r {
			return 0, fmt.Errorf("fillet: edges meeting at a shared corner must use one radius there (got %g and %g)", r, rv)
		}
	}
	return r, nil
}

// varyingPick returns the first pick whose radius varies along the edge, or nil if all are constant.
func varyingPick(ps []filletPick) *filletPick {
	for i := range ps {
		if ps[i].varying() {
			return &ps[i]
		}
	}
	return nil
}

// radiusAtVertex returns the pick's radius at the endpoint vid (r0 at the start vertex, else r1).
func radiusAtVertex(p filletPick, vid uint64) float64 {
	if p.edge.StartVertex().ID() == vid {
		return p.r0
	}
	return p.r1
}

// edgesOf projects the picks' edges.
func edgesOf(ps []filletPick) []*topo.Edge {
	out := make([]*topo.Edge, len(ps))
	for i, p := range ps {
		out[i] = p.edge
	}
	return out
}

// solveBlend builds the corner sphere from the three planar faces meeting at v (the point
// at distance r from all three, inside) and its tangent points on each.
func solveBlend(v *topo.Vertex, faces []*topo.Face, r float64) (*cornerBlend, error) {
	var a [3][3]float64
	var b [3]float64
	for i, f := range faces {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok {
			return nil, fmt.Errorf("fillet: corner face must be planar")
		}
		n := pl.Normal()
		a[i] = [3]float64{n.X, n.Y, n.Z}
		b[i] = n.Dot(pl.Origin.AsVector()) - r // distance r on the inside of each face
	}
	x, ok := solve3(a, b)
	if !ok {
		return nil, fmt.Errorf("fillet: cannot solve corner blend sphere (degenerate faces)")
	}
	s := math.P3(x[0], x[1], x[2])
	sph, err := geom.NewSphere(s, r)
	if err != nil {
		return nil, err
	}
	tan := make(map[uint64]math.Point3, 3)
	for _, f := range faces {
		tan[f.ID()] = s.TranslateBy(f.Geometry().(geom.Plane).Normal().Scale(r))
	}
	return &cornerBlend{vertex: v, center: s, sphere: sph, tan: tan}, nil
}

// vertexByID returns the vertex with id vid from the edge set.
func vertexByID(edges []*topo.Edge, vid uint64) *topo.Vertex {
	for _, e := range edges {
		if e.StartVertex().ID() == vid {
			return e.StartVertex()
		}
		if e.EndVertex().ID() == vid {
			return e.EndVertex()
		}
	}
	return nil
}

// facesAtVertex returns the distinct faces meeting at v.
func facesAtVertex(v *topo.Vertex) []*topo.Face {
	seen := map[uint64]bool{}
	var out []*topo.Face
	for _, e := range v.Edges() {
		for _, f := range e.Faces() {
			if !seen[f.ID()] {
				seen[f.ID()] = true
				out = append(out, f)
			}
		}
	}
	return out
}
