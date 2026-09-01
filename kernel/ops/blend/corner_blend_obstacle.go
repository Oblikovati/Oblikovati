// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// ObstacleFeature carries the geometry of a mid-span obstacle patch (ADR-4, spec §3): the obstacle's
// rim curve, the two Nodes P± where it crosses the receded fillet boundary, and the four neighbour
// pieces the 4-sided FillSurface must weld to. WingStart/WingEnd are the section arcs of the abutting
// cylinder wings AT the Nodes — reused BY VALUE from the wing faces so the patch is G1 to them by
// identity and no T-junction crack appears. WallLine is the fillet's wall-tangent seam; HostPlane and
// the wing/rim are the neighbour surfaces the certify measures G1/G0 against.
type ObstacleFeature struct {
	RimCurve           geom.Curve3    // obstacle base rim (T6: the base ellipse), full curve
	RimArcPts          []math.Point3  // ordered dip-side rim samples P- -> P+ INCLUSIVE (task 6); source for the c1 rail fit (obstacleRimArc)
	Nodes              [2]math.Point3 // P-, P+ : rim ∩ receded boundary
	WingStart, WingEnd geom.Curve3    // cylinder-wing section arcs at P-, P+ (the shared end rails)
	WallLine           geom.Curve3    // wall-tangent seam between the Nodes' wall points
	HostPlane          geom.Plane     // the notched host face's plane (for the rim-side G0 side)
	// Radius, BlendAxis and WallInto drive the G1 neighbour ribbons the FillSurface matches to (Task 4).
	// Radius is the rolling-ball blend radius (the ribbon length; 0 ⇒ derive from the node span).
	// BlendAxis is the unit fillet-cylinder axis the wings extrude along (±axis, sign per node).
	// WallInto is a unit vector IN the wall plane, ⟂ the bottom rail, pointing AWAY from the host plane
	// into the wall — the direction the wall ribbon extrudes. The detector populates these in Task 6.
	Radius    float64      // rolling-ball blend radius → G1-ribbon length
	BlendAxis math.Vector3 // unit fillet-cylinder axis (wings extrude ±along it)
	WallInto  math.Vector3 // unit, in the wall plane ⟂ the bottom rail, into the wall
	// Canal, when non-nil, is the EXACT surf-rst rolling-ball model of this band (fillet_obstacle_canal.go):
	// the ball tangent to the fillet WALL and passing THROUGH the obstacle rim. It is the only signal
	// obstacleCanalProvider keys on, and the same payload the wall face's own front is subdivided from —
	// so a declined canal takes the wall front back to the straight seam with it, and the two can never
	// describe different geometry for the edge they share.
	Canal *obstacleCanal
}

// bsplineObstacleProvider is the obstacle-variant tier of the corner-blend engine: a single Coons
// FillSurface over the four rails, certified. It Fits only obstacle requests; junction requests fall
// through to the junction providers untouched.
type bsplineObstacleProvider struct{}

// Name reports the provider's telemetry kind (never read by assembly; ADR-2 lineage invariance).
func (bsplineObstacleProvider) Name() CornerBlendKind { return BlendKindBSpline }

// Fits claims only obstacle requests (a non-nil ObstacleFeature); junction requests fall through to
// the junction providers untouched.
func (bsplineObstacleProvider) Fits(req CornerBlendRequest) bool { return req.ObstacleFeature != nil }

// Build fills the obstacle band with a single Coons FillSurface over the four rails and certifies it.
// The wall (c0) and both wings (d0,d1) are G1: each is matched to a purpose-built neighbour RIBBON
// (extrudeRibbon) whose rail edge is the fill's side exactly — true G1 with no fitting. The rim (c1)
// is G0 (sharp base-rim crease). A missing wing / failed rail / failed fill ⇒ ok=false so the tier
// moves on and the caller honest-rejects (ADR-3) rather than ship a crack-inducing patch.
func (bsplineObstacleProvider) Build(req CornerBlendRequest) (CornerBlendPatch, Certificate, bool) {
	of := req.ObstacleFeature
	g, ok := obstaclePatchNeighbours(of)
	if !ok {
		return CornerBlendPatch{}, Certificate{}, false
	}
	surf, err := geom.FillSurface(g.c0, g.c1, g.d0, g.d1, obstacleSides(of, g.wingL, g.wingR, g.wall))
	if err != nil {
		return CornerBlendPatch{}, Certificate{}, false
	}
	surf, err = pinFillBoundary(surf, g.c0, g.c1, g.d0, g.d1)
	if err != nil {
		return CornerBlendPatch{}, Certificate{}, false
	}
	surf = relaxCornerInterior(surf)
	cert := certifyObstaclePatch(surf, g, req.Setback)
	return CornerBlendPatch{Surface: surf, Loops: obstaclePatchLoops(of), Kind: BlendKindBSpline}, cert, true
}

// pinFillBoundary overwrites the fill's four boundary control rows/columns (and their weights) with
// the EXACT rail control nets, undoing the corner drift that FillSurface's sequential MatchSurface
// leaves on the BOUNDARY while keeping the G1-matched INTERIOR rows. A single-side match preserves its
// own position row, so only the shared wall∩wing corner cells drifted (a later adjacent match rewrote
// them); this restores exact G0 → watertight → MaxDev≈0. The four corner control points must still
// agree across the two rails meeting there — Task 3 pinned the rail corners, so they do. It works on a
// DEEP COPY of the net (copyNet/copyWeights) so the caller's fill value is never mutated in place.
func pinFillBoundary(fill geom.BSplineSurface, c0, c1, d0, d1 geom.BSplineCurve) (geom.BSplineSurface, error) {
	ctrl, w := copyNet(fill.Ctrl), copyWeights(fill.Weights)
	nu, nv := len(ctrl), len(ctrl[0])
	for i := range nu {
		ctrl[i][0], w[i][0] = c0.Ctrl[i], c0.Weights[i]       // VMin edge ← c0 (wall)
		ctrl[i][nv-1], w[i][nv-1] = c1.Ctrl[i], c1.Weights[i] // VMax edge ← c1 (rim)
	}
	for j := range nv {
		ctrl[0][j], w[0][j] = d0.Ctrl[j], d0.Weights[j]       // UMin edge ← d0 (wingL)
		ctrl[nu-1][j], w[nu-1][j] = d1.Ctrl[j], d1.Weights[j] // UMax edge ← d1 (wingR)
	}
	return geom.NewBSplineSurface(fill.UDegree, fill.VDegree, ctrl, w, fill.UKnots, fill.VKnots)
}

// copyNet deep-copies a control net so a mutation never aliases the source surface's backing arrays.
func copyNet(src [][]math.Point3) [][]math.Point3 {
	out := make([][]math.Point3, len(src))
	for i := range src {
		out[i] = append([]math.Point3(nil), src[i]...)
	}
	return out
}

// copyWeights deep-copies a weight net (see copyNet).
func copyWeights(src [][]float64) [][]float64 {
	out := make([][]float64, len(src))
	for i := range src {
		out[i] = append([]float64(nil), src[i]...)
	}
	return out
}

// relaxCornerInterior is the optional best-effort G1 corner touch-up (spec Option 1 step 2): a light
// interior nudge at the two wall∩wing corners A(u0,v0) and D(u1,v0), moving ONLY corner-adjacent
// INTERIOR control points (never a boundary row). Measurement (task-4-report) shows the pinned patch
// already meets the interior G1 gate, so this is a no-op today; it stays as the documented seam for a
// future tightening rather than a hidden magic step.
func relaxCornerInterior(fill geom.BSplineSurface) geom.BSplineSurface {
	return fill
}

// obstaclePatchGeom bundles the four boundary rails and the three G1 neighbour ribbons — the shared
// input to both the FillSurface (Build) and the G0/G1 measurements (certify), built once so the two
// never drift apart.
type obstaclePatchGeom struct {
	c0, c1, d0, d1     geom.BSplineCurve
	wall, wingL, wingR geom.BSplineSurface
}

// obstaclePatchNeighbours builds the rails (obstacleRails) and the three G1 ribbons: the wall extrudes
// along WallInto, each wing along ±BlendAxis (sign chosen per node so the ribbon runs INTO its wing,
// away from the obstacle band). ok=false on a bad rail or a rejected ribbon net → honest-reject.
func obstaclePatchNeighbours(of *ObstacleFeature) (obstaclePatchGeom, bool) {
	c0, c1, d0, d1, ok := obstacleRails(of)
	if !ok {
		return obstaclePatchGeom{}, false
	}
	c0, c1, d0, d1, ok = refineForG1(c0, c1, d0, d1)
	if !ok {
		return obstaclePatchGeom{}, false
	}
	base, err := geom.CoonsFill(c0, c1, d0, d1)
	if err != nil {
		return obstaclePatchGeom{}, false
	}
	length := ribbonLength(of)
	wall, e0 := extrudeRibbon(c0, orientInward(of.WallInto.Scale(length), inwardCrossV(base, false).Scale(-1)))
	wingL, e1 := extrudeRibbon(d0, orientInward(wingDir(of, of.Nodes[0], length), inwardCrossU(base, false).Scale(-1)))
	wingR, e2 := extrudeRibbon(d1, orientInward(wingDir(of, of.Nodes[1], length), inwardCrossU(base, true).Scale(-1)))
	if e0 != nil || e1 != nil || e2 != nil {
		return obstaclePatchGeom{}, false
	}
	return obstaclePatchGeom{c0: c0, c1: c1, d0: d0, d1: d1, wall: wall, wingL: wingL, wingR: wingR}, true
}

// orientInward flips dir to agree with ref, the seam's OUTWARD cross-derivative (the negated
// plain-Coons inward cross-derivative, matching coons4Sides — F2 fix, f2-reconciliation-report.md,
// task-2-brief.md; the function name predates the flip and is kept to minimize the diff, but every
// call site now anchors outward, not inward). A G1 tangent ribbon fixes the seam's tangent PLANE
// regardless of the extrude SIGN, but the sign sets whether MatchSurface's glued cross-derivative
// lands inside or outside the patch: MatchSurface negates the ribbon derivative across the seam
// (geom/match_surface.go), so an OUTWARD-anchored ribbon lands the fill's cross-derivative back INSIDE
// the patch, while an inward-anchored one (the original bug) forces the surface to fold back one
// station in. The original inward anchor passed the T6 corpus only because antipodal-blind
// creaseAngle + 24×24 sampling missed the fold; ribbonSeamNonFolding (Task 1's boundary-exact probe)
// catches it and TestObstacleT6RibbonNonFolding is the regression witness. Anchoring to the plain
// Coons interior derivative (rather than WallInto/BlendAxis directly) makes the sign robust to
// whatever the detector supplies for those fields (only their LINE — the tangent-plane info — is
// load-bearing).
func orientInward(dir, ref math.Vector3) math.Vector3 {
	if dir.Dot(ref) < 0 {
		return dir.Scale(-1)
	}
	return dir
}

// inwardCrossV returns the plain-Coons cross-derivative pointing INTO the patch from a v-edge: +∂/∂v
// at v=vmin, −∂/∂v at v=vmax, sampled at the edge midpoint.
func inwardCrossV(s geom.BSplineSurface, atMax bool) math.Vector3 {
	u0, u1 := s.UDomain()
	v0, v1 := s.VDomain()
	v, sign := v0, 1.0
	if atMax {
		v, sign = v1, -1.0
	}
	_, dv := s.DerivativesAt((u0+u1)/2, v)
	return dv.Scale(sign)
}

// inwardCrossU returns the plain-Coons cross-derivative pointing INTO the patch from a u-edge: +∂/∂u
// at u=umin, −∂/∂u at u=umax, sampled at the edge midpoint.
func inwardCrossU(s geom.BSplineSurface, atMax bool) math.Vector3 {
	u0, u1 := s.UDomain()
	v0, v1 := s.VDomain()
	u, sign := u0, 1.0
	if atMax {
		u, sign = u1, -1.0
	}
	du, _ := s.DerivativesAt(u, (v0+v1)/2)
	return du.Scale(sign)
}

// refineForG1 inserts interior knots into the four rails (the SAME knots on each compatible pair, so
// c0/c1 and d0/d1 stay knot-compatible) so the fill carries ~obstacleG1Ctrl control columns per
// direction. FillSurface's sequential MatchSurface corrupts only the TWO control columns at each
// corner; on the raw degree-3, 8-point net those columns' basis bleeds to mid-span, but refined they
// occupy a small parametric window (< obstacleCornerExcl), leaving a genuinely G1 interior once the
// corner window is excluded (certify). Refinement is geometry-exact (knot insertion), so it changes
// neither the boundary nor the patch shape/area — only the DOF available to localize the corner
// clobber. The value obstacleG1Ctrl=28 and the window 0.15 are measured together (task-4-report):
// interior crease drops to ~0 (wall) / ~1.5e-8 (wings), below tessellate.SeamAngularTol=1e-6.
func refineForG1(c0, c1, d0, d1 geom.BSplineCurve) (rc0, rc1, rd0, rd1 geom.BSplineCurve, ok bool) {
	uk := refinementKnots(c0, obstacleG1Ctrl)
	vk := refinementKnots(d0, obstacleG1Ctrl)
	rc0, e0 := c0.RefineKnots(uk)
	rc1, e1 := c1.RefineKnots(uk)
	rd0, e2 := d0.RefineKnots(vk)
	rd1, e3 := d1.RefineKnots(vk)
	return rc0, rc1, rd0, rd1, e0 == nil && e1 == nil && e2 == nil && e3 == nil
}

// obstacleG1Ctrl is the target control-column count per direction after refinement — chosen so the
// corner-clobbered 2-column bands sit inside obstacleCornerExcl with margin (measured, task-4-report).
const obstacleG1Ctrl = 28

// refinementKnots returns up to (target − current) uniform interior parameters to insert into c. They
// are strictly interior and distinct; a chance coincidence with an existing single-multiplicity knot
// merely raises it to 2 (≤ degree 3), so knot insertion stays valid.
func refinementKnots(c geom.BSplineCurve, target int) []float64 {
	lo, hi := c.Domain()
	need := target - len(c.Ctrl)
	if need <= 0 {
		return nil
	}
	us := make([]float64, need)
	for i := 1; i <= need; i++ {
		us[i-1] = lo + float64(i)/float64(need+1)*(hi-lo)
	}
	return us
}

// extrudeRibbon builds a degree-(p,1) B-spline whose VMinEdge IS rail and whose second row is rail
// translated by dir — a first-order-correct ribbon into the neighbour surface. For the WALL (planar)
// and the WING (a cylinder whose axis is dir) this ruled strip equals the neighbour to the order
// MatchSurface reads, so FillSurface achieves true G1 with no fitting, and the ribbon's VMinEdge has
// the same control-column count as the fill's side by construction (MatchSurface's precondition).
func extrudeRibbon(rail geom.BSplineCurve, dir math.Vector3) (geom.BSplineSurface, error) {
	n := len(rail.Ctrl)
	ctrl := make([][]math.Point3, n)
	w := make([][]float64, n)
	for i := range rail.Ctrl {
		ctrl[i] = []math.Point3{rail.Ctrl[i], rail.Ctrl[i].TranslateBy(dir)}
		w[i] = []float64{rail.Weights[i], rail.Weights[i]}
	}
	return geom.NewBSplineSurface(rail.Degree, 1, ctrl, w, rail.Knots, []float64{0, 0, 1, 1})
}

// wingDir returns BlendAxis scaled to ±length: the sign is sign((node − bandCenter)·BlendAxis) so the
// ribbon extrudes into the wing that node belongs to (away from the obstacle band), independent of
// which Node is P- vs P+.
func wingDir(of *ObstacleFeature, node math.Point3, length float64) math.Vector3 {
	center := of.Nodes[0].Lerp(of.Nodes[1], 0.5)
	if center.VectorTo(node).Dot(of.BlendAxis) < 0 {
		return of.BlendAxis.Scale(-length)
	}
	return of.BlendAxis.Scale(length)
}

// ribbonLength is the model-relative ribbon length: the blend Radius when set, else a small fraction
// of the node span (ribbon length only affects first-order matching, so a modest value suffices).
func ribbonLength(of *ObstacleFeature) float64 {
	if of.Radius > 0 {
		return of.Radius
	}
	return of.Nodes[0].DistanceTo(of.Nodes[1]) * ribbonSpanFactor
}

// ribbonSpanFactor is a unitless ratio (not a length epsilon): the fallback ribbon length as a
// fraction of the node span, so it scales with the model (ADR-0042).
const ribbonSpanFactor = 0.1

// obstaclePatchLoops builds ONE geometric boundary loop tracing the four rails in order
// (A→D→P+→P-). Task 6 handles the topo weld and per-segment curve identity; a straight-sampled ring
// of the boundary points is enough here.
func obstaclePatchLoops(of *ObstacleFeature) []filletLoop {
	c0, c1, d0, d1, ok := obstacleRails(of)
	if !ok {
		return nil
	}
	pts := boundaryRing(c0, d1, c1, d0)
	return []filletLoop{{pts: pts, curves: make([]geom.Curve3, len(pts))}}
}

// boundaryRing walks A→D (c0) → P+ (d1) → P- (c1 reversed) → A (d0 reversed), sampling each rail open
// (excluding its shared end corner) so the concatenated ring carries no duplicate points.
func boundaryRing(c0, d1, c1, d0 geom.BSplineCurve) []math.Point3 {
	pts := sampleRailOpen(c0, false)
	pts = append(pts, sampleRailOpen(d1, false)...)
	pts = append(pts, sampleRailOpen(c1, true)...)
	return append(pts, sampleRailOpen(d0, true)...)
}

// sampleRailOpen returns ringSegSamples points along c (reversed if rev), EXCLUDING the far endpoint
// so segments concatenate without duplicating the shared corner. It delegates to the generic
// sampleCurve3Open (corner_provider_sphere.go) — a geom.BSplineCurve satisfies geom.Curve3 — so the
// BSpline and Curve3 boundary-ring samplers share one implementation (Task-3 de-dup).
func sampleRailOpen(c geom.BSplineCurve, rev bool) []math.Point3 {
	return sampleCurve3Open(c, rev)
}

// ringSegSamples is the per-rail sample count of the placeholder boundary loop (Task 6 refines it).
const ringSegSamples = 6
