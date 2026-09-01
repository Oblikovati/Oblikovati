// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The DRAWEXE per-face oracle for the SIX two-boss SETBACK-CLOSE corpus cases — S1 S4 S7 T1 T4 T7, the
// members of the nine formerly-false greens whose run-out patches this engine now skins as the exact
// rolling-ball envelope (runout-envelope-report.md). The other three (S6 S9 T3) are one-boss bodies.
//
// WHY THIS FILE EXISTS AT ALL. The corpus pass/fail gate is 1% of WHOLE-BODY area, and the audit proved
// it is STRUCTURALLY BLIND to this defect class: S7's old Coons fill was 18.5% of r wrong in SHAPE and
// 0.03% RIGHT in per-face area (coons4-audit.md §B), so neither the body gate nor a per-face AREA
// assertion would have fired. Area is therefore pinned here as a cheap first line, and the load-bearing
// claim is TestRunoutPatchInteriorMatchesOCCTSurface: DRAWEXE-evaluated points ON OCCT'S OWN patch
// surface, asserted to lie on OURS. That is the one measurement the false greens could not have passed
// — they sat 0.27–1.29 (9–19% of r) away from exactly these points.
//
// It also complements the certificate: Certificate.MaxBallDev is a SELF-CONSISTENCY measure (the loft
// interpolates stations solved from the same BallEnvelope the residual is taken against, so it bounds
// v-interpolation error between exact stations, and cannot tell you the declared model is the RIGHT
// model). These pins are the external check that it is.
//
// PROVENANCE — every number below is DRAWEXE 8.0.0 output, never hand-derived:
//
//	source test-utilities/occt-blend/oracle/drawenv.sh
//	stepread model/feature/occtparity/fixtures/simple/<CASE>.step a * ; explode a_1 E
//	blend result a_1 <radius> <the edge whose mid-parameter point is `pick`> ; explode result F
//	sprops result_<i> 1.e-9                      → flankArea / centralArea
//	mksurface s result_<i> ; bounds s u0 u1 v0 v1 → cutLo/seamLo/seamHi/cutHi (see below)
//	svalue s <u> <v> x y z                        → flankPts / centralPts
//
// The v-bounds ARE the spine stations: OCCT parametrises each run-out patch surface in v by the fillet
// spine coordinate, so the LEFT flank face spans v∈[cutLo, seamLo], the central v∈[seamLo, seamHi] and
// the RIGHT flank v∈[seamHi, cutHi]. That is an independent confirmation of the seam solve (§1.6 of the
// report) and it is what TestRunoutBandStationsMatchOCCT pins. flankFace/centralFace record which
// exploded face each row was read from, so the run is reproducible verbatim.
//
// The interior samples are the QUINCUNX of the face's own parameter box — (0.25,0.25), (0.25,0.75),
// (0.5,0.5), (0.75,0.25), (0.75,0.75) — all strictly interior, so no sample is a boundary point the
// rails already pin, and the centre sample sits exactly where an interior bulge peaks.
type runoutOracleCase struct {
	name   string
	radius float64
	pick   math.Point3 // the corpus pick's edge midpoint (corpus.json locator)
	// flankFace/centralFace are the DRAWEXE `explode result F` names these values were read off.
	flankFace, centralFace string
	// The four spine stations, read off the oracle patch surfaces' own v-bounds.
	cutLo, seamLo, seamHi, cutHi float64
	// `sprops` areas of the LEFT/RIGHT flank (they are mirror-equal) and the central patch.
	flankArea, centralArea float64
	// `svalue` points on the LEFT flank / central oracle SURFACE, at the interior quincunx.
	flankPts, centralPts []math.Point3
}

// runoutOracleCases is the pinned family. Adding a case means re-running the recipe above, never
// copying a number from a report.
func runoutOracleCases() []runoutOracleCase {
	return []runoutOracleCase{
		{
			name: "S1", radius: 6, pick: math.P3(0, -10, 10),
			flankFace: "result_8", centralFace: "result_3",
			cutLo: 3.071796769724492, seamLo: 6.619065773184741,
			seamHi: 13.380934226815263, cutHi: 16.92820323027551,
			flankArea: 26.5949, centralArea: 34.1915,
			flankPts: []math.Point3{
				math.P3(-6.04138593863995, -9.675633922375637, 6.076488654019747),
				math.P3(-4.267751100388139, -9.786265397824899, 6.26306755589246),
				math.P3(-5.154567928435872, -8.934987922579053, 7.798884657894214),
				math.P3(-6.04138593863995, -7.080705301380742, 9.279134620112607),
				math.P3(-4.267751100388139, -8.076388580424098, 9.07850754977736),
			},
			centralPts: []math.Point3{
				math.P3(-1.690461055326719, -8.673499159464516, 9.14079705943858),
				math.P3(1.690461055685305, -8.673499159422265, 9.140797059425875),
				math.P3(1.79299e-10, -9.386605509136292, 8.193301486392802),
				math.P3(-1.690461055326715, -9.798595463148416, 6.951602069722303),
				math.P3(1.690461055685296, -9.798595463148194, 6.951602069651315),
			},
		},
		{
			name: "S4", radius: 8, pick: math.P3(0, -15, 0),
			flankFace: "result_5", centralFace: "result_13",
			cutLo: 4.045548849896678, seamLo: 10.432773831530113,
			seamHi: 19.56722616845347, cutHi: 25.95445115010332,
			flankArea: 59.9262, centralArea: 54.6813,
			flankPts: []math.Point3{
				math.P3(-9.35764516942435, -14.594830876015035, -5.226032573168202),
				math.P3(-6.164032224619452, -14.767569685026533, -4.736593888298914),
				math.P3(-7.76083878592294, -13.7614367287832, -2.951879177894267),
				math.P3(-9.357645169424343, -11.353477884135332, -1.027979362119546),
				math.P3(-6.164032224619473, -12.908127165238822, -1.257088775328487),
			},
			centralPts: []math.Point3{
				math.P3(-2.283612835072178, -14.824313413617075, -3.830474633609373),
				math.P3(2.283612835085644, -14.824313413616986, -3.830474633611537),
				math.P3(6.732e-12, -14.433345396583501, -2.326658686364771),
				math.P3(-2.28361283507218, -13.695401014296321, -1.131799506335612),
				math.P3(2.283612835085645, -13.695401014294958, -1.131799506336062),
			},
		},
		{
			name: "S7", radius: 3, pick: math.P3(0, -15, 0),
			flankFace: "result_5", centralFace: "result_13",
			cutLo: 9.999999999999995, seamLo: 11.182721649403218,
			seamHi: 18.817278350596787, cutHi: 20.0,
			flankArea: 5.30876, centralArea: 25.7386,
			flankPts: []math.Point3{
				math.P3(-4.7043201911902, -14.798894120145917, -1.917737374375894),
				math.P3(-4.112958148081121, -14.815651143022306, -1.946121109952936),
				math.P3(-4.408639153792017, -14.20101069804246, -0.952681512364669),
				math.P3(-4.7043201911902, -13.190047081313258, -0.243772467924945),
				math.P3(-4.112958148081121, -13.34086028720074, -0.29787684145162),
			},
			centralPts: []math.Point3{
				math.P3(-1.908641388633973, -14.722905157924787, -1.486281682067123),
				math.P3(1.908641388633973, -14.722905157924787, -1.486281682067123),
				math.P3(-1e-15, -14.250494268253775, -0.749504831465283),
				math.P3(-1.908641388633972, -13.591805594786978, -0.307556776250309),
				math.P3(1.908641388633973, -13.591805594786981, -0.30755677625031),
			},
		},
		{
			name: "T1", radius: 8, pick: math.P3(0, -30, 0),
			flankFace: "result_5", centralFace: "result_13",
			cutLo: 18.125657912962076, seamLo: 24.48513002571485,
			seamHi: 35.51486997428515, cutHi: 41.87434208703792,
			flankArea: 71.2647, centralArea: 83.7273,
			flankPts: []math.Point3{
				math.P3(-10.284474087475159, -29.50067672690269, -5.179175583559807),
				math.P3(-7.104737896354747, -29.59074692712838, -5.227875768133281),
				math.P3(-8.694606085735153, -28.144709891530017, -2.746701754234363),
				math.P3(-10.284474087475159, -25.506090542124184, -0.770453041617337),
				math.P3(-7.104737896354745, -26.31672234415541, -1.018431262150157),
			},
			centralPts: []math.Point3{
				math.P3(-2.757434987136578, -29.1571911189425, -3.112391993655386),
				math.P3(2.757434987160388, -29.157191118943686, -3.11239199366397),
				math.P3(1.1909e-11, -27.917646652416323, -1.477941684479601),
				math.P3(-2.757434987136607, -26.481164408667972, -0.686373247562636),
				math.P3(2.75743498716041, -26.481164408667482, -0.686373247564536),
			},
		},
		{
			name: "T4", radius: 8, pick: math.P3(0, -30, 0),
			flankFace: "result_7", centralFace: "result_13",
			cutLo: 18.12565791296209, seamLo: 23.126838631950847,
			seamHi: 36.873161368049146, cutHi: 41.87434208703792,
			flankArea: 57.2326, centralArea: 114.296,
			flankPts: []math.Point3{
				math.P3(-10.624048438529439, -29.48844859443406, -5.160329331113893),
				math.P3(-8.123458294576501, -29.566340040237854, -5.231217916558585),
				math.P3(-9.373749431365821, -28.058423373537433, -2.6885616107517),
				math.P3(-10.624048438529439, -25.396037349906486, -0.731727553337986),
				math.P3(-8.123458294576498, -26.09706036214064, -0.958135942018823),
			},
			centralPts: []math.Point3{
				math.P3(-3.436579661304389, -29.282439259302656, -3.780641624999732),
				math.P3(3.436579661313478, -29.28243925930297, -3.780641625001977),
				math.P3(4.55e-12, -28.08240366655456, -1.9175925423794),
				math.P3(-3.436579661304387, -26.53765900601395, -0.841052993575275),
				math.P3(3.436579661313482, -26.537659006013484, -0.84105299357565),
			},
		},
		{
			name: "T7", radius: 6, pick: math.P3(0, -13, 0),
			flankFace: "result_5", centralFace: "result_13",
			cutLo: 9.287857357185727, seamLo: 13.267291396044953,
			seamHi: 26.73270860395505, cutHi: 30.71214264281428,
			flankArea: 33.2141, centralArea: 64.1795,
			flankPts: []math.Point3{
				math.P3(-9.71728416444977, -12.62768784968941, -3.887438062941772),
				math.P3(-7.727566890072214, -12.70004100700217, -3.917337875312092),
				math.P3(-8.722425759532682, -11.626715818234933, -2.071429784073144),
				math.P3(-9.71728416444977, -9.649190647204692, -0.584620332424966),
				math.P3(-7.727566890072215, -10.300369063019527, -0.779968086620776),
			},
			centralPts: []math.Point3{
				math.P3(-3.366355744734985, -12.39354198179546, -1.852296615087561),
				math.P3(3.366355744734979, -12.393541981795455, -1.852296615087554),
				math.P3(9e-15, -11.653784210895475, -0.769323683656787),
				math.P3(-3.366355744734979, -10.725322556888948, -0.446648536897824),
				math.P3(3.366355744734984, -10.725322556888951, -0.446648536897821),
			},
		},
	}
}

// runoutStationTol is the absolute agreement demanded of a spine station against OCCT's own patch
// v-bounds. The measured spread across the six cases is 3e-13…1.7e-11 (OCCT's two neighbouring faces
// disagree with EACH OTHER by up to 4e-10 at a shared seam), so 1e-9 is ~60x the worst observed
// residual and still 8 orders under the 32%/56% seam-placement defect it replaced.
const runoutStationTol = 1e-9

// TestRunoutBandStationsMatchOCCT pins all four spine stations of every two-boss run-out tiling against
// DRAWEXE. The seam is the sharpest single number in the derivation — it is where the SURF-RST contact
// locus on the inner host reaches the inner boss's footprint, NOT where that footprint crosses the plain
// fillet's contact line, and the tiling used to place it 32% out on S1 and 56% out on S4.
func TestRunoutBandStationsMatchOCCT(t *testing.T) {
	t.Parallel()
	for _, c := range runoutOracleCases() {
		t.Run(c.name, func(t *testing.T) {
			tl, _ := runoutOracleTiling(t, c)
			for _, s := range []struct {
				name      string
				got, want float64
			}{
				{"cutLo", tl.cutLo, c.cutLo}, {"seamLo", tl.seamLo, c.seamLo},
				{"seamHi", tl.seamHi, c.seamHi}, {"cutHi", tl.cutHi, c.cutHi},
			} {
				if d := stdmath.Abs(s.got - s.want); d > runoutStationTol {
					t.Errorf("%s = %.15f, OCCT %.15f (off by %g)", s.name, s.got, s.want, d)
				}
			}
		})
	}
}

// runoutAreaRelTol is the per-face surface-area agreement demanded of a run-out patch. Worst measured
// across the twelve pinned faces is 9.7e-5 (T7's central), against DRAWEXE areas printed to 6 significant
// figures, so this is ~10x the worst observed residual. It is deliberately NOT the primary guard: S7's
// old Coons flank was 0.03% right in area while 18.5% of r wrong in shape.
const runoutAreaRelTol = 1e-3

// TestRunoutPatchSurfaceMatchesOCCTPerFace reconciles the lofted SURFACE (integrated, not meshed — the
// mesh carries its own boundary-chording bias) against OCCT's own per-face areas, for all three patches
// of all six two-boss cases. It is what makes the formerly-false greens honest at the coarse level: the
// Coons fill these bands used to get read 49.49 / 19.92 / 19.92 on S1 against the same 34.19 / 26.59 /
// 26.59, i.e. +44.7% / −25.1%. It also pins the provider KIND — a run-out band that silently fell back
// to a Coons fill would still weld, and that is exactly the regression this whole slice exists to stop.
func TestRunoutPatchSurfaceMatchesOCCTPerFace(t *testing.T) {
	t.Parallel()
	for _, c := range runoutOracleCases() {
		t.Run(c.name, func(t *testing.T) {
			tl, res := runoutOracleTiling(t, c)
			for _, f := range []struct {
				name string
				loop func() (RailLoop, bool)
				want float64
			}{
				{"left flank", tl.leftFlank, c.flankArea},
				{"central", tl.centralBand, c.centralArea},
				{"right flank", tl.rightFlank, c.flankArea},
			} {
				assertPatchArea(t, f.name, runoutOraclePatch(t, f.name, f.loop, res), f.want)
			}
		})
	}
}

// assertPatchArea compares one patch surface's integrated area with its DRAWEXE `sprops` value.
func assertPatchArea(t *testing.T, name string, surf geom.BSplineSurface, want float64) {
	t.Helper()
	got := integrateSurfaceArea(surf)
	if rel := stdmath.Abs(got-want) / want; rel > runoutAreaRelTol {
		t.Errorf("%s: surface area %.4f vs OCCT %.4f (rel %.4f%% > %.2f%%)",
			name, got, want, rel*100, runoutAreaRelTol*100)
	}
}

// runoutInteriorTolFrac is how far a point of OCCT's OWN patch surface may sit from ours, as a fraction
// of the fillet radius. Worst measured across the sixty pinned points is 1.5e-6 of r (S4's central), so
// this leaves ~67x headroom; the Coons fills it replaced sat 9–19% of r out, i.e. 900–1900x OVER it.
const runoutInteriorTolFrac = 1e-4

// TestRunoutPatchInteriorMatchesOCCTSurface is the load-bearing per-face guard, and the one the whole
// corpus gate is blind to. It takes points DRAWEXE evaluated on OCCT's own run-out patch SURFACE, deep
// in its interior, and asserts each lies on ours. Unlike area it cannot be satisfied by a shape error
// that happens to integrate right — which is precisely how S7 certified clean while its flank was 18.5%
// of r away from this surface (coons4-audit.md §B: −0.03% in area, 0.555 absolute in shape).
func TestRunoutPatchInteriorMatchesOCCTSurface(t *testing.T) {
	t.Parallel()
	for _, c := range runoutOracleCases() {
		t.Run(c.name, func(t *testing.T) {
			tl, res := runoutOracleTiling(t, c)
			tol := runoutInteriorTolFrac * c.radius
			assertOraclePointsOnPatch(t, c.name+" left flank ("+c.flankFace+")",
				runoutOraclePatch(t, "left flank", tl.leftFlank, res), c.flankPts, tol)
			assertOraclePointsOnPatch(t, c.name+" central ("+c.centralFace+")",
				runoutOraclePatch(t, "central", tl.centralBand, res), c.centralPts, tol)
		})
	}
}

// assertOraclePointsOnPatch fails when any oracle point sits farther than tol from the patch surface.
func assertOraclePointsOnPatch(t *testing.T, name string, surf geom.BSplineSurface, pts []math.Point3, tol float64) {
	t.Helper()
	for i, p := range pts {
		_, _, foot := geom.ClosestPointOnSurface(surf, p)
		if d := float64(foot.DistanceTo(p)); d > tol {
			t.Errorf("%s: OCCT surface point %d %v sits %g off ours (tol %g)", name, i, p, d, tol)
		}
	}
}

// runoutOraclePatch resolves one band loop through the REAL provider tier and returns its surface,
// failing loud if the band declines or falls back off the run-out canal tier.
func runoutOraclePatch(t *testing.T, name string, build func() (RailLoop, bool), res Resolution) geom.BSplineSurface {
	t.Helper()
	lp, ok := build()
	if !ok {
		t.Fatalf("%s: loop did not build", name)
	}
	patch, ok := resolveBlend(lp, res)
	if !ok {
		t.Fatalf("%s: resolveBlend declined", name)
	}
	if patch.Kind != BlendKindRunoutCanal {
		t.Fatalf("%s: kind %q, want %q (a run-out band must never fall back to a Coons fill)",
			name, patch.Kind, BlendKindRunoutCanal)
	}
	return patch.Surface.(geom.BSplineSurface)
}

// runoutOracleTiling resolves one pinned case's real setback tiling, driving the production path
// (importCorpusSolid → computeEdgeFillet → detectSetbackBands → resolveSetbackTiling) rather than a
// hand-built fixture, for the same reason runoutFixture does: the edgeFillet corner fields are
// interdependent invariants only the real solve satisfies.
func runoutOracleTiling(t *testing.T, c runoutOracleCase) (setbackTiling, Resolution) {
	t.Helper()
	ef, res := runoutOracleFillet(t, c)
	b, ok := detectSetbackBands(ef, res)
	if !ok {
		t.Fatalf("%s: detectSetbackBands declined", c.name)
	}
	tl, ok := resolveSetbackTiling(b, ef, res)
	if !ok {
		t.Fatalf("%s: resolveSetbackTiling declined", c.name)
	}
	return tl, res
}

// runoutOracleFillet imports one pinned case's corpus solid and solves its picked edge's fillet at the
// corpus radius, at the body's OWN model-relative resolution (ADR-0042).
func runoutOracleFillet(t *testing.T, c runoutOracleCase) (edgeFillet, Resolution) {
	t.Helper()
	body := importCorpusSolid(t, "simple/"+c.name)
	e := edgeAtMidpoint(body, c.pick)
	if e == nil {
		t.Fatalf("%s: picked edge (midpoint %v) not found", c.name, c.pick)
	}
	fil, err := computeEdgeFillet(body, filletPick{edge: e, r0: c.radius, r1: c.radius},
		map[uint64]*cornerBlend{}, map[uint64]*cornerMiter{}, FillConcaveOutward)
	if err != nil {
		t.Fatalf("%s: computeEdgeFillet(r=%v): %v", c.name, c.radius, err)
	}
	return fil, ResolutionForBody(body)
}

// integrateSurfaceArea is a midpoint quadrature of |S_u × S_v| over the patch domain — a like-for-like
// comparison with DRAWEXE's `sprops`, which is itself a surface quadrature, and independent of the
// tessellator's boundary chording (the trap n4_cornerweld_layer_test.go documents).
func integrateSurfaceArea(s geom.BSplineSurface) float64 {
	u0, u1 := s.UDomain()
	v0, v1 := s.VDomain()
	const n = 64
	du, dv := (u1-u0)/n, (v1-v0)/n
	sum := 0.0
	for i := range n {
		for j := range n {
			su, sv := s.DerivativesAt(u0+(float64(i)+0.5)*du, v0+(float64(j)+0.5)*dv)
			sum += float64(su.Cross(sv).Length()) * du * dv
		}
	}
	return sum
}
