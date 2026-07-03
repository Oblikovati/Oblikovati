// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// FullRoundFilletDefinition is a full-round fillet (#694, Inventor FilletFullRoundSet): it replaces a
// CENTER face between two SIDE faces with a round tangent to both sides, removing the center face —
// the rounded top of a rib/wall. PARALLEL sides give a constant half-cylinder (radius = half the
// side-to-side distance); CONVERGING (non-parallel) sides give a round whose radius is the value
// tangent to both with its apex on the center plane. Built by reconstruction (a cut tool), not an
// edge fillet, since an edge fillet cannot fully consume the center face at the round radius.
type FullRoundFilletDefinition struct {
	Side1Keys  [][]byte
	CenterKeys [][]byte
	Side2Keys  [][]byte
}

// FullRoundFilletFeature rounds a center face into a round between two sides (parallel or converging).
type FullRoundFilletFeature struct{ def *FullRoundFilletDefinition }

// Definition returns the feature's definition.
func (f *FullRoundFilletFeature) Definition() *FullRoundFilletDefinition { return f.def }

// Kind names the feature type.
func (f *FullRoundFilletFeature) Kind() string { return "full-round-fillet" }

// Recompute replaces the center face with a full round between the two side faces.
func (f *FullRoundFilletFeature) Recompute(in Input) (Output, error) {
	return fullRoundFilletBody(in, f.def, "full-round-fillet")
}

// fullRoundFilletBody reconstructs the full round, dispatching on whether the sides are parallel.
// PARALLEL sides give a constant half-cylinder, cut as a corner tool (the center-face-footprint box
// minus the round cylinder). NON-PARALLEL sides converge to a virtual apex, so the round's radius is
// the value tangent to both sides with its apex on the center plane, cut as per-side sliver prisms
// (nonParallelFullRound). Non-planar faces or a degenerate frame are clean errors (the feature Sick).
func fullRoundFilletBody(in Input, def *FullRoundFilletDefinition, feat string) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	center, err := singlePlanarFace(body, def.CenterKeys, feat, "center")
	if err != nil {
		return Output{}, err
	}
	side1, err := singlePlanarFace(body, def.Side1Keys, feat, "side1")
	if err != nil {
		return Output{}, err
	}
	side2, err := singlePlanarFace(body, def.Side2Keys, feat, "side2")
	if err != nil {
		return Output{}, err
	}
	if parallelSides(side1, side2) {
		return parallelFullRound(in, body, def, center, feat)
	}
	return nonParallelFullRound(in, body, center, side1, side2, feat)
}

// parallelSides reports whether the two side faces' planes are parallel (the constant-cylinder case).
func parallelSides(side1, side2 planarFace) bool {
	return stdmath.Abs(float64(normalize(side1.Normal()).Dot(normalize(side2.Normal())))) > 0.999
}

// parallelFullRound cuts the constant half-cylinder corner tool between two parallel sides.
func parallelFullRound(in Input, body *topo.Body, def *FullRoundFilletDefinition, center planarFace, feat string) (Output, error) {
	fr, err := fullRoundFrame(body, def, center, feat)
	if err != nil {
		return Output{}, err
	}
	pc := center.face.RangeBox().Center()
	l := faceExtentAlong(center.face, pc, fr.axis) + 4*fr.radius // generous overhang so the tool cuts cleanly
	tool, err := fullRoundCornerTool(pc, fr.up, fr.sideN, fr.axis, fr.radius, l, feat, in.Diag)
	if err != nil {
		return Output{}, err
	}
	result, err := ops.BooleanWithDiagnostics(ops.Cut, planarizedDiag(body, feat, in.Diag), tool, in.Diag)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}

// convergingFrame is the geometry of a non-parallel full round: the tangent cylinder's axis point C
// and radius r, the rib axis, the in-cross-section "cross" direction, up (center normal), and the
// center-face centroid pc.
type convergingFrame struct {
	C, pc           math.Point3
	r               float64
	axis, up, cross math.Vector3
}

// nonParallelFullRound rounds the center face between two converging (non-parallel) sides (#694): it
// cuts a corner SLIVER PRISM at each side — the region between the original top corner/side and the
// round arc — so the round is tangent to both sides with its apex on the center plane (it replaces
// the center face exactly, removing the top corners). Bounding each sliver at its side's tangent line
// is what avoids undercutting the rib below the round (a plain boolean tool cannot, since the tangent
// cylinder bulges outside the slanted sides below the tangent).
func nonParallelFullRound(in Input, body *topo.Body, center, side1, side2 planarFace, feat string) (Output, error) {
	fr, err := convergingRoundFrame(center, side1, side2, feat)
	if err != nil {
		return Output{}, err
	}
	l := faceExtentAlong(center.face, fr.pc, fr.axis) + 4*fr.r // overhang so the prism spans the rib
	result := body
	for _, side := range []planarFace{side1, side2} {
		prism, perr := cornerSliverPrism(fr, l, center, side, feat)
		if perr != nil {
			return Output{}, perr
		}
		result, err = ops.BooleanWithDiagnostics(ops.Cut, planarizedDiag(result, feat, in.Diag), prism, in.Diag)
		if err != nil {
			return Output{}, fmt.Errorf("%s: cutting the corner round: %w", feat, err)
		}
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}

// convergingRoundFrame solves the tangent cylinder for non-parallel sides: a cylinder tangent to
// both side planes (axis ∥ their intersection line) whose apex — its topmost point along +up —
// lies on the center-face plane, so it replaces the center face exactly. The axis point C and radius
// r satisfy (C−o1)·n1 = (C−o2)·n2 = (C−pc)·up = −r with C in the cross-section through pc; eliminating
// r gives a 3×3 system in C, then r = −n1·(C−o1).
func convergingRoundFrame(center, side1, side2 planarFace, feat string) (convergingFrame, error) {
	up := normalize(center.Normal())
	n1, n2 := normalize(side1.Normal()), normalize(side2.Normal())
	axis := normalize(n1.Cross(n2))
	if float64(axis.Dot(axis)) < 0.5 {
		return convergingFrame{}, fmt.Errorf("%s: the side faces are parallel (no converging apex)", feat)
	}
	pc := center.face.RangeBox().Center()
	o1, o2 := side1.Origin, side2.Origin
	C, ok := linearSolve3(n1.Sub(n2), n1.Sub(up), axis,
		planeDot(n1, o1)-planeDot(n2, o2), planeDot(n1, o1)-planeDot(up, pc), planeDot(axis, pc))
	if !ok {
		return convergingFrame{}, fmt.Errorf("%s: cannot solve the round geometry (degenerate faces)", feat)
	}
	r := -float64(n1.Dot(o1.VectorTo(C)))
	if r <= 1e-9 {
		return convergingFrame{}, fmt.Errorf("%s: the sides do not form a positive round (radius %g)", feat, r)
	}
	return convergingFrame{C: C, pc: pc, r: r, axis: axis, up: up, cross: normalize(up.Cross(axis))}, nil
}

// cornerSliverPrism builds one corner of the non-parallel round as a prism in the cross-section
// (cross, up) plane through C: the sliver polygon is the original top corner, down the side to its
// tangent point (C + r·n), the round arc (chorded) back to the apex (C + r·up), then along the center
// face back to the corner — extruded ±l/2 along the rib axis.
func cornerSliverPrism(fr convergingFrame, l float64, center, side planarFace, feat string) (*topo.Body, error) {
	n := normalize(side.Normal())
	corner, ok := sharedEdgeMidpoint(center.face, side.face)
	if !ok {
		return nil, fmt.Errorf("%s: the center face does not border a side", feat)
	}
	to2 := func(p math.Point3) math.Point2 {
		d := fr.C.VectorTo(p)
		return math.P2(float64(fr.cross.Dot(d)), float64(fr.up.Dot(d)))
	}
	tangent2, apex2 := to2(fr.C.TranslateBy(n.Scale(fr.r))), to2(fr.C.TranslateBy(fr.up.Scale(fr.r)))
	poly := append([]math.Point2{to2(corner), tangent2}, roundArcChords(tangent2, apex2, fr.r)...)
	plane, err := sketch.NewPlane(fr.C, fr.cross.AsUnit(), fr.up.AsUnit())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", feat, err)
	}
	return buildPrism(poly, plane, span{near: -l / 2, far: l / 2}, 0, feat), nil
}

// roundArcChords samples the round circle (centre origin in the cross-section, radius r) from the
// tangent point to the apex the short way, returning points j=1..k (apex included, tangent not — it
// is already in the polygon). Density matches the hole/fillet facet convention.
func roundArcChords(tangent, apex math.Point2, r float64) []math.Point2 {
	a0 := stdmath.Atan2(float64(tangent.Y), float64(tangent.X))
	a1 := stdmath.Atan2(float64(apex.Y), float64(apex.X))
	for a1-a0 > stdmath.Pi {
		a1 -= 2 * stdmath.Pi
	}
	for a0-a1 > stdmath.Pi {
		a1 += 2 * stdmath.Pi
	}
	k := int(stdmath.Ceil(stdmath.Abs(a1-a0) / (2 * stdmath.Pi / 32)))
	if k < 2 {
		k = 2
	}
	out := make([]math.Point2, 0, k)
	for j := 1; j <= k; j++ {
		a := a0 + (a1-a0)*float64(j)/float64(k)
		out = append(out, math.P2(r*stdmath.Cos(a), r*stdmath.Sin(a)))
	}
	return out
}

// sharedEdgeMidpoint returns the midpoint of an edge shared by faces a and b, if any.
func sharedEdgeMidpoint(a, b *topo.Face) (math.Point3, bool) {
	for _, e := range a.Edges() {
		for _, f := range e.Faces() {
			if f == b {
				return e.StartVertex().Point().Midpoint(e.EndVertex().Point()), true
			}
		}
	}
	return math.Point3{}, false
}

// planeDot is n·(p as a position vector) — the plane-equation constant for normal n through p.
func planeDot(n math.Vector3, p math.Point3) float64 { return float64(n.Dot(p.AsVector())) }

// linearSolve3 solves [rowA;rowB;rowC]·x = [a;b;c] by Cramer's rule, reporting false if singular.
func linearSolve3(rowA, rowB, rowC math.Vector3, a, b, c float64) (math.Point3, bool) {
	m := [3][3]float64{
		{float64(rowA.X), float64(rowA.Y), float64(rowA.Z)},
		{float64(rowB.X), float64(rowB.Y), float64(rowB.Z)},
		{float64(rowC.X), float64(rowC.Y), float64(rowC.Z)},
	}
	det3 := func(mm [3][3]float64) float64 {
		return mm[0][0]*(mm[1][1]*mm[2][2]-mm[1][2]*mm[2][1]) -
			mm[0][1]*(mm[1][0]*mm[2][2]-mm[1][2]*mm[2][0]) +
			mm[0][2]*(mm[1][0]*mm[2][1]-mm[1][1]*mm[2][0])
	}
	det := det3(m)
	if stdmath.Abs(det) < 1e-12 {
		return math.Point3{}, false
	}
	col := func(i int, v [3]float64) [3][3]float64 {
		mm := m
		for r := 0; r < 3; r++ {
			mm[r][i] = v[r]
		}
		return mm
	}
	rhs := [3]float64{a, b, c}
	return math.P3(det3(col(0, rhs))/det, det3(col(1, rhs))/det, det3(col(2, rhs))/det), true
}

// ribFrame is the orthonormal frame + radius of a full round, derived from the three faces.
type ribFrame struct {
	up, sideN, axis math.Vector3
	radius          float64
}

// fullRoundFrame resolves the two side faces and derives the rib frame: up (center normal), sideN
// (side normal), axis (along the rib), and the round radius (half the side-to-side distance). It
// errors when the sides are not parallel, the center is not perpendicular to them, or the radius is 0.
func fullRoundFrame(body *topo.Body, def *FullRoundFilletDefinition, center planarFace, feat string) (ribFrame, error) {
	side1, err := singlePlanarFace(body, def.Side1Keys, feat, "side1")
	if err != nil {
		return ribFrame{}, err
	}
	side2, err := singlePlanarFace(body, def.Side2Keys, feat, "side2")
	if err != nil {
		return ribFrame{}, err
	}
	up, sideN := normalize(center.Normal()), normalize(side1.Normal())
	if d := stdmath.Abs(float64(normalize(side2.Normal()).Dot(sideN))); d < 0.999 {
		return ribFrame{}, fmt.Errorf("%s: the two side faces must be parallel (|n1·n2| = %.3f)", feat, d)
	}
	axis := normalize(up.Cross(sideN))
	if float64(axis.Dot(axis)) < 0.5 {
		return ribFrame{}, fmt.Errorf("%s: the center face must be perpendicular to the sides", feat)
	}
	r := stdmath.Abs(float64(side1.Origin.VectorTo(side2.Origin).Dot(sideN))) / 2
	if r <= 0 {
		return ribFrame{}, fmt.Errorf("%s: the side faces are coincident (zero radius)", feat)
	}
	return ribFrame{up: up, sideN: sideN, axis: axis, radius: r}, nil
}

// fullRoundCornerTool builds the cut tool: a box covering the center-face footprint from the round's
// base plane up to the center face, MINUS the round cylinder — i.e. the two sharp corners the round
// shaves off. pc is the center-face centre; up/sideN/axisDir the orthonormal rib frame; r the radius;
// l the length along the rib. rec collects the box-minus-cylinder boolean's fallback
// diagnostics (#1601; nil discards).
func fullRoundCornerTool(pc math.Point3, up, sideN, axisDir math.Vector3, r, l float64, feat string, rec *diag.Recorder) (*topo.Body, error) {
	plane, err := sketch.NewPlane(pc, sideN.AsUnit(), up.AsUnit())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", feat, err)
	}
	// Box (in the sideN×up plane, centred on pc): full width 2r across the sides, depth r below the
	// centre face, extruded ±l/2 along the rib axis (the plane normal).
	rect := []math.Point2{{X: -r, Y: -r}, {X: r, Y: -r}, {X: r, Y: 0}, {X: -r, Y: 0}}
	box := buildPrism(rect, plane, span{near: -l / 2, far: l / 2}, 0, feat)

	axisPt := pc.TranslateBy(up.Scale(-r)) // the cylinder axis sits r below the centre face, on the midline
	base := axisPt.TranslateBy(axisDir.Scale(-l / 2))
	cyl, err := brep.SolidCylinder(base, axisDir, r, l)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", feat, err)
	}
	corner, err := ops.BooleanWithDiagnostics(ops.Cut, planarizedDiag(box, feat, rec), planarizedDiag(cyl, feat, rec), rec)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", feat, err)
	}
	return corner, nil
}

// planarFace pairs a face with its plane geometry.
type planarFace struct {
	face *topo.Face
	geom.Plane
}

// singlePlanarFace resolves exactly one planar face from keys (a full round takes one face per set).
func singlePlanarFace(body *topo.Body, keys [][]byte, feat, role string) (planarFace, error) {
	want := keyLookup(keys)
	var found *topo.Face
	for _, f := range body.Faces() {
		if want[string(f.ReferenceKey())] {
			if found != nil {
				return planarFace{}, fmt.Errorf("%s: %s must be a single face", feat, role)
			}
			found = f
		}
	}
	if found == nil {
		return planarFace{}, fmt.Errorf("%s: %s face not found", feat, role)
	}
	pl, ok := found.Geometry().(geom.Plane)
	if !ok {
		return planarFace{}, fmt.Errorf("%s: %s must be a planar face", feat, role)
	}
	return planarFace{face: found, Plane: pl}, nil
}

// faceExtentAlong returns the span of the face's vertices projected onto dir, relative to origin.
func faceExtentAlong(f *topo.Face, origin math.Point3, dir math.Vector3) float64 {
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, v := range f.Vertices() {
		d := float64(origin.VectorTo(v.Point()).Dot(dir))
		lo, hi = stdmath.Min(lo, d), stdmath.Max(hi, d)
	}
	if hi < lo {
		return 0
	}
	return hi - lo
}

// normalize returns the unit vector in v's direction (zero stays zero).
func normalize(v math.Vector3) math.Vector3 {
	if l := float64(v.Length()); l > 1e-12 {
		return v.Scale(math.Scalar(1 / l))
	}
	return math.V3(0, 0, 0)
}

// AddFullRoundFillet replaces the center face with a full round between the two side faces (parallel or converging)
// (#694). Each set is a single planar face.
func (c *DressUpFeatures) AddFullRoundFillet(side1Keys, centerKeys, side2Keys [][]byte) *PartFeature {
	return c.engine.Add(&FullRoundFilletFeature{def: &FullRoundFilletDefinition{
		Side1Keys: side1Keys, CenterKeys: centerKeys, Side2Keys: side2Keys,
	}})
}
