// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/blend"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Tangent-stripe fillet (ADR-0050 P4b, #1797) — filleting a tangent CHAIN whose segments do not all
// share the same support pair, e.g. the top perimeter of a box whose vertical edges are already
// rounded: a closed loop of 4 straight top∩side edges (plane∩plane → cylinder blend) and 4 arc
// top∩cylinder edges (plane∩cylinder → torus blend). The whole loop shares ONE face (the top plane),
// so it builds as a single continuous stripe: the blend engine (blend.Marcher) supplies every segment
// surface + its two support contacts, consecutive segments meet along a shared section circle (they
// sample the same spine abscissa, so their end sections coincide to machine precision — a G1 junction,
// not a miter), and the stripe is stitched here. Mirrors OCCT ChFi3d_Builder over a ChFiDS_Spine.

// stripeSeg is one solved segment of the stripe: its blend surface and the two support contacts
// (shared side = the common face, wall side = the segment's other support), with the section
// endpoints the marcher trimmed. topA/wallA are the segment's entry section feet, ...B the exit.
type stripeSeg struct {
	wall         *topo.Face
	surf         geom.Surface
	topContact   geom.Curve3 // on the shared face, topA → topB
	wallContact  geom.Curve3 // on the wall, wallA → wallB
	topA, topB   math.Point3
	wallA, wallB math.Point3
}

// tangentStripe is a solved continuous blend over a closed tangent chain sharing one face.
type tangentStripe struct {
	shared   *topo.Face
	r        float64
	closed   bool           // a closed loop (no terminals) vs an open run (two flat setback caps)
	edges    []*topo.Edge   // the chain edges, ordered; edges[i] is segment i's guide
	segs     []stripeSeg    // one per chain edge
	apex     []math.Point3  // apex[j] = tube apex of the section circle at junction j (entry of seg j)
	junction []*topo.Vertex // junction[j] = the original top vertex where segs[j-1] meets segs[j]; nil at an open terminal
	down     []*topo.Edge   // down[j] = the descending smooth edge consumed below junction[j]; nil at a terminal
	// cutOnShared[j]: down[j] lies on the SHARED face, so its remnant reattaches at the shared-side
	// foot (vS1); false = the wall side (vW). Resolved with the crossing station (simple/Y9).
	cutOnShared []bool
	term        [2]stripeTerm // (open only) the two run-out terminals: [0]=entry of seg 0, [1]=exit of the last seg
}

// stripeTerm is one run-out end of an OPEN stripe — the tube's flat setback cap. The cap is the quarter
// disk bounded by the end section arc (topA→apex→wallA) and the two lines back to the surviving corner
// vertex; it lies in the section plane ⊥ the spine, so it never intrudes on the neighbouring geometry.
// Mirrors OCCT's free-end ChFi3d_CoupeParPlan planar cut and our own arc-fillet setback end-cap.
type stripeTerm struct {
	vertex      *topo.Vertex // the original corner vertex the tube ends at (survives; the cap's apex-of-triangle)
	apex        math.Point3  // the section arc's exposed midpoint
	topA, wallA math.Point3  // the section feet on the shared face and the wall
	seg         int          // the segment this terminal ends
}

// curvedTangentChain reports whether the picks are a constant-radius tangent chain at least one of
// whose edges borders a curved (cylinder) face, returning the ordered edges, the radius, and whether
// the run closes a loop (closed=true) or is a single contiguous open sub-run (closed=false). ok=false
// leaves the selection to the existing paths (an all-planar chain, a non-uniform radius, or a gapped /
// scattered subset) — so nothing there regresses. The caller routes both a closed loop and a
// contiguous open run to the stripe assembler; the open run terminates in flat setback caps.
func curvedTangentChain(body *topo.Body, picks []EdgeFilletRadii) (edges []*topo.Edge, r float64, closed, ok bool) {
	if len(picks) < 2 || !uniformConstRadius(picks) {
		return nil, 0, false, false
	}
	maxKeys, isClosed, err := TangentEdgeChain(body, picks[0].Key, DefaultTangentChainAngle)
	if err != nil || !picksWithinChain(picks, maxKeys) || !anyCurvedAdjacent(body, picks) {
		return nil, 0, false, false
	}
	edges, closed, ok = orderPicksOnChain(body, picks, maxKeys, isClosed)
	if !ok {
		return nil, 0, false, false
	}
	return edges, picks[0].R0, closed, true
}

// orderPicksOnChain orders the picked edges along the maximal tangent chain and reports whether they
// are the whole closed loop (closed=true) or a single CONTIGUOUS open run (closed=false). maxKeys is
// already chain-ordered (consecutive keys share a vertex), so a valid selection is either every key
// (closed loop) or one unbroken stretch of them; a gapped or scattered subset returns ok=false.
func orderPicksOnChain(body *topo.Body, picks []EdgeFilletRadii, maxKeys [][]byte, isClosed bool) (edges []*topo.Edge, closed, ok bool) {
	if isClosed && len(picks) == len(maxKeys) {
		return edgesForKeys(body, maxKeys), true, true
	}
	picked := make(map[string]bool, len(picks))
	for _, p := range picks {
		picked[string(p.Key)] = true
	}
	run, runOk := contiguousRun(maxKeys, picked, isClosed)
	if !runOk {
		return nil, false, false
	}
	return edgesForKeys(body, run), false, true
}

// contiguousRun returns the picked keys as one unbroken stretch of the chain order, or ok=false when
// the picks break into more than one run. On a closed chain the single run may wrap the seam (a
// prefix + suffix that are adjacent through the loop closure); on an open chain no wrap is allowed.
func contiguousRun(maxKeys [][]byte, picked map[string]bool, isClosed bool) ([][]byte, bool) {
	var idx []int
	for i, k := range maxKeys {
		if picked[string(k)] {
			idx = append(idx, i)
		}
	}
	if len(idx) == 0 {
		return nil, false
	}
	gapAt, gaps := 0, 0
	for i := 1; i < len(idx); i++ {
		if idx[i] != idx[i-1]+1 {
			gaps, gapAt = gaps+1, i
		}
	}
	if gaps == 0 {
		return keysAtIndices(maxKeys, idx), true // one linear stretch
	}
	if gaps == 1 && isClosed {
		return keysAtIndices(maxKeys, append(idx[gapAt:], idx[:gapAt]...)), true // wraps the seam
	}
	return nil, false
}

// keysAtIndices gathers maxKeys at the given (chain-ordered) positions.
func keysAtIndices(maxKeys [][]byte, idx []int) [][]byte {
	out := make([][]byte, len(idx))
	for i, j := range idx {
		out[i] = maxKeys[j]
	}
	return out
}

// edgesForKeys resolves chain keys to their edges in body order.
func edgesForKeys(body *topo.Body, keys [][]byte) []*topo.Edge {
	es := make([]*topo.Edge, len(keys))
	for i, k := range keys {
		es[i], _ = body.FindEdgeByKey(k)
	}
	return es
}

// picksWithinChain reports whether every pick key lies on the maximal tangent chain — i.e. the picks
// are one tangent run (or a contiguous part of one), not an unrelated scatter of edges.
func picksWithinChain(picks []EdgeFilletRadii, chainKeys [][]byte) bool {
	on := make(map[string]bool, len(chainKeys))
	for _, k := range chainKeys {
		on[string(k)] = true
	}
	for _, p := range picks {
		if !on[string(p.Key)] {
			return false
		}
	}
	return true
}

// anyCurvedAdjacent reports whether any pick borders a curved (cylinder) face — the trigger that
// distinguishes a stripe/partial-result case from the all-planar chains the existing path handles.
func anyCurvedAdjacent(body *topo.Body, picks []EdgeFilletRadii) bool {
	for _, p := range picks {
		if e, ok := body.FindEdgeByKey(p.Key); ok {
			if _, _, isCP := cylinderPlaneEdge(e); isCP {
				return true
			}
		}
	}
	return false
}

// uniformConstRadius reports whether every pick is the same constant radius with a plain arc section
// (no per-end taper, intermediate point, or conic/G2 cross-section) — the stripe's constant-r regime.
func uniformConstRadius(picks []EdgeFilletRadii) bool {
	r := picks[0].R0
	for _, p := range picks {
		if p.R0 != p.R1 || p.R0 != r || len(p.Mids) > 0 || !p.Cross.IsArc() {
			return false
		}
	}
	return true
}

// solveTangentStripe drives the blend engine over the chain and assembles the per-segment contacts,
// the interior junction section apices, and (for an open run) the two flat setback terminals. It errors
// clearly on the cases it does not cover (no common face) rather than producing a wrong body — OCCT's
// localized partial-result contract.
func solveTangentStripe(body *topo.Body, edges []*topo.Edge, closed bool, r float64) (*tangentStripe, error) {
	shared := commonFaceOfAll(edges)
	if shared == nil {
		return nil, fmt.Errorf("fillet: tangent-chain segments do not share a common face (general stripes are future work)")
	}
	sp, err := blend.NewSpine(edges, closed)
	if err != nil {
		return nil, err
	}
	m := &blend.Marcher{Inside: func(p math.Point3) bool { return PointInsideBody(body, p) },
		Res: geom.ResolutionForBox(body.RangeBox())}
	st := &tangentStripe{shared: shared, r: r, closed: closed, edges: edges}
	if err := st.marchSegments(sp, m, r); err != nil {
		return nil, err
	}
	if err := st.solveApices(sp, m, r); err != nil {
		return nil, err
	}
	if err := st.solveJunctions(); err != nil {
		return nil, err
	}
	if err := st.resolveJunctionCrossings(sp, m); err != nil {
		return nil, err
	}
	st.reseatSurfaces()
	return st, nil
}

// marchSegments runs the blend engine over the spine and maps each returned segment onto the shared
// face / wall roles, filling st.segs. On a partial-result status it localizes the failure to the segment
// the marcher stopped on and the guide point there (OCCT's per-SurfData ChFiDS_ErrorStatus contract),
// so the caller can name the faulty entity rather than fail globally.
func (st *tangentStripe) marchSegments(sp *blend.Spine, m *blend.Marcher, r float64) error {
	run := m.March(sp, blend.ConstRadiusFillet{R: r})
	if run.Status != blend.StatusOk {
		return fmt.Errorf("fillet: blend %v at chain segment %d (near %s) — radius %g exceeds the local bound there",
			run.Status, len(run.Segments), spineFailPoint(sp, len(run.Segments)), r)
	}
	for _, bs := range run.Segments {
		seg, err := stripeSegOf(bs, st.shared)
		if err != nil {
			return err
		}
		st.segs = append(st.segs, seg)
	}
	return nil
}

// spineFailPoint describes the guide location of the segment the marcher stopped on, for a localized
// error message. It clamps to the last edge when the failure is at the run's very end.
func spineFailPoint(sp *blend.Spine, seg int) string {
	if seg >= sp.NbEdges() {
		seg = sp.NbEdges() - 1
	}
	first, last := sp.EdgeSpineRange(seg)
	p := sp.PointAt((first + last) / 2)
	return fmt.Sprintf("(%.3f, %.3f, %.3f)", p.X, p.Y, p.Z)
}

// reseatSurfaces re-anchors each blend surface's angular reference so its EXPOSED patch sits in the
// middle of the periodic (u) domain, never straddling the 0/2π seam. The marcher builds the surface
// with an arbitrary Ref; a seam-crossing trim loop defeats the curved-face tessellator (which meshes
// the whole band instead of the exposed quarter — a documented tessellation limitation), so the fillet
// would read as material on the wrong side. This mirrors the arc-fillet catalog's Ref choice.
func (st *tangentStripe) reseatSurfaces() {
	for i := range st.segs {
		st.segs[i].surf = reseatBlendSurface(st.segs[i].surf, st.apex[i], st.segs[i].wallA)
	}
}

// reseatBlendSurface rebuilds a cylinder/torus blend with its angular origin placed so the exposed
// patch is centred away from the seam: a cylinder gets Ref opposite the apex (apex at u=π); a torus
// gets Ref at the segment's entry radial (its 90° major arc runs u∈[0,π/2]). Non-primitive surfaces
// (a fitted B-spline) are returned unchanged — their trim is metric, not periodic.
func reseatBlendSurface(surf geom.Surface, apex, entryWall math.Point3) geom.Surface {
	switch s := surf.(type) {
	case geom.Cylinder:
		// Place the exposed apex at u=±π/2 (mid-quadrant): with ParamAt ranging (−π,π] BOTH 0 and π are
		// seams, so Ref must be PERPENDICULAR to the apex radial (= axis × apexDir) to keep the 90° exposed
		// arc, u∈[π/4,3π/4], clear of both.
		apexRadial := perpComponent(s.Origin.VectorTo(apex), s.AxisDir)
		ref := apexRadial.Cross(s.AxisDir.AsVector()) // sign puts the apex at u=+π/2, the exposed arc in [π/4,3π/4]
		if c, err := geom.NewCylinderWithRef(s.Origin, s.AxisDir.AsVector(), ref, s.Radius); err == nil {
			return c
		}
	case geom.Torus:
		ref := perpComponent(s.Center.VectorTo(entryWall), s.AxisDir)
		if t, err := geom.NewTorusWithRef(s.Center, s.AxisDir.AsVector(), ref, s.MajorRadius, s.MinorRadius); err == nil {
			return t
		}
	}
	return surf
}

// solveJunctions records, per INTERIOR junction, the original top vertex two consecutive chain edges
// share and the single vertical smooth edge descending from it — the edge the stripe splits at depth r
// (its upper part consumed by the blend, its lower part surviving). A closed loop has n such junctions
// (the wrap included); an open run has n−1 (its two ends are terminals, resolved separately).
func (st *tangentStripe) solveJunctions() error {
	n := len(st.edges)
	st.junction = make([]*topo.Vertex, n)
	st.down = make([]*topo.Edge, n)
	first := 0
	if !st.closed {
		if err := st.solveTerminalVertices(); err != nil {
			return err
		}
		first = 1 // seg 0's entry is a terminal, not a junction
	}
	for j := first; j < n; j++ {
		prev := st.edges[(j-1+n)%n]
		v := sharedVertex(prev, st.edges[j])
		if v == nil {
			return fmt.Errorf("fillet: stripe chain edges %d and %d do not meet at a vertex", (j-1+n)%n, j)
		}
		d := descendingEdge(v, prev, st.edges[j])
		if d == nil {
			return fmt.Errorf("fillet: stripe junction %d has no single descending edge to split", j)
		}
		st.junction[j], st.down[j] = v, d
	}
	return nil
}

// solveTerminalVertices sets the surviving corner vertex at each end of an open run: the guide edge's
// endpoint NOT shared with its inner neighbour (the run-out corner the flat cap folds back to).
func (st *tangentStripe) solveTerminalVertices() error {
	n := len(st.edges)
	v0 := freeEndVertex(st.edges[0], st.edges[1])
	vn := freeEndVertex(st.edges[n-1], st.edges[n-2])
	if v0 == nil || vn == nil {
		return fmt.Errorf("fillet: open stripe terminal has no free end vertex")
	}
	st.term[0].vertex, st.term[1].vertex = v0, vn
	return nil
}

// freeEndVertex returns e's vertex that is NOT the one it shares with neighbour nb — the run-out end.
func freeEndVertex(e, nb *topo.Edge) *topo.Vertex {
	shared := sharedVertex(e, nb)
	if e.StartVertex() != shared {
		return e.StartVertex()
	}
	return e.EndVertex()
}

// sharedVertex returns the vertex shared by two edges, or nil when they meet at none.
func sharedVertex(e1, e2 *topo.Edge) *topo.Vertex {
	for _, a := range []*topo.Vertex{e1.StartVertex(), e1.EndVertex()} {
		for _, b := range []*topo.Vertex{e2.StartVertex(), e2.EndVertex()} {
			if a == b {
				return a
			}
		}
	}
	return nil
}

// descendingEdge returns the one edge at v that is neither of the two chain edges — the vertical
// smooth tangent line the two walls share below the junction. nil unless exactly one such edge exists.
func descendingEdge(v *topo.Vertex, e1, e2 *topo.Edge) *topo.Edge {
	var found *topo.Edge
	for _, e := range v.Edges() {
		if e == e1 || e == e2 {
			continue
		}
		if found != nil {
			return nil // ambiguous
		}
		found = e
	}
	return found
}

// stripeSegOf maps a marcher BlendSegment onto a stripeSeg by identifying which support is the shared
// face (its contact runs on the top) and which is the wall.
func stripeSegOf(bs blend.BlendSegment, shared *topo.Face) (stripeSeg, error) {
	if bs.OnS1.Face == shared {
		return stripeSeg{wall: bs.OnS2.Face, surf: bs.Surface,
			topContact: bs.OnS1.Curve, wallContact: bs.OnS2.Curve,
			topA: bs.Start1.Point, topB: bs.End1.Point, wallA: bs.Start2.Point, wallB: bs.End2.Point}, nil
	}
	if bs.OnS2.Face == shared {
		return stripeSeg{wall: bs.OnS1.Face, surf: bs.Surface,
			topContact: bs.OnS2.Curve, wallContact: bs.OnS1.Curve,
			topA: bs.Start2.Point, topB: bs.End2.Point, wallA: bs.Start1.Point, wallB: bs.End1.Point}, nil
	}
	return stripeSeg{}, fmt.Errorf("fillet: a stripe segment does not border the shared face")
}

// solveApices computes each segment ENTRY's tube apex — the exposed midpoint of the section circle two
// consecutive blend faces share — via the engine's ball centre + section at the entry guide point. For
// an open run it then resolves the two run-out terminals (the entry of seg 0 and the exit of the last).
func (st *tangentStripe) solveApices(sp *blend.Spine, m *blend.Marcher, r float64) error {
	n := len(st.segs)
	st.apex = make([]math.Point3, n)
	for j := 0; j < n; j++ {
		a, ok := st.entryApex(sp, m, r, j)
		if !ok {
			return fmt.Errorf("fillet: no section at stripe junction %d", j)
		}
		st.apex[j] = a
	}
	if st.closed {
		return nil
	}
	return st.solveTerminalApices(sp, m, r)
}

// entryApex is the exposed section apex at segment j's entry (the spine's first abscissa on edge j).
func (st *tangentStripe) entryApex(sp *blend.Spine, m *blend.Marcher, r float64, j int) (math.Point3, bool) {
	first, _ := sp.EdgeSpineRange(j)
	c, ok := m.BallCentre(sp.PointAt(first), st.shared.Geometry(), st.segs[j].wall.Geometry(), r)
	if !ok {
		return math.Point3{}, false
	}
	return exposedApex(c, st.segs[j].topA, st.segs[j].wallA, r), true
}

// solveTerminalApices fills the two open-run terminals' cap geometry: terminal 0 reuses seg 0's entry
// section; terminal 1 is seg (n−1)'s EXIT section (spine's last abscissa there). The surviving vertices
// are set later in solveTerminalVertices.
func (st *tangentStripe) solveTerminalApices(sp *blend.Spine, m *blend.Marcher, r float64) error {
	n := len(st.segs)
	st.term[0].apex, st.term[0].topA, st.term[0].wallA, st.term[0].seg = st.apex[0], st.segs[0].topA, st.segs[0].wallA, 0
	_, last := sp.EdgeSpineRange(n - 1)
	c, ok := m.BallCentre(sp.PointAt(last), st.shared.Geometry(), st.segs[n-1].wall.Geometry(), r)
	if !ok {
		return fmt.Errorf("fillet: no section at the open stripe's end terminal")
	}
	st.term[1] = stripeTerm{apex: exposedApex(c, st.segs[n-1].topB, st.segs[n-1].wallB, r),
		topA: st.segs[n-1].topB, wallA: st.segs[n-1].wallB, seg: n - 1}
	return nil
}

// exposedApex is the tube apex of the section circle: the point on the arc between the two contacts
// (topA on the shared face, wallA on the wall) on the CORNER side — centre + r along the bisector of
// the two contact directions. We compute it geometrically rather than via the engine's exposed-arc
// test, because that test reads "outside the solid", and a convex edge's corner is still MATERIAL on
// the pre-fillet body being cut, so the inside-test would pick the wrong (material-side) half.
func exposedApex(centre, topA, wallA math.Point3, r float64) math.Point3 {
	u1, e1 := math.UnitVector3FromVector(centre.VectorTo(topA))
	u2, e2 := math.UnitVector3FromVector(centre.VectorTo(wallA))
	if e1 != nil || e2 != nil {
		return centre
	}
	bis, err := math.UnitVector3FromVector(u1.AsVector().Add(u2.AsVector()))
	if err != nil {
		return centre
	}
	return centre.TranslateBy(bis.AsVector().Scale(math.Scalar(r)))
}

// commonFaceOfAll returns the face bounding every edge in the run, or nil when no single face is
// shared by all of them (the tangent chain does not ride one common support).
func commonFaceOfAll(edges []*topo.Edge) *topo.Face {
	if len(edges) == 0 {
		return nil
	}
	for _, f := range edges[0].Faces() {
		if faceBoundsAll(f, edges) {
			return f
		}
	}
	return nil
}

// faceBoundsAll reports whether face f is one of the two bounding faces of every edge in the run.
func faceBoundsAll(f *topo.Face, edges []*topo.Edge) bool {
	for _, e := range edges {
		if !edgeHasFace(e, f) {
			return false
		}
	}
	return true
}

// edgeHasFace reports whether f bounds e.
func edgeHasFace(e *topo.Edge, f *topo.Face) bool {
	for _, ef := range e.Faces() {
		if ef == f {
			return true
		}
	}
	return false
}
