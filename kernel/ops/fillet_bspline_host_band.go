// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"
	"os"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Band construction for the B-spline-host canal: arc-length edge anchors → marched
// stations (geom.MarchCanalEdgeStations) → exact-station loft (geom.LoftCanalStations),
// with the station density DOUBLED until the measured mid-interval envelope error is under
// the model-relative bound — the adaptive discipline of the elliptic rim canal
// (resolveEllipticRimStations), generalized to numeric hosts. Every station column is
// exact on the true envelope (marched to |dist−r| ≤ weld on both hosts); the density only
// controls the loft's between-station interpolation.

const (
	// bsplineHostStationsMin/Max bound the adaptive doubling (the elliptic precedent's
	// 32→512 window: reachable cost, honest-reject past the cap).
	bsplineHostStationsMin = 32
	bsplineHostStationsMax = 512
	// bsplineHostEnvelopeCoef scales the model-relative envelope bound: 1e3·res.Weld() ≈
	// 1e-6 of the model — four orders inside the corpus' 1% area gate (the
	// ellipticRimEnvelopeCoef rationale, unchanged).
	bsplineHostEnvelopeCoef = 1e3
	// bsplineHostArcSamples is the interior arc probe count per mid-interval column.
	bsplineHostArcSamples = 3
	// bsplineHostOverrunFrac sizes each open end's prolong span as a fraction of r: the
	// trim curve of a cap oblique up to ~63° stays inside 2r of overrun.
	bsplineHostOverrunFrac = 2.0
)

// buildBsplineHostCanal marches, lofts and certifies the band for one picked edge,
// resolving the station density and the walk orientation. ok=false (fall through to the
// existing refusal) on any march/loft decline or an envelope error still over bound at the
// station cap.
func buildBsplineHostCanal(body *topo.Body, e *topo.Edge, aF, bF *topo.Face, r float64) (*bsplineHostCanal, bool) {
	res := ResolutionForBody(body)
	spec, ok := newBsplineHostMarchSpec(e, aF, bF, r, res)
	if !ok {
		return nil, false
	}
	if spec.closed {
		// CLOSED B-spline-host rims are declined here, not built (Oblikovati#585 regression:
		// TestImportedNurbsDuctVolumeAndFolds). The closed-rim weld's receded wall is a
		// periodic-in-u face whose new rim carries no pcurve, and the ONLY mesher path proven
		// to tessellate that correctly (periodicNurbsFaceMesh's covering CDT, admitted via a
		// "lacks a pcurve" gate) turned out to be UNSAFE in general: measured on the committed
		// bulged_duct fixture, an ordinary imported periodic face — never touched by any
		// fillet — has the SAME "no pcurve on its rim/seam" shape (STEP healing does not
		// attach pcurves to a periodic surface's own natural boundary), so that gate routed a
		// perfectly good import through the covering CDT and cost it −1.18% volume (ceiling
		// 1%). Reverting the gate (periodic_nurbs_mesh.go) fixes the import but leaves this
		// closed-rim weld with no safe tessellation path — measured directly: without the
		// gate, J9 folds 21 edges / 113-289 free edges, B2 folds 62 / 302-698 free edges, both
		// far from watertight. Tessellation correctness preempts feature work (CLAUDE.md), so
		// the honest choice is to decline here — the SAME flat "cannot round an edge bordering
		// a curved face" refusal a closed rim always got before this engine — rather than ship
		// a body that renders wrong. Re-deriving a genuine safe periodic mesh path for an
		// op-rebuilt rim (distinct from "any pcurve-less periodic face") is future work.
		return nil, false
	}
	trial, err := bsplineHostCanalDir(spec, 1, bsplineHostStationsMin, res)
	if err != nil {
		wgBsplineDebug("trial march", err)
		return nil, false
	}
	dir, ok := bsplineHostWalkDirection(trial, spec)
	if !ok {
		return nil, false
	}
	if spec, ok = bsplineHostRetainedCut(spec, trial); !ok {
		return nil, false
	}
	return bsplineHostRefineDensity(spec, dir, res)
}

// bsplineHostRetainedCut resolves an OPEN spec's retained cut window from the trial (PROBE)
// band: the trial doubles as the crossing locator, and the FINAL band is rebuilt over just the
// retained arc window, so the hosts' wild deep-extension feet can never pollute the loft's
// on-edge interpolation (the cubic v-interpolation is quasi-local — a wild prolong station
// bleeds ~3 knot spans inward, which is exactly what the G5 rail-off-host plateau measured). A
// CLOSED spec has no cut to resolve and passes through unchanged (byte-identical).
func bsplineHostRetainedCut(spec bsplineHostMarchSpec, trial *bsplineHostCanal) (bsplineHostMarchSpec, bool) {
	if spec.closed {
		return spec, true
	}
	cut, ok := bsplineHostProbeCut(trial, spec)
	if !ok {
		return spec, false
	}
	spec.cut = cut
	return spec, true
}

// bsplineHostRefineDensity doubles the station density until the measured envelope error is
// under the model-relative bound (or the density cap is reached), returning the first sound,
// resolved band. ok=false (fall through to the existing refusal) on any march/loft decline or an
// envelope error still over bound at the station cap.
func bsplineHostRefineDensity(spec bsplineHostMarchSpec, dir float64, res Resolution) (*bsplineHostCanal, bool) {
	for n := bsplineHostStationsMin; n <= bsplineHostStationsMax; n *= 2 {
		canal, err, over := bsplineHostCanalAt(spec, dir, n, res)
		if err == nil && !over {
			return canal, true
		}
		if err != nil {
			wgBsplineDebug("band decline", err)
			return nil, false // march/loft decline: refusing is the do-no-harm floor
		}
	}
	wgBsplineDebug("envelope over bound at station cap", nil)
	return nil, false // envelope error still over bound at the station cap — honest reject
}

// wgBsplineDebug is a temporary stderr trace of band declines, active only under
// WG_BSPLINE_DEBUG=1 (removed before the wave lands — never a shipped log path).
func wgBsplineDebug(msg string, err error) {
	if !wgBsplineDebugOn() {
		return
	}
	fmt.Fprintf(os.Stderr, "WG_BSPLINE: %s", msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, ": %v", err)
	}
	fmt.Fprintln(os.Stderr)
}

// wgBsplineDebugOn reports whether the temporary WG_BSPLINE_DEBUG trace is armed.
func wgBsplineDebugOn() bool { return os.Getenv("WG_BSPLINE_DEBUG") == "1" }

// bsplineHostMarchSpec is the resolved march input for one edge: hosts, seed, anchors
// source and classification. For an OPEN edge, cap0/cap1 are the capping planes at the
// edge start/end (resolved up front — the two-phase build needs them to place the final
// band's cut span), and cut is the retained arc window the FINAL band spans.
type bsplineHostMarchSpec struct {
	e          *topo.Edge
	aF, bF     *topo.Face
	hostA      geom.CanalMarchHost
	hostB      geom.CanalMarchHost
	r          float64
	concave    bool
	closed     bool
	arcTable   *edgeArcTable
	cap0, cap1 bsplineHostCapPlane
	cut        [2]float64
}

// newBsplineHostMarchSpec packages the march input, declining on an unreadable edge curve,
// a missing capping face, or (for now) a non-planar capping — the I5/I7-class B-spline-
// capped ends are a later tier and keep the byte-identical flat refusal.
func newBsplineHostMarchSpec(e *topo.Edge, aF, bF *topo.Face, r float64, res Resolution) (bsplineHostMarchSpec, bool) {
	tab, ok := newEdgeArcTable(e.Geometry())
	if !ok {
		return bsplineHostMarchSpec{}, false
	}
	weld := res.Weld()
	spec := bsplineHostMarchSpec{
		e: e, aF: aF, bF: bF,
		hostA: bsplineHostMarchHost(aF, weld), hostB: bsplineHostMarchHost(bF, weld),
		r: r, concave: ClassifyEdgeConvexity(e) == EdgeConcave,
		closed: e.StartVertex() == e.EndVertex(), arcTable: tab,
	}
	if spec.closed {
		return spec, true
	}
	return openSpecWithCappings(spec)
}

// openSpecWithCappings resolves an open edge's two capping planes and the initial probe
// cut window (the full edge plus the bsplineHostOverrunFrac·r prolong each side).
func openSpecWithCappings(spec bsplineHostMarchSpec) (bsplineHostMarchSpec, bool) {
	ef := edgeFillet{a: spec.aF, b: spec.bF, edge: spec.e}
	cap0, why0 := newBsplineHostCapPlane(spec.e.StartVertex(), ef)
	if why0 != "" {
		wgBsplineDebug("capping (start): "+why0, nil)
		return bsplineHostMarchSpec{}, false
	}
	cap1, why1 := newBsplineHostCapPlane(spec.e.EndVertex(), ef)
	if why1 != "" {
		wgBsplineDebug("capping (end): "+why1, nil)
		return bsplineHostMarchSpec{}, false
	}
	spec.cap0, spec.cap1 = cap0, cap1
	spec.cut = [2]float64{-bsplineHostOverrunFrac * spec.r, spec.arcTable.length + bsplineHostOverrunFrac*spec.r}
	return spec, true
}

// bsplineHostWalkDirection fixes the anchor walk sense ONCE off the coarse trial band: +1
// keeps the edge's own parameter direction, −1 reverses it so the lofted band's normal
// points out of the solid (the ellipticRimWalkDirection decision, decided before the
// refinement loop so it cannot oscillate).
func bsplineHostWalkDirection(trial *bsplineHostCanal, spec bsplineHostMarchSpec) (float64, bool) {
	flip, ok := bsplineHostBandOutwardFlip(trial.stations, trial.surf, spec.concave)
	if !ok {
		wgBsplineDebug("outward flip undecided", nil)
		return 0, false
	}
	if flip {
		return -1, true
	}
	return 1, true
}

// bsplineHostProbeCutMarginFrac pads the probe's crossing hull (as a fraction of r) so the
// final band strictly contains the trim curves re-solved on its own surface.
const bsplineHostProbeCutMarginFrac = 0.05

// bsplineHostProbeCut locates all four rail∩cap crossings on the PROBE band and returns
// the retained arc window [min−margin, max+margin] the final band will span. ok=false when
// a crossing is missing or ambiguous (the same refusal the end trims would give).
func bsplineHostProbeCut(probe *bsplineHostCanal, spec bsplineHostMarchSpec) ([2]float64, bool) {
	vp := bsplineHostStationV(probe.stations)
	startIsLow := probe.stations[probe.plan.iEdge0].Anchor.DistanceTo(spec.e.StartVertex().Point()) <=
		probe.stations[probe.plan.iEdge1].Anchor.DistanceTo(spec.e.StartVertex().Point())
	capLow, capHigh := spec.cap0, spec.cap1
	if !startIsLow {
		capLow, capHigh = capHigh, capLow
	}
	lo, ok1 := probeEndCrossingArcs(probe, vp, probe.plan.iEdge0, capLow)
	hi, ok2 := probeEndCrossingArcs(probe, vp, probe.plan.iEdge1, capHigh)
	if !ok1 || !ok2 {
		return [2]float64{}, false
	}
	m := bsplineHostProbeCutMarginFrac * spec.r
	return [2]float64{stdmath.Min(lo[0], lo[1]) - m, stdmath.Max(hi[0], hi[1]) + m}, true
}

// probeEndCrossingArcs solves one end's two rail crossings on the probe band and maps
// their v-parameters back to edge arc coordinates.
func probeEndCrossingArcs(probe *bsplineHostCanal, vp []float64, endIdx int, cap bsplineHostCapPlane) ([2]float64, bool) {
	iLo, iHi := bsplineHostEndWindow(probe, endIdx)
	vA, _, whyA := railCapCrossing(probe.surf, 0, vp, iLo, iHi, cap)
	vB, _, whyB := railCapCrossing(probe.surf, 1, vp, iLo, iHi, cap)
	if whyA != "" || whyB != "" {
		wgBsplineDebug("probe crossing: "+whyA+whyB, nil)
		return [2]float64{}, false
	}
	return [2]float64{probeArcAtV(probe, vp, vA), probeArcAtV(probe, vp, vB)}, true
}

// probeArcAtV linearly maps a band v-parameter back to the edge arc coordinate through
// the station ladder.
func probeArcAtV(probe *bsplineHostCanal, vp []float64, v float64) float64 {
	arcs := probe.plan.arcs
	for i := 0; i+1 < len(vp); i++ {
		if v >= vp[i] && v <= vp[i+1] {
			f := 0.0
			if vp[i+1] > vp[i] {
				f = (v - vp[i]) / (vp[i+1] - vp[i])
			}
			return arcs[i] + f*(arcs[i+1]-arcs[i])
		}
	}
	return arcs[len(arcs)-1]
}

// bsplineHostCanalAt builds the band at one station density and measures its envelope
// error: over=true means the band is sound but under-resolved (the caller doubles).
func bsplineHostCanalAt(spec bsplineHostMarchSpec, dir float64, n int, res Resolution) (*bsplineHostCanal, error, bool) {
	canal, err := bsplineHostCanalDir(spec, dir, n, res)
	if err != nil {
		return nil, err, false
	}
	if e := bsplineHostEnvelopeError(canal, spec); e > bsplineHostEnvelopeCoef*res.Weld() {
		wgBsplineDebug(fmt.Sprintf("envelope %.3e > bound %.3e at n=%d", e, bsplineHostEnvelopeCoef*res.Weld(), n), nil)
		return nil, nil, true
	}
	return canal, nil, false
}

// bsplineHostCanalDir marches the stations in the given walk direction and lofts them.
// The march is SEEDED at the edge-start station (never a prolong tip — the hosts'
// polynomial extension there makes the global inversion and the two-plane seed
// unreliable) and continues outward both ways.
func bsplineHostCanalDir(spec bsplineHostMarchSpec, dir float64, n int, res Resolution) (*bsplineHostCanal, error) {
	plan := newBsplineHostAnchorPlan(spec, dir, n)
	c0, ok := bsplineHostSeedCentre(spec.e, spec.aF, spec.bF, plan.anchors[plan.iEdge0].P, spec.r)
	if !ok {
		return nil, errBsplineHostSeed
	}
	stations, err := geom.MarchCanalEdgeStationsSeeded(spec.hostA, spec.hostB, spec.r, plan.anchors, plan.iEdge0, c0, res.Weld())
	if err != nil {
		return nil, err
	}
	if spec.closed {
		if err := snapClosedStationLoop(stations, spec, res.Weld()); err != nil {
			return nil, err
		}
	}
	return loftBsplineHostCanal(spec, stations, plan, res)
}

// errBsplineHostSeed names the seed decline (near-tangent hosts or unreadable normals).
var errBsplineHostSeed = errBspline("bspline-host canal: seed centre undefined (near-tangent hosts or unreadable host normal)")

// errBspline is a tiny local error type so band declines carry a message without fmt
// allocation on the hot doubling loop.
type errBspline string

func (e errBspline) Error() string { return string(e) }

// snapClosedStationLoop verifies the closure station equals station 0 to weld and snaps it
// BY REUSE (byte-identical POSITIONS), so the loft closes with no crack — the closure
// discipline of ellipticRimStationsAt, applied to the marched loop. The snapped station's
// LIFTED foot parameters keep the march's continuation (station 0's parameters shifted one
// period up), so a certificate seed interpolated across the closure never straddles the
// seam with un-lifted values (the T8 +Inf-seed regression).
func snapClosedStationLoop(stations []geom.CanalEdgeStation, spec bsplineHostMarchSpec, weld float64) error {
	last := len(stations) - 1
	if d := float64(stations[last].Center.DistanceTo(stations[0].Center)); d > 10*weld {
		return errBspline("bspline-host canal: closed loop does not close: gap over closure tolerance")
	}
	dirA := stdmath.Copysign(1, stations[last].FootA.U-stations[0].FootA.U)
	dirB := stdmath.Copysign(1, stations[last].FootB.U-stations[0].FootB.U)
	snapped := stations[0]
	snapped.FootA.U += dirA * spec.hostA.PeriodU // 0 on an open host: byte-identical reuse
	snapped.FootB.U += dirB * spec.hostB.PeriodU
	stations[last] = snapped
	return nil
}

// loftBsplineHostCanal lofts the stations and packages the band with its isocurve rails.
func loftBsplineHostCanal(spec bsplineHostMarchSpec, stations []geom.CanalEdgeStation, plan bsplineHostAnchorPlan, res Resolution) (*bsplineHostCanal, error) {
	centers, feetA, feetB := bsplineHostStationRows(stations)
	surf, err := geom.LoftCanalStations(centers, feetA, feetB, spec.r, res.Weld())
	if err != nil {
		return nil, err
	}
	railA, err := geom.SurfaceIsoCurve(surf, true, 0)
	if err != nil {
		return nil, err
	}
	railB, err := geom.SurfaceIsoCurve(surf, true, 1)
	if err != nil {
		return nil, err
	}
	seamMid, ok := bsplineHostSeamMid(stations[0], spec.r)
	if !ok {
		return nil, errBspline("bspline-host canal: station-0 feet antipodal — no seam midpoint")
	}
	return &bsplineHostCanal{
		surf: surf, stations: stations, railA: railA, railB: railB, seamMid: seamMid,
		r: spec.r, concave: spec.concave, closed: spec.closed, plan: plan,
		hostA: spec.aF, hostB: spec.bF,
	}, nil
}

// bsplineHostStationRows splits the stations into the three loft rows.
func bsplineHostStationRows(stations []geom.CanalEdgeStation) (centers, feetA, feetB []math.Point3) {
	centers = make([]math.Point3, len(stations))
	feetA = make([]math.Point3, len(stations))
	feetB = make([]math.Point3, len(stations))
	for i, st := range stations {
		centers[i], feetA[i], feetB[i] = st.Center, st.FootA.P, st.FootB.P
	}
	return centers, feetA, feetB
}

// bsplineHostAnchorPlan is one march's anchor list with the two EDGE-END station indices:
// stations [iEdge0, iEdge1] anchor on the picked edge itself; indices outside are prolong
// stations riding the hosts' natural extension (prolong-then-trim,
// BRepBlend_SurfRstLineBuilder.cxx). arcs holds each anchor's signed arc coordinate
// (0 at the edge start, negative before it) for the end-window searches.
type bsplineHostAnchorPlan struct {
	anchors []geom.CanalEdgeAnchor
	arcs    []float64
	iEdge0  int
	iEdge1  int
}

// newBsplineHostAnchorPlan builds the section anchors for one march: n+1 arc-length-
// uniform on-edge anchors, GEOMETRICALLY refined toward both open ends (the cap-trim
// region needs station spacing far below the uniform step to hold the envelope bound on
// the retained sliver), plus geometric prolong anchors out to bsplineHostOverrunFrac·r.
// A closed edge gets the uniform loop with the seam-tangent closure fix instead.
func newBsplineHostAnchorPlan(spec bsplineHostMarchSpec, dir float64, n int) bsplineHostAnchorPlan {
	if spec.closed {
		return closedAnchorPlan(spec, dir, n)
	}
	arcs := openAnchorArcs(spec.arcTable.length, spec.cut, n)
	return openAnchorPlan(spec, dir, arcs)
}

// closedAnchorPlan is the closed-rim anchor loop: uniform, with the closure station in
// the IDENTICAL section plane as station 0 — one-sided polyline tangents at the seam
// differ by the discretization, which would open an r·Δangle closure gap.
func closedAnchorPlan(spec bsplineHostMarchSpec, dir float64, n int) bsplineHostAnchorPlan {
	on := spec.arcTable.uniformAnchors(n+1, dir)
	seamT := spec.arcTable.closedSeamTangent(dir)
	on[0].T, on[len(on)-1].T = seamT, seamT
	on[len(on)-1].P = on[0].P
	arcs := make([]float64, len(on))
	for k := range arcs {
		arcs[k] = spec.arcTable.length * float64(k) / float64(n)
	}
	return bsplineHostAnchorPlan{anchors: on, arcs: arcs, iEdge0: 0, iEdge1: len(on) - 1}
}

// openAnchorArcs is the sorted signed arc-coordinate ladder of an open edge's anchors
// over the retained cut window: uniform interior + geometric refinement (step halvings
// down to step/64) toward the cut ends AND the edge ends (where the hosts' extension
// kinks the foot rails) + geometric prolong runs out to the exact cut bounds.
func openAnchorArcs(length float64, cut [2]float64, n int) []float64 {
	onLo, onHi := stdmath.Max(0, cut[0]), stdmath.Min(length, cut[1])
	step := (onHi - onLo) / float64(n)
	arcs := make([]float64, 0, n+64)
	for k := 0; k <= n; k++ {
		arcs = append(arcs, onLo+step*float64(k))
	}
	for d := step / 2; d >= step/64; d /= 2 {
		arcs = append(arcs, onLo+d, onHi-d, d, length-d)
	}
	arcs = appendProlongArcs(arcs, cut, length, step)
	arcs = clampArcsToCut(arcs, cut)
	sort.Float64s(arcs)
	return dedupArcs(arcs, step/256)
}

// appendProlongArcs adds the geometric prolong runs past each edge end (fine near the
// end, coarse toward the cut bound) plus the exact cut endpoints.
func appendProlongArcs(arcs []float64, cut [2]float64, length, step float64) []float64 {
	if cut[0] < 0 {
		for d := step / 64; d < -cut[0]; d *= 2 {
			arcs = append(arcs, -d)
		}
		arcs = append(arcs, cut[0])
	}
	if cut[1] > length {
		for d := step / 64; d < cut[1]-length; d *= 2 {
			arcs = append(arcs, length+d)
		}
		arcs = append(arcs, cut[1])
	}
	return arcs
}

// clampArcsToCut drops ladder values outside the retained cut window (an edge-end
// refinement value can overshoot a cut that begins inside the edge).
func clampArcsToCut(arcs []float64, cut [2]float64) []float64 {
	out := arcs[:0]
	for _, a := range arcs {
		if a >= cut[0] && a <= cut[1] {
			out = append(out, a)
		}
	}
	return out
}

// dedupArcs drops near-duplicate arc coordinates (a refinement value landing on a uniform
// station) so no two section planes coincide.
func dedupArcs(arcs []float64, tol float64) []float64 {
	out := arcs[:1]
	for _, a := range arcs[1:] {
		if a-out[len(out)-1] > tol {
			out = append(out, a)
		}
	}
	return out
}

// openAnchorPlan realizes the arc ladder into anchors: on-edge arcs evaluate the arc
// table; prolong arcs extrapolate along the end tangents keeping the END section normal
// (the walking line continues in the last section family). dir=−1 reverses the whole
// march order (and the section normals with it).
func openAnchorPlan(spec bsplineHostMarchSpec, dir float64, arcs []float64) bsplineHostAnchorPlan {
	L := spec.arcTable.length
	p0, t0 := spec.arcTable.at(0)
	p1, t1 := spec.arcTable.at(L)
	plan := bsplineHostAnchorPlan{anchors: make([]geom.CanalEdgeAnchor, len(arcs)), arcs: arcs, iEdge0: -1}
	for k, s := range arcs {
		plan.anchors[k] = openAnchorAt(spec, s, L, p0, t0, p1, t1)
		if s >= 0 && plan.iEdge0 < 0 {
			plan.iEdge0 = k
		}
		if s <= L {
			plan.iEdge1 = k
		}
	}
	if dir < 0 {
		reverseAnchorPlan(&plan)
	}
	return plan
}

// openAnchorAt evaluates one anchor of the open plan: on-edge from the table, prolong by
// end-tangent extrapolation.
func openAnchorAt(spec bsplineHostMarchSpec, s, length float64, p0 math.Point3, t0 math.Vector3, p1 math.Point3, t1 math.Vector3) geom.CanalEdgeAnchor {
	if s < 0 {
		return geom.CanalEdgeAnchor{P: p0.TranslateBy(t0.Scale(math.Scalar(s))), T: t0}
	}
	if s > length {
		return geom.CanalEdgeAnchor{P: p1.TranslateBy(t1.Scale(math.Scalar(s - length))), T: t1}
	}
	p, t := spec.arcTable.at(s)
	return geom.CanalEdgeAnchor{P: p, T: t}
}

// reverseAnchorPlan flips the march order in place: anchors reversed, tangents negated,
// arc coordinates mirrored, edge indices swapped.
func reverseAnchorPlan(plan *bsplineHostAnchorPlan) {
	n := len(plan.anchors)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		plan.anchors[i], plan.anchors[j] = plan.anchors[j], plan.anchors[i]
		plan.arcs[i], plan.arcs[j] = plan.arcs[j], plan.arcs[i]
	}
	for i := range plan.anchors {
		plan.anchors[i].T = plan.anchors[i].T.Scale(-1)
	}
	plan.iEdge0, plan.iEdge1 = n-1-plan.iEdge1, n-1-plan.iEdge0
}
