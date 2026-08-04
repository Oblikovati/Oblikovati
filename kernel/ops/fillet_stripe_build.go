// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// rebuildWithStripe reconstructs the body with the tangent stripe rounded in: the shared face and the
// walls are re-trimmed onto the blend's contact curves, the vertical smooth edge below each junction
// is split at the tube depth (its upper part consumed), and one blend face per segment is added,
// consecutive faces sharing a junction section circle. Mirrors the topological half of OCCT
// ChFi3d_Builder — the geometry came from the blend engine (fillet_stripe.go). See ADR-0050 P4b.
func rebuildWithStripe(b *topo.Body, st *tangentStripe) (*topo.Body, error) {
	g := &stripeBuild{
		st: st, bld: topo.NewBuilder(b.IsSolid(), b.Lineage()),
		verts: map[*topo.Vertex]*topo.Vertex{}, edges: map[*topo.Edge]*topo.Edge{},
		chainIdx: map[*topo.Edge]int{}, downIdx: map[*topo.Edge]int{},
		junIdx:    map[*topo.Vertex]int{},
		weld:      float64(geom.ResolutionForBox(b.RangeBox()).Weld()),
		sharedFwd: make([]bool, len(st.segs)),
	}
	g.index()
	g.copySurvivingVertices(b)
	if err := g.addNewVertsAndEdges(); err != nil {
		return nil, err
	}
	g.copySurvivingEdges(b)
	for _, f := range b.Faces() {
		g.copyFace(f)
	}
	g.addBlendFaces()
	if !st.closed {
		g.addCapFaces()
	}
	return g.bld.Build(), nil
}

// copySurvivingVertices carries over every original vertex except the interior junction corners the
// stripe consumes (replaced by section feet). Open-run terminal corners survive and are copied here.
func (g *stripeBuild) copySurvivingVertices(b *topo.Body) {
	for _, v := range b.Vertices() {
		if _, gone := g.junIdx[v]; !gone {
			g.verts[v] = g.bld.AddVertex(v.Point(), v.Lineage())
		}
	}
}

// copySurvivingEdges carries over every original edge except the chain guides (replaced by contacts) and
// the split vertical edges (replaced by lower remnants); both are re-trimmed by copyFace instead.
func (g *stripeBuild) copySurvivingEdges(b *topo.Body) {
	for _, e := range b.Edges() {
		if _, chain := g.chainIdx[e]; chain {
			continue
		}
		if _, down := g.downIdx[e]; down {
			continue
		}
		g.edges[e] = g.bld.AddEdge(e.Geometry(), g.verts[e.StartVertex()], g.verts[e.EndVertex()], e.Lineage())
	}
}

type stripeBuild struct {
	st       *tangentStripe
	bld      *topo.Builder
	verts    map[*topo.Vertex]*topo.Vertex
	edges    map[*topo.Edge]*topo.Edge
	chainIdx map[*topo.Edge]int // chain edge → segment index
	downIdx  map[*topo.Edge]int // vertical smooth edge → junction index
	junIdx   map[*topo.Vertex]int
	vS1, vW  []*topo.Vertex // per segment ENTRY: shared-side foot, wall-side foot
	topE     []*topo.Edge   // per segment: shared-face contact
	wallE    []*topo.Edge   // per segment: wall contact
	section  []*topo.Edge   // per interior junction: tube section circle vS1[j]→vW[j]
	lowerE   []*topo.Edge   // per interior junction: surviving lower part of the split vertical edge
	// open-run terminals only: the last segment's EXIT feet, the two flat cap arcs, and the connectors
	// folding each cap back to its surviving corner vertex (on the shared face and the wall).
	vEndS1, vEndW *topo.Vertex
	cap           [2]*topo.Edge // [0]=start terminal cap arc, [1]=end terminal cap arc
	connTop       [2]*topo.Edge // corner vertex → shared-face foot
	connWall      [2]*topo.Edge // corner vertex → wall foot
	weld          float64       // model-relative coincidence tolerance (descending-edge remnant guard)
	// sharedFwd[i]: the SHARED face's retrim walks segment i's contact entry→exit. Recorded while
	// copying the shared face so the blend face can take the topologically OPPOSITE direction —
	// edge-use pairing is decided by topology, and only the face's Reversed flag by geometry.
	sharedFwd []bool
}

// index builds the lookup maps from the solved stripe: chain edges → segment, down edges → junction,
// junction vertices → index. An open run's terminal slots are nil (its ends survive, not consumed) and
// are skipped, so a terminal vertex is copied verbatim rather than replaced by section feet.
func (g *stripeBuild) index() {
	for i, e := range g.st.edges {
		g.chainIdx[e] = i
	}
	for j, d := range g.st.down {
		if d == nil { // an open-run terminal: nothing consumed here
			continue
		}
		g.downIdx[d] = j
		g.junIdx[g.st.junction[j]] = j
	}
}

// addNewVertsAndEdges creates the stripe's new vertices (the shared-side and wall-side section feet at
// every segment entry, plus an open run's terminal-exit feet) and its new edges (the two contacts per
// segment, the section circle + lower split-edge remnant per interior junction, and — for an open run —
// the two flat cap arcs and their corner connectors).
func (g *stripeBuild) addNewVertsAndEdges() error {
	n := len(g.st.segs)
	lin := func(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("stripe", role, i)) }
	g.vS1, g.vW = make([]*topo.Vertex, n), make([]*topo.Vertex, n)
	for j := 0; j < n; j++ {
		g.vS1[j] = g.bld.AddVertex(g.st.segs[j].topA, lin("s1", j))
		g.vW[j] = g.bld.AddVertex(g.st.segs[j].wallA, lin("w", j))
	}
	if !g.st.closed {
		g.vEndS1 = g.bld.AddVertex(g.st.segs[n-1].topB, lin("s1", n))
		g.vEndW = g.bld.AddVertex(g.st.segs[n-1].wallB, lin("w", n))
	}
	g.topE, g.wallE = make([]*topo.Edge, n), make([]*topo.Edge, n)
	for i := 0; i < n; i++ {
		g.topE[i] = g.bld.AddEdge(g.st.segs[i].topContact, g.vS1[i], g.exitFootS1(i), lin("top", i))
		g.wallE[i] = g.bld.AddEdge(g.st.segs[i].wallContact, g.vW[i], g.exitFootW(i), lin("wall", i))
	}
	if err := g.addSectionsAndRemnants(lin); err != nil {
		return err
	}
	return g.addTerminalEdges(lin)
}

// addSectionsAndRemnants adds, per INTERIOR junction, the tube section circle two blend faces share and
// the surviving lower part of the split vertical smooth edge. An open run's terminal slots are nil (no
// section circle, no split there) and are skipped — their flat cap arcs are added in addTerminalEdges.
func (g *stripeBuild) addSectionsAndRemnants(lin func(string, int) topo.Lineage) error {
	n := len(g.st.segs)
	g.section, g.lowerE = make([]*topo.Edge, n), make([]*topo.Edge, n)
	for j := 0; j < n; j++ {
		if g.st.junction[j] == nil { // an open-run terminal
			continue
		}
		arc, err := geom.Arc3dByThreePoints(g.st.segs[j].topA, g.st.apex[j], g.st.segs[j].wallA)
		if err != nil {
			return fmt.Errorf("fillet: cannot build the section circle at stripe junction %d: %w", j, err)
		}
		g.section[j] = g.bld.AddEdge(arc, g.vS1[j], g.vW[j], lin("sec", j))
		if err := g.addDownRemnant(j, lin); err != nil {
			return err
		}
	}
	return nil
}

// addDownRemnant adds the surviving remnant of junction j's descending edge, cut at the crossing
// foot on whichever side (shared or wall) the edge lies. The remnant is rebuilt as a line, so a
// curved descending edge whose remnant would leave that line is refused rather than misdrawn.
func (g *stripeBuild) addDownRemnant(j int, lin func(string, int) topo.Lineage) error {
	cutPt, cutV := g.st.segs[j].wallA, g.vW[j]
	if g.st.cutOnShared[j] {
		cutPt, cutV = g.st.segs[j].topA, g.vS1[j]
	}
	bottom := otherVertex(g.st.down[j], g.st.junction[j])
	if err := g.assertStraightRemnant(j, cutPt, bottom.Point()); err != nil {
		return err
	}
	g.lowerE[j] = g.bld.AddEdge(geom.NewLineSegment(cutPt, bottom.Point()), cutV, g.verts[bottom], lin("lower", j))
	return nil
}

// assertStraightRemnant verifies the descending edge's curve actually runs along the straight
// remnant chord (its curve point midway between the cut and the far end lies on the chord) —
// the honest refusal for a curved descending edge the line rebuild would misrepresent.
func (g *stripeBuild) assertStraightRemnant(j int, cutPt, bottom math.Point3) error {
	d := g.st.down[j].Geometry()
	tCut, _ := geom.CurveParamAtPoint3(d, cutPt)
	tBot, _ := geom.CurveParamAtPoint3(d, bottom)
	mid := d.PointAt((tCut + tBot) / 2)
	chordMid := centroidPts([]math.Point3{cutPt, bottom})
	if float64(mid.DistanceTo(chordMid)) > 10*g.weld {
		return fmt.Errorf("fillet: stripe junction %d: descending edge is curved (chord gap %.3g) — "+
			"its remnant cannot be rebuilt as a line", j, float64(mid.DistanceTo(chordMid)))
	}
	return nil
}

// addTerminalEdges builds an open run's two flat cap arcs (shared foot → apex → wall foot) and the two
// corner connectors per terminal — the lines folding the cap back to its surviving vertex on the shared
// face and on the wall. A closed loop has no terminals, so this is a no-op.
func (g *stripeBuild) addTerminalEdges(lin func(string, int) topo.Lineage) error {
	if g.st.closed {
		return nil
	}
	feet := [2][2]*topo.Vertex{{g.vS1[0], g.vW[0]}, {g.vEndS1, g.vEndW}}
	for t := 0; t < 2; t++ {
		tm := g.st.term[t]
		arc, err := geom.Arc3dByThreePoints(tm.topA, tm.apex, tm.wallA)
		if err != nil {
			return fmt.Errorf("fillet: cannot build the flat cap arc at open stripe terminal %d: %w", t, err)
		}
		g.cap[t] = g.bld.AddEdge(arc, feet[t][0], feet[t][1], lin("cap", t))
		vtx := g.verts[tm.vertex]
		g.connTop[t] = g.bld.AddEdge(geom.NewLineSegment(tm.vertex.Point(), tm.topA), vtx, feet[t][0], lin("ctop", t))
		g.connWall[t] = g.bld.AddEdge(geom.NewLineSegment(tm.vertex.Point(), tm.wallA), vtx, feet[t][1], lin("cwall", t))
	}
	return nil
}

// exitFootS1 / exitFootW return segment i's EXIT feet on the shared face / wall: an interior segment
// exits into the next segment's entry feet; an open run's last segment exits into its terminal feet.
func (g *stripeBuild) exitFootS1(i int) *topo.Vertex {
	if n := len(g.st.segs); !g.st.closed && i == n-1 {
		return g.vEndS1
	}
	return g.vS1[(i+1)%len(g.st.segs)]
}

func (g *stripeBuild) exitFootW(i int) *topo.Vertex {
	if n := len(g.st.segs); !g.st.closed && i == n-1 {
		return g.vEndW
	}
	return g.vW[(i+1)%len(g.st.segs)]
}

// copyFace copies a face, re-trimming the shared face (chain edge → shared contact) and the walls
// (chain edge → wall contact, vertical edge → lower remnant); others copied verbatim. A chain-edge use
// may expand to several new edges (an open-run terminal inserts a corner connector), so mapUse returns
// an ordered list.
func (g *stripeBuild) copyFace(f *topo.Face) {
	specs := make([]topo.LoopSpec, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		uses := make([]topo.Use, 0, len(l.EdgeUses()))
		for _, u := range l.EdgeUses() {
			uses = append(uses, g.mapUse(f, u)...)
		}
		if l.IsOuter() {
			specs = append(specs, topo.OuterLoop(uses...))
		} else {
			specs = append(specs, topo.InnerLoop(uses...))
		}
	}
	if f.Reversed() {
		g.bld.AddReversedFace(f.Geometry(), f.Lineage(), specs...)
		return
	}
	g.bld.AddFace(f.Geometry(), f.Lineage(), specs...)
}

// mapUse substitutes a removed edge's use with its stripe replacement(s), orienting each new edge so the
// path runs the same way the original use did. A chain edge maps to its contact (plus a terminal corner
// connector at an open-run end); a split vertical edge maps to its lower remnant; anything else copies.
func (g *stripeBuild) mapUse(f *topo.Face, u *topo.EdgeUse) []topo.Use {
	if i, ok := g.chainIdx[u.Edge()]; ok {
		return g.mapChainUse(f, u, i)
	}
	from := useFromVertex(u)
	if j, ok := g.downIdx[u.Edge()]; ok {
		fromV := g.verts[from]
		if from == g.st.junction[j] {
			fromV = g.vW[j]
			if g.st.cutOnShared[j] {
				fromV = g.vS1[j] // the descending edge lies on the shared face — cut at the shared foot
			}
		}
		return []topo.Use{dirUse(g.lowerE[j], fromV)}
	}
	return []topo.Use{{Edge: g.edges[u.Edge()], Reversed: u.Reversed()}}
}

// mapChainUse replaces a chain edge's use on the shared face or a wall with its re-trimmed contact,
// inserting the flat cap's corner connector when this segment's entry (i==0) or exit (i==n−1) is an
// open-run terminal. The pieces are built entry→exit and emitted reversed when the use runs exit→entry.
func (g *stripeBuild) mapChainUse(f *topo.Face, u *topo.EdgeUse, i int) []topo.Use {
	contact, entryFoot, exitFoot := g.topE[i], g.vS1[i], g.exitFootS1(i)
	startConn, endConn := g.connTop[0], g.connTop[1]
	if f != g.st.shared {
		contact, entryFoot, exitFoot = g.wallE[i], g.vW[i], g.exitFootW(i)
		startConn, endConn = g.connWall[0], g.connWall[1]
	}
	fwd := []dirPiece{{contact, entryFoot, exitFoot}}
	n := len(g.st.segs)
	if !g.st.closed && i == 0 {
		fwd = append([]dirPiece{{startConn, g.verts[g.st.term[0].vertex], entryFoot}}, fwd...)
	}
	if !g.st.closed && i == n-1 {
		fwd = append(fwd, dirPiece{endConn, exitFoot, g.verts[g.st.term[1].vertex]})
	}
	forward := useFromVertex(u) == g.entryVertex(i)
	if f == g.st.shared {
		g.sharedFwd[i] = forward // the blend face must walk this contact the OPPOSITE way
	}
	if forward {
		return forwardPieces(fwd)
	}
	return reversePieces(fwd)
}

// entryVertex is the original body vertex at segment i's entry (spine-first) end — an interior junction,
// or the open run's start terminal for the first segment.
func (g *stripeBuild) entryVertex(i int) *topo.Vertex {
	if !g.st.closed && i == 0 {
		return g.st.term[0].vertex
	}
	return g.st.junction[i]
}

// dirPiece is one directed edge of a chain-use replacement path, a→b in the segment's entry→exit sense.
type dirPiece struct {
	e    *topo.Edge
	a, b *topo.Vertex
}

// forwardPieces emits the path entry→exit (each piece used from its a vertex).
func forwardPieces(fwd []dirPiece) []topo.Use {
	out := make([]topo.Use, len(fwd))
	for i, p := range fwd {
		out[i] = dirUse(p.e, p.a)
	}
	return out
}

// reversePieces emits the path exit→entry (order reversed, each piece used from its b vertex).
func reversePieces(fwd []dirPiece) []topo.Use {
	out := make([]topo.Use, len(fwd))
	for i, p := range fwd {
		out[len(fwd)-1-i] = dirUse(p.e, p.b)
	}
	return out
}

// dirUse returns the use of edge e that starts at from (reversed when from is e's end vertex).
func dirUse(e *topo.Edge, from *topo.Vertex) topo.Use {
	return topo.Use{Edge: e, Reversed: e.StartVertex() != from}
}

// addBlendFaces adds one blend face per segment: the quad topE[i] · section · wallE[i] · section. The
// LOOP DIRECTION is decided topologically — the blend walks the shared-face contact OPPOSITE to the
// shared face's own retrimmed use of it (sharedFwd), which forces correct edge-use pairing all around
// the quad (the wall contact and the junction sections then pair with THEIR other users by the same
// global consistency). Only the face's Reversed flag comes from geometry: when the forced loop winds
// against the blend surface's normal, the face is ADDED REVERSED rather than the loop re-wound — the
// old geometric re-winding kept the surface outward but silently broke pairing whenever the shared
// face sat on the other side of the tube (simple/Y9, shared = the drum wall). An interior boundary is
// a junction section circle; an open run's two ends are terminal cap arcs (see entry/exitSecEdge).
func (g *stripeBuild) addBlendFaces() {
	n := len(g.st.segs)
	lin := func(i int) topo.Lineage { return topo.NewLineage(topo.Tok("stripe", "blend", i)) }
	for i := 0; i < n; i++ {
		loop, ring := g.blendLoop(i)
		if blendRingFlipped(g.st.segs[i].surf, ring) {
			g.bld.AddReversedFace(g.st.segs[i].surf, lin(i), topo.OuterLoop(loop...))
			continue
		}
		g.bld.AddFace(g.st.segs[i].surf, lin(i), topo.OuterLoop(loop...))
	}
}

// blendLoop is segment i's blend-face boundary walked opposite to the shared face's use of the
// contact, with the ring corners in the same traversal order (for the Reversed-flag winding test).
func (g *stripeBuild) blendLoop(i int) ([]topo.Use, []math.Point3) {
	es1, ew := g.exitFootS1(i), g.exitFootW(i)
	if g.sharedFwd[i] {
		loop := []topo.Use{
			dirUse(g.topE[i], es1),
			dirUse(g.entrySecEdge(i), g.vS1[i]),
			dirUse(g.wallE[i], g.vW[i]),
			dirUse(g.exitSecEdge(i), ew),
		}
		return loop, []math.Point3{es1.Point(), g.vS1[i].Point(), g.vW[i].Point(), ew.Point()}
	}
	loop := []topo.Use{
		dirUse(g.topE[i], g.vS1[i]),
		dirUse(g.exitSecEdge(i), es1),
		dirUse(g.wallE[i], ew),
		dirUse(g.entrySecEdge(i), g.vW[i]),
	}
	return loop, []math.Point3{g.vS1[i].Point(), es1.Point(), ew.Point(), g.vW[i].Point()}
}

// entrySecEdge / exitSecEdge return the section arc bounding segment i at its entry / exit — an interior
// junction's section circle, or (for an open run's first/last segment) the flat terminal cap arc.
func (g *stripeBuild) entrySecEdge(i int) *topo.Edge {
	if !g.st.closed && i == 0 {
		return g.cap[0]
	}
	return g.section[i]
}

func (g *stripeBuild) exitSecEdge(i int) *topo.Edge {
	if n := len(g.st.segs); !g.st.closed && i == n-1 {
		return g.cap[1]
	}
	return g.section[(i+1)%len(g.st.segs)]
}

// blendRingFlipped reports whether the quad ring winds against the blend surface's outward normal —
// the triangle from three ring corners has a normal opposing surf.NormalAt at the ring centroid.
func blendRingFlipped(surf geom.Surface, ring []math.Point3) bool {
	nrm := ring[0].VectorTo(ring[1]).Cross(ring[0].VectorTo(ring[2]))
	u, v := surf.ParamAt(centroidPts(ring))
	return float64(nrm.Dot(surf.NormalAt(u, v))) < 0
}

// filletTangentStripe rounds a closed tangent chain (mixed straight/arc segments sharing one face) as
// one continuous blend stripe — the #1797 top-perimeter case the per-edge miter path could not build.
func filletTangentStripe(body *topo.Body, edges []*topo.Edge, closed bool, r float64) (*topo.Body, error) {
	st, err := solveTangentStripe(body, edges, closed, r)
	if err != nil {
		return nil, err
	}
	res, err := rebuildWithStripe(body, st)
	if err != nil {
		return nil, err
	}
	if rep := Validate(res); !rep.Valid || !res.IsSolid() {
		return nil, fmt.Errorf("fillet: tangent-stripe result is not a valid solid %v", rep.Issues)
	}
	return res, nil
}
